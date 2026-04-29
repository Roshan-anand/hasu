package handlers

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/Roshan-anand/godploy/internal/db"
	deploymentqueue "github.com/Roshan-anand/godploy/internal/jobs/deployment/queue"
	"github.com/Roshan-anand/godploy/internal/lib"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

type CreateAppServiceReq struct {
	ProjectID   uuid.UUID `json:"project_id" validate:"required"`
	Name        string    `json:"name" validate:"required"`
	AppName     string    `json:"app_name" validate:"required"`
	Description string    `json:"description"`
	GitProvider string    `json:"git_provider" validate:"required"`
	GhAppID     int64     `json:"gh_app_id" validate:"required"`
	GitRepoID   string    `json:"git_repo_id" validate:"required"`
	GitRepoName string    `json:"git_repo_name" validate:"required"`
	GitRepoURL  string    `json:"git_repo_url" validate:"required"`
	GitBranch   string    `json:"git_branch" validate:"required"`
	BuildPath   string    `json:"build_path" validate:"required"`
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

	// create a new app service
	b.AppName += lib.GenerateRandomID(6)
	service, err := q.CreateAppService(h.qCtx, db.CreateAppServiceParams{
		ID:          lib.NewID(),
		ProjectID:   b.ProjectID,
		Type:        types.AppServiceType,
		Name:        b.Name,
		AppName:     b.AppName,
		Description: b.Description,
		GitProvider: b.GitProvider,
		GitRepoID:   b.GitRepoID,
		GitRepoName: b.GitRepoName,
		GitBranch:   b.GitBranch,
		BuildPath:   b.BuildPath,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, lib.Res{Message: "Failed to create service"})
	}

	// create a new deployment for the app service
	deploymentName := fmt.Sprintf("%s-%s", b.AppName, lib.GenerateRandomID(6))
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
		Url:          b.GitRepoURL,
		Branch:       b.GitBranch,
		Token:        token,
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

	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	if err := h.Server.DB.Queries.DeleteAppService(h.qCtx, b.ServiceId); err != nil {
		return c.JSON(http.StatusInternalServerError, lib.Res{Message: "Failed to delete service"})
	}

	return c.JSON(http.StatusOK, lib.Res{Message: "Successsfully deleted service"})
}
