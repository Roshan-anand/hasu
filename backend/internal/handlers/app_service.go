package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/url"

	"github.com/Roshan-anand/godploy/internal/db"
	deploymentqueue "github.com/Roshan-anand/godploy/internal/jobs/deployment/queue"
	"github.com/Roshan-anand/godploy/internal/lib"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

type CreateAppServiceReq struct {
	OrgID         uuid.UUID `json:"org_id" validate:"required"`
	Name          string    `json:"name" validate:"required"`
	GitProvider   string    `json:"git_provider" validate:"required"`
	GhAppID       int64     `json:"gh_app_id" validate:"required"`
	GitRepoID     string    `json:"git_repo_id" validate:"required"`
	GitRepoName   string    `json:"git_repo_name" validate:"required"`
	GitRepoURL    string    `json:"git_repo_url" validate:"required"`
	DefaultBranch string    `json:"default_branch" validate:"required"`
	BuildPath     string    `json:"build_path" validate:"required"`
	WatchPath     string    `json:"watch_path" validate:"required"`
}

type UpdateAppServiceReq struct {
	ServiceID     uuid.UUID `json:"service_id" validate:"required"`
	GitProvider   string    `json:"git_provider" validate:"required"`
	GhAppID       int64     `json:"gh_app_id" validate:"required"`
	GitRepoID     string    `json:"git_repo_id" validate:"required"`
	GitRepoName   string    `json:"git_repo_name" validate:"required"`
	GitRepoURL    string    `json:"git_repo_url" validate:"required"`
	DefaultBranch string    `json:"default_branch" validate:"required"`
	BuildPath     string    `json:"build_path" validate:"required"`
	WatchPath     string    `json:"watch_path" validate:"required"`
}

// create a new app service
//
// route: POST /api/service/app
func (h *ServiceHandler) CreateAppService(c *echo.Context) error {
	b := new(CreateAppServiceReq)
	q := h.Server.DB.Queries

	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	ghApp, err := q.GetGhAppByAppId(h.qCtx, b.GhAppID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusBadRequest, lib.Res{Message: "invalid github app"})
		}
		return c.JSON(http.StatusInternalServerError, lib.Res{Message: "failed to verify github app"})
	}

	// parse url
	u, err := url.Parse(b.GitRepoURL)
	if err != nil {
		panic(err)
	}
	url := u.Host + u.Path

	serviceName := fmt.Sprintf("%s-%s", b.Name, lib.GenerateRandomID(6))
	service, err := q.CreateAppService(h.qCtx, db.CreateAppServiceParams{
		ID:             lib.NewID(),
		OrganizationID: b.OrgID,
		Type:           types.AppServiceType,
		ServiceID:      "",
		Name:           b.Name,
		AppName:        serviceName,
		GitProvider:    b.GitProvider,
		GhAppID:        ghApp.AppID,
		GitRepoID:      b.GitRepoID,
		GitRepoName:    b.GitRepoName,
		GitRepoUrl:     url,
		DefaultBranch:  b.DefaultBranch,
		BuildPath:      b.BuildPath,
		WatchPath:      b.WatchPath,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, lib.Res{Message: "Failed to create service"})
	}

	// create a new deployment for the app service
	deploymentName := fmt.Sprintf("%s-%s", serviceName, lib.GenerateRandomID(6))
	dID, err := q.CreateDeployment(h.qCtx, db.CreateDeploymentParams{
		ID:        lib.NewID(),
		Name:      deploymentName,
		ServiceID: service.ID,
		Status:    types.DeploymentInProgress,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, lib.Res{Message: "Failed to create deployment"})
	}

	// get gh token
	token, err := lib.GetGhToken(ghApp.AppID, ghApp.InstallationID.Int64, ghApp.PemKey)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, lib.Res{Message: "Failed to get github token"})
	}

	// push a new deployment job to the queue
	h.Server.DeploymentQ.EnqueuePullJob(&deploymentqueue.PullJobData{
		DeploymentID: dID,
		Token:        token,
		Url:          url,
		Branch:       b.DefaultBranch,
		BuildPath:    b.BuildPath,
	})

	return c.JSON(http.StatusOK, service)
}

// get app service details by id
//
// route: GET /api/service/app/:id
func (h *ServiceHandler) GetAppServiceById(c *echo.Context) error {
	serviceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, lib.Res{Message: "invalid service id"})
	}

	service, err := h.Server.DB.Queries.GetAppServiceById(h.qCtx, serviceID)
	if err != nil {
		return c.JSON(http.StatusNotFound, lib.Res{Message: "service not found"})
	}

	return c.JSON(http.StatusOK, service)
}

// delete app service
//
// route: DELETE /api/service/app
func (h *ServiceHandler) DeleteAppService(c *echo.Context) error {
	b := new(ServiceReq)
	q := h.Server.DB.Queries

	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	dIDs, err := q.GetAllDeploymentIdsByServiceID(h.qCtx, b.ServiceId)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, lib.Res{Message: "Failed to get deployments"})
	}

	if err := h.Server.DB.Queries.DeleteAppService(h.qCtx, b.ServiceId); err != nil {
		return c.JSON(http.StatusInternalServerError, lib.Res{Message: "Failed to delete service"})
	}

	go h.Server.BadgerDB.DeleteAllLogsByDeploymentID(dIDs)

	return c.JSON(http.StatusOK, lib.Res{Message: "Successsfully deleted service"})
}

// route: POST /api/service/app/update
func (h *ServiceHandler) UpdateAppService(c *echo.Context) error {
	b := new(UpdateAppServiceReq)
	q := h.Server.DB.Queries

	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	ghApp, err := q.GetGhAppByAppId(h.qCtx, b.GhAppID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusBadRequest, lib.Res{Message: "invalid github app"})
		}
		return c.JSON(http.StatusInternalServerError, lib.Res{Message: "failed to verify github app"})
	}

	if err := q.UpdateAppServiceDetails(h.qCtx, db.UpdateAppServiceDetailsParams{
		GitProvider:   b.GitProvider,
		GhAppID:       ghApp.AppID,
		GitRepoID:     b.GitRepoID,
		GitRepoName:   b.GitRepoName,
		GitRepoUrl:    b.GitRepoURL,
		DefaultBranch: b.DefaultBranch,
		BuildPath:     b.BuildPath,
		WatchPath:     b.WatchPath,
		ID:            b.ServiceID,
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, lib.Res{Message: "failed to update app service"})
	}

	return c.JSON(http.StatusOK, lib.Res{Message: "app service updated successfully"})
}
