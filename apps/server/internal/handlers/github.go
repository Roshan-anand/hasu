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
	deployjob "github.com/Roshan-anand/godploy/internal/jobs/deployment"
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

type PRInfo struct {
	ID         int64  `json:"id"`
	Number     int    `json:"number"`
	Title      string `json:"title"`
	State      string `json:"state"`
	HtmlURL    string `json:"html_url"`
	HeadBranch string `json:"head_branch"`
	RepoID     int64  `json:"repo_id"`
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
		fmt.Println("err new gh service :", err)
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
			fmt.Println("err list repo :", err)
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

		serviceIDs, err := q.GetAllAppServicesByRepo(h.qCtx, db.GetAllAppServicesByRepoParams{
			GhRepoID: repo.GetID(),
			Branch:   branch,
		})
		if err != nil {
			return nil
		}

		for _, sID := range serviceIDs {
			// TODO : check if watch path matches the pushed code commit

			// push a new deployment job to the queue
			if _, _, _, err := h.Server.Services.Deployment.AssignRebuild(context.Background(), &deployjob.RebuildServiceParams{
				ServiceID:  sID,
				CommitHash: pushEvent.GetAfter(),
				CommitMsg:  pushEvent.GetHeadCommit().GetMessage(),
				Source:     "webhook",
			}, nil); err != nil {
				// Failure for one service should not block others; the error is already
				// logged by the deployment service.
				continue
			}
		}
	}

	// pull_request: cache PR, create preview on open, rebuild on sync, cleanup on close
	// usecase : too keep a cached layer of PR info in DB
	if eventType == "pull_request" {
		prEvent, ok := event.(*github.PullRequestEvent)
		if !ok {
			return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "Invalid PR event payload"})
		}

		action := prEvent.GetAction()
		pr := prEvent.GetPullRequest()
		repo := prEvent.GetRepo()

		switch action {
		case "opened", "reopened", "synchronize":
			_ = q.UpsertPullRequest(h.qCtx, db.UpsertPullRequestParams{
				ID:         uuid.New(),
				RepoID:     repo.GetID(),
				PrNumber:   int64(pr.GetNumber()),
				Title:      pr.GetTitle(),
				HeadBranch: prEvent.GetPullRequest().GetHead().GetRef(),
				BaseBranch: prEvent.GetPullRequest().GetBase().GetRef(),
				State:      pr.GetState(),
				HtmlUrl:    pr.GetHTMLURL(),
			})
		case "closed":
			_ = q.DeletePullRequest(h.qCtx, db.DeletePullRequestParams{
				RepoID:   repo.GetID(),
				PrNumber: int64(pr.GetNumber()),
			})

			instanceIDs, err := q.GetAllInstanceByPR(h.qCtx, fmt.Sprintf("%d", pr.GetNumber()))
			if err != nil {
				fmt.Println("failed to get instances for PR:", err)
				return nil
			}

			for _, instanceID := range instanceIDs {
				if err := h.Server.Services.Deployment.DeletePreview(h.qCtx, instanceID); err != nil {
					fmt.Println("failed to delete preview for instance:", instanceID, "error:", err)
				}
			}
		}
	}

	// issue_comment: handle /godploy deploy command
	if eventType == "issue_comment" {
		icEvent, ok := event.(*github.IssueCommentEvent)
		if !ok {
			return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "Invalid issue_comment event payload"})
		}

		// verify if comment was created
		if icEvent.GetAction() != "created" {
			return nil
		}

		// only allow users with write/admin access to trigger deploy command
		commenter := icEvent.GetComment().GetUser().GetLogin()
		repoOwner := icEvent.GetRepo().GetOwner().GetLogin()
		repoName := icEvent.GetRepo().GetName()

		if icEvent.GetOrganization() != nil {
			// org repo: check collaborator permission level via GitHub API
			gh, err := ghservice.New(q, appID)
			if err != nil {
				fmt.Println("failed to create gh client for permission check:", err)
				return nil
			}
			perm, _, err := gh.Client.Repositories.GetPermissionLevel(context.Background(), repoOwner, repoName, commenter)
			if err != nil {
				fmt.Println("failed to check collaborator permission:", err)
				return nil
			}
			permLevel := perm.GetPermission()
			if permLevel != "admin" && permLevel != "write" {
				fmt.Printf("commenter %s lacks write/admin access to %s/%s (has: %s)\n", commenter, repoOwner, repoName, permLevel)
				return nil
			}
		} else {
			// user repo: commenter must be the repo owner
			if repoOwner != commenter {
				fmt.Println("not owner ", repoOwner, commenter)
				return nil
			}
		}

		// parse comment body for command
		cmd := strings.Split(strings.TrimSpace(icEvent.GetComment().GetBody()), " ")
		len := len(cmd)

		if len < 2 || cmd[0] != "/godploy" || cmd[1] != "deploy" {
			return nil
		}

		// only process PR comments (issue comments on PRs have PullRequestLinks)
		issue := icEvent.GetIssue()
		if issue == nil || issue.GetPullRequestLinks() == nil {
			return nil
		}

		repo := icEvent.GetRepo()
		prNumber := issue.GetNumber()
		repoID := repo.GetID()

		// upsert PR cache (comment may arrive before pull_request webhook)
		_ = q.UpsertPullRequest(h.qCtx, db.UpsertPullRequestParams{
			ID:         uuid.New(),
			RepoID:     repoID,
			PrNumber:   int64(prNumber),
			Title:      issue.GetTitle(),
			HeadBranch: "", // unknown from comment event; resolved later
			BaseBranch: "",
			State:      "open",
			HtmlUrl:    issue.GetHTMLURL(),
		})

		// check if project name is mentioned
		var projectName string
		if len == 3 {
			projectName = cmd[2]
		}

		// get all projects associated with this PR
		projectIDs, err := q.GetAllProjectIDsByPR(h.qCtx, db.GetAllProjectIDsByPRParams{
			ProjectName: projectName,
			RepoID:      repoID,
			PrNumber:    fmt.Sprintf("%d", prNumber),
		})
		if err != nil {
			fmt.Println("failed to get projects for PR:", err)
			return nil
		}

		// attempt to create a preview for this PR
		for _, ID := range projectIDs {
			fmt.Println("start a assign create preview for ", ID)
			if err := h.Server.Services.Deployment.AssignCreatePreview(h.qCtx, &deployjob.CreatePreviewJobParams{
				ProjectID:      ID,
				Name:           fmt.Sprintf("pr-%d", prNumber),
				PRNumber:       prNumber,
				RepoID:         int(repo.GetID()),
				GitSourceType:  "pr",
				GitSourceValue: fmt.Sprintf("%d", prNumber),
			}, nil); err != nil {
				fmt.Println("erro r ", err)
			}
		}
	}

	return nil
}

func fetchRepoPRs(ctx context.Context, gh *ghservice.GithubService, ghRepoName string, repoID int64) ([]PRInfo, error) {
	parts := strings.Split(ghRepoName, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repository name format: %s", ghRepoName)
	}
	owner := parts[0]
	repo := parts[1]

	opts := &github.PullRequestListOptions{
		State: "open",
		ListOptions: github.ListOptions{
			PerPage: 100,
			Page:    1,
		},
	}

	var prInfos []PRInfo
	for {
		prs, resp, err := gh.Client.PullRequests.List(ctx, owner, repo, opts)
		if err != nil {
			return nil, err
		}

		for _, pr := range prs {
			headBranch := ""
			if pr.Head != nil && pr.Head.Ref != nil {
				headBranch = *pr.Head.Ref
			}
			prInfos = append(prInfos, PRInfo{
				ID:         pr.GetID(),
				Number:     pr.GetNumber(),
				Title:      pr.GetTitle(),
				State:      pr.GetState(),
				HtmlURL:    pr.GetHTMLURL(),
				HeadBranch: headBranch,
				RepoID:     repoID,
			})
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return prInfos, nil
}

// GetGithubPRList gets all PRs for a given service's repo
// route: GET /api/provider/github/pr/list?service_id=
func (h *GitHandler) GetGithubPRList(c *echo.Context) error {
	q := h.Server.DB.Queries

	serviceID, err := uuid.Parse(c.QueryParam("service_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "Invalid service_id"})
	}

	service, err := q.GetAppServiceOnly(h.qCtx, serviceID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusNotFound, types.Res[struct{}]{Message: "Service not found"})
		}
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to fetch service details"})
	}

	gh, err := ghservice.New(q, service.GhAppID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to create github client"})
	}

	prInfos, err := fetchRepoPRs(h.ghCtx, gh, service.GhRepoName, service.GhRepoID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to fetch pull requests"})
	}

	return c.JSON(http.StatusOK, types.Res[[]PRInfo]{
		Message: "Success",
		Data:    prInfos,
	})
}

// GetGithubPRListByInstance gets all PRs for all services in an instance
// route: GET /api/provider/github/pr/instance?instance_id=
func (h *GitHandler) GetGithubPRListByInstance(c *echo.Context) error {
	q := h.Server.DB.Queries

	instanceID, err := uuid.Parse(c.QueryParam("instance_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "Invalid instance_id"})
	}

	services, err := q.GetAppServicesByInstanceId(h.qCtx, instanceID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to fetch services"})
	}

	res := make(map[string][]PRInfo)
	clients := make(map[int64]*ghservice.GithubService)

	for _, s := range services {
		gh, ok := clients[s.GhAppID]
		if !ok {
			var err error
			gh, err = ghservice.New(q, s.GhAppID)
			if err != nil {
				fmt.Printf("Error creating github client for app %d: %v\n", s.GhAppID, err)
				continue
			}
			clients[s.GhAppID] = gh
		}

		prInfos, err := fetchRepoPRs(h.ghCtx, gh, s.GhRepoName, s.GhRepoID)
		if err != nil {
			fmt.Printf("Error fetching PRs for repo %s: %v\n", s.GhRepoName, err)
			res[s.Name] = []PRInfo{}
			continue
		}

		res[s.Name] = prInfos
	}

	return c.JSON(http.StatusOK, types.Res[map[string][]PRInfo]{
		Message: "Success",
		Data:    res,
	})
}
