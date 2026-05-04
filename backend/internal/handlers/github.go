package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Roshan-anand/godploy/internal/config"
	"github.com/Roshan-anand/godploy/internal/db"
	"github.com/Roshan-anand/godploy/internal/lib"
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
	u := c.Get(h.Server.Config.EchoCtxUserKey).(lib.AuthUser)

	state, err := lib.GenerateCSRFToken()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, lib.Res{Message: "Failed to create github app"})
	}

	user, err := q.GetUserByEmail(h.qCtx, u.Email)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, lib.Res{Message: "Failed to create github app"})
	}

	if err := q.CreateRedirectSession(h.qCtx, db.CreateRedirectSessionParams{
		State:     state,
		OrgID:     user.OrgID,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, lib.Res{Message: "Failed to create github app"})
	}

	// TODO : start a worker which waits for 1hr and check if github app is not filled else remove it.
	// modify the removeSession

	manifest, err := getManifestData(h.Server.Config.ServerUrl, state)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, lib.Res{Message: "Failed to create github app"})
	}

	tmpl, err := template.New("manifest").Parse(githubManifestFormTmpl)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, lib.Res{Message: "Failed to create github app"})
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, map[string]string{
		"State":    state,
		"Manifest": manifest,
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, lib.Res{Message: "Failed to create github app"})
	}

	return c.HTML(http.StatusOK, buf.String())
}

// get github app credentials from GitHub
//
// route: GET /api/provider/github/app/callback
func (h *GitHandler) CreateGithubAppCallback(c *echo.Context) error {
	q := h.Server.DB.Queries
	// u := c.Get(h.Server.Config.EchoCtxUserKey).(lib.AuthUser)

	code := c.QueryParam("code")
	state := c.QueryParam("state")

	// validate the state
	sData, err := q.GetRedirectSession(h.qCtx, state)
	if err != nil {
		return c.JSON(http.StatusBadRequest, lib.Res{Message: "Invalid state"})
	}

	if time.Now().After(sData.ExpiresAt) {
		go removeSession(q, state)
		return c.JSON(http.StatusBadRequest, lib.Res{Message: "State has expired"})
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
	encryptedPem, err := lib.EncryptPEM(convRes.PEM)
	if err != nil {
		return c.Redirect(http.StatusFound, "/?github_error=internal")
	}

	// store the app credentials in db
	ghAppId, err := q.CreateGithubApp(h.qCtx, db.CreateGithubAppParams{
		ID:             lib.NewID(),
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
		fmt.Println("Error updating redirect session with github app id:", err)
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
		fmt.Println("Error fetching redirect session:", err)
		return c.JSON(http.StatusBadRequest, lib.Res{Message: "Invalid state"})
	}
	go removeSession(q, state)

	instllation_id, err := strconv.ParseInt(c.QueryParam("installation_id"), 10, 64)
	if err != nil {
		fmt.Println("Error parsing installation ID:", err)
		return c.JSON(http.StatusBadRequest, lib.Res{Message: "Invalid installation ID"})
	}

	// varify installation ID
	ghApp, err := q.GetGhAppByAppId(h.qCtx, ghAppId.Int64)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, lib.Res{Message: "Failed to setup github app"})
	}

	// get app client
	appClient, err := lib.CreateAppClient(ghApp.AppID, ghApp.PemKey)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, lib.Res{Message: "Failed to setup github app"})
	}

	// verify installation ID by making an authenticated request to GitHub API
	_, _, err = appClient.Apps.GetInstallation(context.Background(), instllation_id)
	if err != nil {
		fmt.Println("Error verifying installation ID:", err)
		return c.JSON(http.StatusBadRequest, lib.Res{Message: "Invalid installation ID"})
	}

	if err := q.InsertInstallationID(h.qCtx, db.InsertInstallationIDParams{
		InstallationID: sql.NullInt64{
			Int64: instllation_id,
			Valid: true,
		},
		AppID: ghApp.AppID,
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, lib.Res{Message: "Failed to setup github app"})
	}

	// TODO: update the url to route to git provider page with success message
	return c.Redirect(http.StatusFound, h.Server.Config.WebUrl+"/#/git")
}

// get all the github app info
//
// route: GET /api/provider/github/app/list
func (h *GitHandler) GetAllGithubApps(c *echo.Context) error {
	u := c.Get(h.Server.Config.EchoCtxUserKey).(lib.AuthUser)
	q := h.Server.DB.Queries

	ghApps, err := q.GetAllGhAppsByEmail(h.qCtx, u.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusOK, nil)
		}
		fmt.Println("Error fetching github app:", err)
		return c.JSON(http.StatusInternalServerError, lib.Res{Message: "Failed to get github app"})
	}

	return c.JSON(http.StatusOK, ghApps)
}

// delete github app for admin users
//
// route: DELETE /api/provider/github/app
func (h *GitHandler) DeleteGithubApp(c *echo.Context) error {
	u := c.Get(h.Server.Config.EchoCtxUserKey).(lib.AuthUser)
	b := new(DeleteGithubAppReq)

	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	q := h.Server.DB.Queries
	if isAdmin, err := q.IsUserAdmin(h.qCtx, u.Email); err != nil {
		return c.JSON(http.StatusInternalServerError, lib.Res{Message: "internal server error"})
	} else if !isAdmin {
		return c.JSON(http.StatusForbidden, lib.Res{Message: "admin access required"})
	}

	ghApp, err := q.GetGhAppByAppId(h.ghCtx, b.AppID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, lib.Res{Message: "Failed to delete github app"})
	}

	client, err := lib.CreateAppClient(ghApp.AppID, ghApp.PemKey)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, lib.Res{Message: "Failed to delete github app"})
	}

	_, err = client.Apps.DeleteInstallation(h.qCtx, ghApp.AppID)
	if err != nil {
		fmt.Println("Error deleting github app installation:", err)
		return c.JSON(http.StatusInternalServerError, lib.Res{Message: "Failed to delete github app"})
	}

	if err := q.DeleteGithubApp(h.qCtx, b.AppID); err != nil {
		return c.JSON(http.StatusInternalServerError, lib.Res{Message: "Failed to delete github app"})
	}

	return c.JSON(http.StatusOK, lib.Res{Message: "Github app deleted successfully"})
}

// get list of repos accessible by the github app
//
// route: GET /api/provider/github/repo/list?app_id=
func (h *GitHandler) GetGithubRepoList(c *echo.Context) error {
	q := h.Server.DB.Queries

	appID, err := strconv.ParseInt(c.QueryParam("app_id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, lib.Res{Message: "Invalid app_id"})
	}

	ghApp, err := q.GetGhAppByAppId(h.qCtx, appID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusConflict, lib.Res{Message: "No github connected"})
		}
		return c.JSON(http.StatusInternalServerError, lib.Res{Message: "Failed to get github repos"})
	}

	if !ghApp.InstallationID.Valid || ghApp.InstallationID.Int64 == 0 {
		return c.JSON(http.StatusConflict, lib.Res{Message: "No github connected"})
	}

	ghClient, err := lib.CreateGithubClient(context.Background(), ghApp.AppID, ghApp.InstallationID.Int64, ghApp.PemKey)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, lib.Res{Message: "Failed to get github repos"})
	}

	opts := &github.ListOptions{
		PerPage: 100,
		Page:    1,
	}

	repos := make([]GetGithubRepoListRes, 0)

	for {
		pageRepos, resp, err := ghClient.Apps.ListRepos(h.ghCtx, opts)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, lib.Res{Message: "Failed to get github repos"})
		}

		for _, repo := range pageRepos.Repositories {
			owner := repo.GetOwner().GetLogin()
			repoName := repo.GetName()
			if owner == "" || repoName == "" {
				continue
			}

			repos = append(repos, GetGithubRepoListRes{
				ID:            repo.GetID(),
				Name:          repo.GetName(),
				FullName:      repo.GetFullName(),
				Private:       repo.GetPrivate(),
				DefaultBranch: repo.GetDefaultBranch(),
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

	return c.JSON(http.StatusOK, repos)
}
