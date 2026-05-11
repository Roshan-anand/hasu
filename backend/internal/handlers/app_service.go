package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/Roshan-anand/godploy/internal/db"
	deploymentqueue "github.com/Roshan-anand/godploy/internal/jobs/deployment/queue"
	"github.com/Roshan-anand/godploy/internal/lib"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/moby/moby/client"
)

type DockerBuildReq struct {
	FilePath     string `json:"file_path"`
	ContextPath  string `json:"context_path"`
	BuildStage   string `json:"build_stage"`
	BuildArgs    string `json:"build_args"`
	BuildSecrets string `json:"build_secrets"`
}

type CreateAppServiceReq struct {
	OrgID         uuid.UUID       `json:"org_id" validate:"required"`
	Name          string          `json:"name" validate:"required,min=3,max=50"`
	GitProvider   string          `json:"git_provider" validate:"required"`
	GhAppID       int64           `json:"gh_app_id" validate:"required"`
	GhRepoID      string          `json:"gh_repo_id" validate:"required"`
	GhRepoName    string          `json:"gh_repo_name" validate:"required"`
	GhRepoURL     string          `json:"gh_repo_url" validate:"required"`
	DefaultBranch string          `json:"default_branch" validate:"required"`
	BuildPath     string          `json:"build_path" validate:"required"`
	WatchPath     string          `json:"watch_path" validate:"required"`
	Env           string          `json:"env"`
	DockerBuild   *DockerBuildReq `json:"docker_build"`
}

type UpdateDomainReq struct {
	BranchID uuid.UUID `json:"branch_id" validate:"required"`
	Domain   string    `json:"domain" validate:"required"`
	Port     int32     `json:"port" validate:"required"`
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

	// check if service name already exists in the organization
	if exists, err := q.ServiceNameExists(h.qCtx, db.ServiceNameExistsParams{
		OrgID: b.OrgID,
		Name:  b.Name,
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res{Message: "Failed to check service name"})
	} else if exists {
		return c.JSON(http.StatusConflict, types.Res{Message: "Service name already exists"})
	}

	// get the github app details
	ghApp, err := q.GetGhAppByAppId(h.qCtx, b.GhAppID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusBadRequest, types.Res{Message: "invalid github app"})
		}
		return c.JSON(http.StatusInternalServerError, types.Res{Message: "failed to verify github app"})
	}

	// parse url
	u, err := url.Parse(b.GhRepoURL)
	if err != nil {
		panic(err)
	}
	url := u.Host + u.Path

	// used as unique container name and code storing path
	serviceName := fmt.Sprintf("%s-%s-%s", b.Name, b.DefaultBranch, lib.GenerateRandomID(6))
	imgName := strings.ToLower(fmt.Sprintf("%s-%s-dyp_%s", b.Name, b.DefaultBranch, lib.GenerateRandomID(6)))

	// start a new db transaction
	tx, err := h.Server.DB.Pool.BeginTx(context.Background(), nil)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res{Message: "Failed to create service"})
	}
	q = q.WithTx(tx)

	// create a new service
	service, err := q.CreateAppService(h.qCtx, db.CreateAppServiceParams{
		ID:             lib.GeneratePrimaryKey(),
		OrganizationID: b.OrgID,
		Type:           types.AppServiceType,
		Name:           b.Name,
		GitProvider:    b.GitProvider,
		GhAppID:        ghApp.AppID,
		GhRepoID:       b.GhRepoID,
		GhRepoName:     b.GhRepoName,
		GhRepoUrl:      url,
		BuildPath:      b.BuildPath,
		WatchPath:      b.WatchPath,
		Env:            b.Env,
		BuildArgs:      b.DockerBuild.BuildArgs,
		BuildSecrets:   b.DockerBuild.BuildSecrets,
	})
	if err != nil {
		tx.Rollback()
		return c.JSON(http.StatusInternalServerError, types.Res{Message: "Failed to create service"})
	}

	// create a new branch for the app service
	branchID, err := q.CreateAppServiceBranch(h.qCtx, db.CreateAppServiceBranchParams{
		ID:               lib.GeneratePrimaryKey(),
		IsDefaultBranch:  true,
		BranchName:       b.DefaultBranch,
		SwarmServiceName: serviceName,
		ServiceID:        service.ID,
	})
	if err != nil {
		tx.Rollback()
		return c.JSON(http.StatusInternalServerError, types.Res{Message: "Failed to create service branch"})
	}

	// TODO : get commit msg from client side
	// create a new deployment for the app service
	dID, err := q.CreateDeployment(h.qCtx, db.CreateDeploymentParams{
		ID:        lib.GeneratePrimaryKey(),
		BranchID:  branchID,
		CommitMsg: "s",
		IsLatest:  true,
	})
	if err != nil {
		tx.Rollback()
		return c.JSON(http.StatusInternalServerError, types.Res{Message: "Failed to create deployment"})
	}

	if err := tx.Commit(); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res{Message: "Failed to create service"})
	}

	// get gh token
	token, err := lib.GetGhToken(ghApp.AppID, ghApp.InstallationID.Int64, ghApp.PemKey)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res{Message: "Failed to get github token"})
	}

	// push a new deployment job to the queue
	h.Server.DeploymentQ.EnqueuePullJob(&deploymentqueue.PullJobData{
		DeploymentID:      dID,
		Token:             token,
		Url:               url,
		Branch:            b.DefaultBranch,
		SwarmServiceName:  serviceName,
		BuildPath:         b.BuildPath,
		DockerFilePath:    b.DockerBuild.FilePath,
		DockerContextPath: b.DockerBuild.ContextPath,
		DockerBuildStage:  b.DockerBuild.BuildStage,
		ImgName:           imgName,
	})

	return c.JSON(http.StatusOK, service.ID)
}

// get app service details by id
//
// route: GET /api/service/app/:id
func (h *ServiceHandler) GetAppServiceById(c *echo.Context) error {
	serviceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res{Message: "invalid service id"})
	}

	service, err := h.Server.DB.Queries.GetAppServiceById(h.qCtx, serviceID)
	if err != nil {
		return c.JSON(http.StatusNotFound, types.Res{Message: "service not found"})
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

	serviceInfo, err := q.GetAllSwarmServiceAndImagesByAppServiceId(h.qCtx, b.ServiceId)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res{Message: "Failed to get deployments"})
	}

	// arrange all ids and imgs sepratly for easy access
	dIDs := make([]uuid.UUID, len(serviceInfo))
	imgs := make([]string, len(serviceInfo))
	swarmServiceNames := make(map[string]struct{})
	for i, s := range serviceInfo {
		dIDs[i] = s.DeploymentID
		if s.ImageName.Valid {
			imgs[i] = s.ImageName.String
		}
		swarmServiceNames[s.SwarmServiceName] = struct{}{}
	}

	// stop all the services running and remove all the images
	go func() {
		h.Server.Docker.RemoveServices(swarmServiceNames)
		h.Server.Docker.RemoveImages(imgs)
	}()

	// delete all logs related to the service deployments
	go h.Server.BadgerDB.DeleteAllLogsByDeploymentID(dIDs)

	// delete the app service
	if err := h.Server.DB.Queries.DeleteAppService(h.qCtx, b.ServiceId); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res{Message: "Failed to delete service"})
	}

	return c.JSON(http.StatusOK, types.Res{Message: "Successsfully deleted service"})
}

// update domain and port
//
// route: PUT /api/service/app/domain
func (h *ServiceHandler) UpdateAppServiceDomain(c *echo.Context) error {
	b := new(UpdateDomainReq)
	q := h.Server.DB.Queries
	docker := h.Server.Docker.Client

	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	// check if url is valid
	if _, err := url.Parse(b.Domain); err != nil {
		return c.JSON(http.StatusBadRequest, types.Res{Message: "invalid domain"})
	}

	swarmService, err := q.GetSwarmServiceByBranchId(h.qCtx, b.BranchID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res{Message: "Failed to get swarm service"})
	}

	// get service spec to update the labels
	inspectRes, err := docker.ServiceInspect(context.Background(), swarmService, client.ServiceInspectOptions{})
	if err != nil {
		fmt.Printf("Failed to inspect swarm service: %v\n", err)
		return c.JSON(http.StatusInternalServerError, types.Res{Message: "Failed to inspect swarm service"})
	}
	serviceV := inspectRes.Service.Meta.Version
	spec := inspectRes.Service.Spec

	// add domain specfic labels
	spec.Annotations.Labels[fmt.Sprintf("traefik.http.routers.%s.rule", swarmService)] = fmt.Sprintf("Host(`%s`)", b.Domain)
	spec.Annotations.Labels[fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port", swarmService)] = fmt.Sprintf("%d", b.Port)

	// update the swarm service
	if _, err := docker.ServiceUpdate(context.Background(), swarmService, client.ServiceUpdateOptions{
		Version: serviceV,
		Spec:    spec,
	}); err != nil {
		fmt.Printf("Failed to update swarm service: %v\n", err)
		return c.JSON(http.StatusInternalServerError, types.Res{Message: "Failed to update swarm service"})
	}

	// update the branch table
	if err := q.SetDomianAndPortForBranch(h.qCtx, db.SetDomianAndPortForBranchParams{
		Domain: b.Domain,
		Port:   b.Port,
		ID:     b.BranchID,
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res{Message: "Failed to update domain and port"})
	}

	return c.JSON(http.StatusOK, types.Res{Message: "Successfully updated domain and port"})
}

// get branch domain and port by service id
//
// route: GET /api/service/app/domain?service_id
func (h *ServiceHandler) GetBranchDomain(c *echo.Context) error {
	q := h.Server.DB.Queries

	serviceID, err := uuid.Parse(c.QueryParam("service_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res{Message: "invalid service_id"})
	}

	branches, err := q.GetBranchesDomainByServiceId(h.qCtx, serviceID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res{Message: "Failed to get branch domain"})
	}

	return c.JSON(http.StatusOK, branches)
}
