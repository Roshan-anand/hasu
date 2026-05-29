package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Roshan-anand/godploy/internal/config"
	"github.com/Roshan-anand/godploy/internal/db"
	deploymentqueue "github.com/Roshan-anand/godploy/internal/jobs/deployment/queue"
	"github.com/Roshan-anand/godploy/internal/lib/auth"
	ghservice "github.com/Roshan-anand/godploy/internal/lib/gh"
	"github.com/Roshan-anand/godploy/internal/lib/security"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/go-playground/validator/v10"
	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

type GitHandler struct {
	Server   *config.Server
	Validate *validator.Validate
	qCtx     context.Context
	ghCtx    context.Context
}

type GitHubCreateAppRes struct {
	ID            int64  `json:"id"`
	Slug          string `json:"slug"`
	WebhookSecret string `json:"webhook_secret"`
	PEM           string `json:"pem"`
	Name          string `json:"name"`
}

type GetGithubAppReq struct {
	AppID int64 `json:"app_id" validate:"required"`
}

type GetGithubAppRes struct {
	Name        string    `json:"name"`
	CreatedAt   string    `json:"created_at"`
	GithubAppID uuid.UUID `json:"github_app_id"`
}

type DeleteGithubAppReq struct {
	AppID int64 `json:"app_id" validate:"required"`
}

type GetGithubRepoListRes struct {
	ID            int64    `json:"id"`
	Name          string   `json:"name"`
	FullName      string   `json:"full_name"`
	Private       bool     `json:"private"`
	DefaultBranch string   `json:"default_branch"`
	Branches      []string `json:"branches"`
	HtmlURL       string   `json:"html_url"`
	RepoURL       string   `json:"repo_url"`
}

func InitGitHandlers(s *config.Server) *GitHandler {
	return &GitHandler{
		Server:   s,
		Validate: validator.New(),
		qCtx:     context.Background(),
		ghCtx:    context.Background(),
	}
}

// initiate github app creation
//
// route: GET /api/provider/github/app/create
func (h *GitHandler) CreateGithubApp(c *echo.Context) error {
	q := h.Server.DB.Queries
	u := c.Get(h.Server.Config.EchoCtxUserKey).(auth.AuthUser)

	state, err := security.GenerateCSRFToken()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to create github app"})
	}

	user, err := q.GetUserByEmail(h.qCtx, u.Email)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to create github app"})
	}

	if err := q.CreateRedirectSession(h.qCtx, db.CreateRedirectSessionParams{
		State:     state,
		OrgID:     user.OrgID,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to create github app"})
	}

	// TODO : start a worker which waits for 1hr and check if github app is not filled else remove it.
	// modify the removeSession

	manifest, err := getManifestData(h.Server.Config.ServerUrl, state)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to create github app"})
	}

	tmpl, err := template.New("manifest").Parse(githubManifestFormTmpl)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to create github app"})
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, map[string]string{
		"State":    state,
		"Manifest": manifest,
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to create github app"})
	}

	return c.HTML(http.StatusOK, buf.String())
}

// get github app credentials from GitHub
//
// route: GET /api/provider/github/app/callback
func (h *GitHandler) CreateGithubAppCallback(c *echo.Context) error {
	q := h.Server.DB.Queries
	// u := c.Get(h.Server.Config.EchoCtxUserKey).(auth.AuthUser)

	code := c.QueryParam("code")
	state := c.QueryParam("state")

	// validate the state
	sData, err := q.GetRedirectSession(h.qCtx, state)
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "Invalid state"})
	}

	if time.Now().After(sData.ExpiresAt) {
		go removeSession(q, state)
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "State has expired"})
	}

	conversionURL := fmt.Sprintf("https://api.github.com/app-manifests/%s/conversions", code)
	req, err := http.NewRequest("POST", conversionURL, nil)
	if err != nil {
		return c.Redirect(http.StatusFound, "/?github_error=internal")
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return c.Redirect(http.StatusFound, "/?github_error=github_api_error")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return c.Redirect(http.StatusFound, "/?github_error=code_invalid")
	}

	var convRes GitHubCreateAppRes
	if err := json.NewDecoder(resp.Body).Decode(&convRes); err != nil {
		return c.Redirect(http.StatusFound, "/?github_error=github_api_error")
	}

	// encrypt PEM
	encryptedPem, err := security.EncryptPEM(convRes.PEM)
	if err != nil {
		return c.Redirect(http.StatusFound, "/?github_error=internal")
	}

	// store the app credentials in db
	ghAppId, err := q.CreateGithubApp(h.qCtx, db.CreateGithubAppParams{
		ID:             security.GeneratePrimaryKey(),
		Name:           convRes.Name,
		AppID:          convRes.ID,
		OrganizationID: sData.OrgID,
		WebhookSecret:  convRes.WebhookSecret,
		PemKey:         encryptedPem,
	})
	if err != nil {
		return c.Redirect(http.StatusFound, "/?github_error=internal")
	}

	// update the session with github app id
	if err := q.UpdateRedirectSession(h.qCtx, db.UpdateRedirectSessionParams{
		GhAppID: sql.NullInt64{
			Int64: ghAppId,
			Valid: true,
		},
		State: state,
	}); err != nil {
		return c.Redirect(http.StatusFound, "/?github_error=internal")
	}
	// go removeSession(query, state)

	installUrl := fmt.Sprintf("https://github.com/apps/%s/installations/new", convRes.Slug)
	return c.Redirect(http.StatusFound, installUrl)
}

// installing github app
//
// route: GET /api/provider/github/app/setup
func (h *GitHandler) SetupGithubApp(c *echo.Context) error {
	q := h.Server.DB.Queries

	state := c.QueryParam("state")
	ghAppId, err := q.GetRedirectSessionGhAppID(h.qCtx, state)
	if err != nil || !ghAppId.Valid {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "Invalid state"})
	}
	go removeSession(q, state)

	instllation_id, err := strconv.ParseInt(c.QueryParam("installation_id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "Invalid installation ID"})
	}

	// get app client
	appClient, err := ghservice.NewGithubAppClient(q, ghAppId.Int64)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to setup github app"})
	}

	// verify installation ID by making an authenticated request to GitHub API
	_, _, err = appClient.Apps.GetInstallation(context.Background(), instllation_id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "Invalid installation ID"})
	}

	if err := q.InsertInstallationID(h.qCtx, db.InsertInstallationIDParams{
		InstallationID: sql.NullInt64{
			Int64: instllation_id,
			Valid: true,
		},
		AppID: ghAppId.Int64,
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to setup github app"})
	}

	// TODO: update the url to route to git provider page with success message
	return c.Redirect(http.StatusFound, h.Server.Config.WebUrl+"/#/git")
}

// get all the github app info
//
// route: GET /api/provider/github/app/list
func (h *GitHandler) GetAllGithubApps(c *echo.Context) error {
	u := c.Get(h.Server.Config.EchoCtxUserKey).(auth.AuthUser)
	q := h.Server.DB.Queries

	ghApps, err := q.GetAllGhAppsByEmail(h.qCtx, u.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusOK, types.Res[any]{
				Message: "",
				Data:    nil,
			})
		}
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to get github app"})
	}

	return c.JSON(http.StatusOK, types.Res[[]db.GetAllGhAppsByEmailRow]{
		Message: "",
		Data:    ghApps,
	})
}

// delete github app for admin users
//
// route: DELETE /api/provider/github/app
func (h *GitHandler) DeleteGithubApp(c *echo.Context) error {
	u := c.Get(h.Server.Config.EchoCtxUserKey).(auth.AuthUser)
	b := new(DeleteGithubAppReq)

	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	q := h.Server.DB.Queries
	if isAdmin, err := q.IsUserAdmin(h.qCtx, u.Email); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "internal server error"})
	} else if !isAdmin {
		return c.JSON(http.StatusForbidden, types.Res[struct{}]{Message: "admin access required"})
	}

	appClient, err := ghservice.NewGithubAppClient(q, b.AppID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to delete github app"})
	}

	_, err = appClient.Apps.DeleteInstallation(h.qCtx, b.AppID)
	if err != nil {
		fmt.Println("Error deleting github app installation:", err)
	}

	if err := q.DeleteGithubApp(h.qCtx, b.AppID); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to delete github app"})
	}

	return c.JSON(http.StatusOK, types.Res[int64]{Message: "Github app deleted successfully", Data: b.AppID})
}

// get list of repos accessible by the github app
//
// route: GET /api/provider/github/repo/list?app_id=
func (h *GitHandler) GetGithubRepoList(c *echo.Context) error {
	q := h.Server.DB.Queries

	appID, err := strconv.ParseInt(c.QueryParam("app_id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "Invalid app_id"})
	}

	// create a new github client
	gh, err := ghservice.New(q, appID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to get github repos"})
	}

	opts := &github.ListOptions{
		PerPage: 100,
		Page:    1,
	}

	repos := make([]GetGithubRepoListRes, 0)

	for {
		pageRepos, resp, err := gh.Client.Apps.ListRepos(h.ghCtx, opts)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to get github repos"})
		}

		for _, repo := range pageRepos.Repositories {
			owner := repo.GetOwner().GetLogin()
			repoName := repo.GetName()
			if owner == "" || repoName == "" {
				continue
			}

			// Fetch branches per repo so the UI can offer an explicit deploy branch.
			branches := make([]string, 0)
			branchOpts := &github.BranchListOptions{
				ListOptions: github.ListOptions{
					PerPage: 100,
					Page:    1,
				},
			}
			for {
				pageBranches, branchResp, err := gh.Client.Repositories.ListBranches(h.ghCtx, owner, repoName, branchOpts)
				if err != nil {
					return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to get github branches"})
				}

				for _, branch := range pageBranches {
					branchName := branch.GetName()
					if branchName == "" {
						continue
					}
					branches = append(branches, branchName)
				}

				if branchResp.NextPage == 0 {
					break
				}
				branchOpts.Page = branchResp.NextPage
			}

			repos = append(repos, GetGithubRepoListRes{
				ID:            repo.GetID(),
				Name:          repo.GetName(),
				FullName:      repo.GetFullName(),
				Private:       repo.GetPrivate(),
				DefaultBranch: repo.GetDefaultBranch(),
				Branches:      branches,
				HtmlURL:       repo.GetHTMLURL(),
				RepoURL:       repo.GetCloneURL(),
			})
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	if len(repos) == 0 {
		return c.NoContent(http.StatusNoContent)
	}

	return c.JSON(http.StatusOK, types.Res[[]GetGithubRepoListRes]{
		Message: "",
		Data:    repos,
	})
}

// github webhook handler
//
// route: POST /api/provider/github/webhook
func (h *GitHandler) GithubWebhook(c *echo.Context) error {
	q := h.Server.DB.Queries
	req := c.Request()
	sign := req.Header.Get("X-Hub-Signature-256")
	if sign == "" {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "Missing signature"})
	}

	appIDStr := req.Header.Get("X-GitHub-Hook-Installation-Target-ID")
	if appIDStr == "" {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "Missing webhook target id"})
	}

	appID, err := strconv.ParseInt(appIDStr, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "Invalid webhook target id"})
	}

	ghApp, err := q.GetGhAppByAppId(h.qCtx, appID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusUnauthorized, types.Res[struct{}]{Message: "Unknown webhook target"})
		}
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to process webhook"})
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "Invalid request body"})
	}

	// Validate signature using webhook secret from DB and constant-time compare.
	mac := hmac.New(sha256.New, []byte(ghApp.WebhookSecret))
	mac.Write(body)
	expectedSign := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expectedSign), []byte(sign)) {
		return c.JSON(http.StatusUnauthorized, types.Res[struct{}]{Message: "Invalid signature"})
	}

	eventType := req.Header.Get("X-GitHub-Event")
	if eventType == "" {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "Missing event type"})
	}

	event, err := github.ParseWebHook(eventType, body)
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "Invalid webhook payload"})
	}

	if eventType == "push" {
		pushEvent, ok := event.(*github.PushEvent)
		if !ok {
			return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "Invalid push event payload"})
		}

		repo := pushEvent.GetRepo()
		branch := strings.TrimPrefix(pushEvent.GetRef(), "refs/heads/")

		services, err := q.GetAllAppServicesByRepo(h.qCtx, db.GetAllAppServicesByRepoParams{
			GhRepoID:   repo.GetID(),
			BranchName: branch,
		})
		if err != nil {
			return nil
		}

		// TODO : make downgraddeployment, creatdeployment, generting gh token, unmarshiling env actions inside the worker as it is a redundant process
		for _, s := range services {
			fmt.Println("starting webhook job for :", s.Name, s.BranchName)
			// TODO : check if watch path matches the pushed code commit

			// start a new db transaction
			tx, err := h.Server.DB.Pool.BeginTx(context.Background(), nil)
			if err != nil {
				fmt.Println("Error starting transaction:", err)
				return nil
			}
			tq := q.WithTx(tx)

			var newStatus types.DeploymentStatus
			if s.DeploymentStatus == types.DeploymentReady {
				newStatus = types.DeploymentInactive
			} else {
				newStatus = types.DeploymentPruned
			}

			// update the previous deployment is_latest to false
			if err := tq.DownGradeDeployment(h.qCtx, db.DownGradeDeploymentParams{
				DeploymentID: s.DeploymentID,
				Status:       newStatus,
			}); err != nil {
				tx.Rollback()
				fmt.Println("Error downgrading previous deployment:", err)
				return nil
			}

			// create a new deployment
			dID, err := tq.CreateDeployment(h.qCtx, db.CreateDeploymentParams{
				ID:         security.GeneratePrimaryKey(),
				BranchID:   s.BranchID,
				CommitHash: pushEvent.GetAfter(),
				CommitMsg:  pushEvent.GetHeadCommit().GetMessage(),
				IsCurrent:  true,
			})
			if err != nil {
				tx.Rollback()
				fmt.Println("Error creating deployment:", err)
				return nil
			}

			// create new github client
			gh, err := ghservice.New(q, s.GhAppID)
			if err != nil {
				tx.Rollback()
				fmt.Println("Error creating github client:", err)
				return nil
			}

			// used as unique image and service name
			unique := generateServiceAndImgName(s.Name, s.BranchName)

			envStr, err := UnmarshalServiceEnv(&ServiceEnvByte{
				Env:          s.Env,
				BuildArgs:    s.BuildArgs,
				BuildSecrets: s.BuildSecrets,
			})
			if err != nil {
				fmt.Println("Error unmarshaling service env:", err)
				return nil
			}

			if err := tx.Commit(); err != nil {
				fmt.Println("Error committing transaction:", err)
				return nil
			}

			// push a new deployment job to the queue
			h.Server.DeploymentQ.EnqueuePullJob(&deploymentqueue.PullJobData{
				Type:              deploymentqueue.RebuildJob,
				DeploymentID:      dID,
				Token:             gh.Token,
				Url:               s.GhRepoUrl,
				Branch:            s.BranchName,
				SwarmServiceName:  s.SwarmServiceName,
				BuildPath:         s.BuildPath,
				DockerFilePath:    s.DockerFilepath,
				DockerContextPath: s.DockerContextpath,
				DockerBuildStage:  s.DockerBuildstage,
				ImgName:           unique.ImgName,
				Env:               envStr.Env,
				BuildArgs:         envStr.BuildArgs,
				BuildSecrets:      envStr.BuildSecrets,
			})

		}
	}

	return nil
}
