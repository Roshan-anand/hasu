package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/url"

	"github.com/Roshan-anand/godploy/internal/db"
	deploymentqueue "github.com/Roshan-anand/godploy/internal/jobs/deployment/queue"
	ghservice "github.com/Roshan-anand/godploy/internal/lib/gh"
	"github.com/Roshan-anand/godploy/internal/lib/security"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/Roshan-anand/godploy/internal/lib/utils"
	"github.com/docker/docker/api/types/swarm"
	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

type DockerBuildReq struct {
	FilePath    string `json:"file_path"`
	ContextPath string `json:"context_path"`
	BuildStage  string `json:"build_stage"`
}

type CreateAppServiceReq struct {
	ProjectID     uuid.UUID       `json:"project_id" validate:"required"`
	Name          string          `json:"name" validate:"required,min=3,max=50"`
	GitProvider   string          `json:"git_provider" validate:"required"`
	GhAppID       int64           `json:"gh_app_id" validate:"required"`
	GhRepoID      int64           `json:"gh_repo_id" validate:"required"`
	DefaultBranch string          `json:"default_branch" validate:"required"`
	BuildPath     string          `json:"build_path" validate:"required"`
	WatchPath     string          `json:"watch_path" validate:"required"`
	Env           []string        `json:"env"`
	BuildArgs     []string        `json:"build_args"`
	BuildSecrets  []string        `json:"build_secrets"`
	DockerBuild   *DockerBuildReq `json:"docker_build"`
	Public        bool            `json:"public"`
}

type CreatePreviewAppServiceReq struct {
	ServiceID uuid.UUID `json:"service_id" validate:"required"`
	Branch    string    `json:"branch" validate:"required"`
}

type UpdateDomainReq struct {
	BranchID uuid.UUID `json:"branch_id" validate:"required"`
	Domain   string    `json:"domain" validate:"required"`
	Port     int32     `json:"port" validate:"required"`
}

type UpdateEnvReq struct {
	ServiceID    uuid.UUID `json:"service_id" validate:"required"`
	Env          []string  `json:"env" validate:"required"`
	BuildArgs    []string  `json:"build_args" validate:"required"`
	BuildSecrets []string  `json:"build_secrets" validate:"required"`
}

type GetEnvRes struct {
	Env          []string `json:"env" validate:"required"`
	BuildArgs    []string `json:"build_args" validate:"required"`
	BuildSecrets []string `json:"build_secrets" validate:"required"`
}

type RebuildServiceReq struct {
	BranchID uuid.UUID `json:"branch_id" validate:"required"`
}

type RoolbackServiceReq struct {
	BranchID uuid.UUID `json:"branch_id" validate:"required"`
}

type ScaleAppServiceReq struct {
	ServiceId uuid.UUID `json:"service_id" validate:"required"`
	Replicas  uint64    `json:"replicas" validate:"required"`
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
		Name:      b.Name,
		ProjectID: b.ProjectID,
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to check service name"})
	} else if exists {
		return c.JSON(http.StatusConflict, types.Res[struct{}]{Message: "Service name already exists"})
	}

	// create a new github client
	gh, err := ghservice.New(q, b.GhAppID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: fmt.Sprintf("github app with app id %d not found", b.GhAppID)})
		}
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "failed to create github client"})
	}

	// get the github repository details
	repo, err := gh.GetRepo(b.GhRepoID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "failed to fetch repository info from github"})
	}

	// get the latest commit info of the selected branch
	commit, err := gh.GetLatestCommit(repo.Owner, repo.Name, b.DefaultBranch)
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "failed to fetch latest commit info from github"})
	}

	url, err := utils.GetUrltHostNPath(repo.URL)
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "invalid repository url"})
	}

	// used as unique image and service name
	unique := generateServiceAndImgName(b.Name, b.DefaultBranch)

	// clear the evnironment array
	b.Env = cleanArray(b.Env)
	b.BuildArgs = cleanArray(b.BuildArgs)
	b.BuildSecrets = cleanArray(b.BuildSecrets)

	// convert into bytes
	envByte, err := MarshalServiceEnv(&ServiceEnvArray{
		Env:          b.Env,
		BuildArgs:    b.BuildArgs,
		BuildSecrets: b.BuildSecrets,
	})
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "invalid env values"})
	}

	// start a new db transaction
	tx, err := h.Server.DB.Pool.BeginTx(context.Background(), nil)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to create service"})
	}
	tq := q.WithTx(tx)

	// create a new service
	service, err := tq.CreateAppService(h.qCtx, db.CreateAppServiceParams{
		ID:                security.GeneratePrimaryKey(),
		ProjectID:         b.ProjectID,
		Type:              types.AppServiceType,
		Name:              b.Name,
		GitProvider:       b.GitProvider,
		GhAppID:           b.GhAppID,
		GhRepoID:          b.GhRepoID,
		GhRepoName:        repo.FullName,
		GhRepoUrl:         url,
		BuildPath:         b.BuildPath,
		WatchPath:         b.WatchPath,
		Env:               envByte.Env,
		BuildArgs:         envByte.BuildArgs,
		BuildSecrets:      envByte.BuildSecrets,
		DockerFilepath:    b.DockerBuild.FilePath,
		DockerContextpath: b.DockerBuild.ContextPath,
		DockerBuildstage:  b.DockerBuild.BuildStage,
	})
	if err != nil {
		tx.Rollback()
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to create service"})
	}

	// create a new branch for the app service
	branchID, err := tq.CreateAppServiceBranch(h.qCtx, db.CreateAppServiceBranchParams{
		ID:               security.GeneratePrimaryKey(),
		IsDefaultBranch:  true,
		IsPublic:         b.Public,
		BranchName:       b.DefaultBranch,
		SwarmServiceName: unique.ServiceName,
		ServiceID:        service.ID,
	})
	if err != nil {
		tx.Rollback()
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to create service branch"})
	}

	// create a new deployment for the app service
	dID, err := tq.CreateDeployment(h.qCtx, db.CreateDeploymentParams{
		ID:         security.GeneratePrimaryKey(),
		BranchID:   branchID,
		CommitHash: commit.Hash,
		CommitMsg:  commit.Message,
		IsCurrent:  true,
	})
	if err != nil {
		tx.Rollback()
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to create deployment"})
	}

	// get project network
	network, err := tq.GetProjectNetwork(h.qCtx, b.ProjectID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to get project network"})
	}

	if err := tx.Commit(); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to create service"})
	}

	// push a new deployment job to the queue
	h.Server.DeploymentQ.EnqueuePullJob(&deploymentqueue.PullJobData{
		Type:              deploymentqueue.DeployJob,
		DeploymentID:      dID,
		Token:             gh.Token,
		Url:               url,
		Branch:            b.DefaultBranch,
		SwarmServiceName:  unique.ServiceName,
		BuildPath:         b.BuildPath,
		DockerFilePath:    b.DockerBuild.FilePath,
		DockerContextPath: b.DockerBuild.ContextPath,
		DockerBuildStage:  b.DockerBuild.BuildStage,
		ImgName:           unique.ServiceName,
		Env:               b.Env,
		BuildArgs:         b.BuildArgs,
		BuildSecrets:      b.BuildSecrets,
		IsPublic:          b.Public,
		NetworkName:       network,
	})

	return c.JSON(http.StatusOK, types.Res[uuid.UUID]{
		Message: "",
		Data:    service.ID,
	})
}

// TODO : under development, will be added in future PR
// create a new app service for a sub branch
//
// route: POST /api/service/app/preview
func (h *ServiceHandler) CreatePreviewAppService(c *echo.Context) error {
	b := new(CreatePreviewAppServiceReq)
	q := h.Server.DB.Queries

	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	service, err := q.GetAppServiceById(h.qCtx, b.ServiceID)
	if err != nil {
		return c.JSON(http.StatusNotFound, types.Res[struct{}]{Message: "service not found"})
	}

	// create a new github client
	gh, err := ghservice.New(q, service.GhAppID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "failed to create github client"})
	}

	// get the github repository details
	repo, err := gh.GetRepo(service.GhRepoID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "failed to fetch repository info from github"})
	}

	// varify if it is a valid branch
	branches, _, err := gh.Client.Repositories.ListBranches(context.Background(), repo.Owner, repo.Name, &github.BranchListOptions{})
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "failed to fetch branches from github"})
	}

	isValidBranch := false
	for _, branch := range branches {
		if branch.GetName() == b.Branch {
			isValidBranch = true
			break
		}
	}

	if !isValidBranch {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "invalid branch name"})
	}

	// check if branch with same name already in deployment
	if exists, err := q.CheckBranchExists(h.qCtx, db.CheckBranchExistsParams{
		ServiceID:  service.ID,
		BranchName: b.Branch,
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to check branch name"})
	} else if exists {
		return c.JSON(http.StatusConflict, types.Res[struct{}]{Message: "Branch with same name already exists"})
	}

	// used as unique image and service name
	unique := generateServiceAndImgName(service.Name, b.Branch)

	// start a new db transaction
	tx, err := h.Server.DB.Pool.BeginTx(context.Background(), nil)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to create service"})
	}
	tq := q.WithTx(tx)

	// create a new branch for the app service
	branchID, err := tq.CreateAppServiceBranch(h.qCtx, db.CreateAppServiceBranchParams{
		ID:               security.GeneratePrimaryKey(),
		IsDefaultBranch:  true,
		BranchName:       b.Branch,
		SwarmServiceName: unique.ServiceName,
		ServiceID:        service.ID,
	})
	if err != nil {
		tx.Rollback()
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to create service branch"})
	}

	// TODO : varify if passing addrs of struct is more efficient than passing 2 strings as args
	// get the latest commit info of the selected branch
	commit, err := gh.GetLatestCommit(repo.Owner, repo.Name, b.Branch)
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "failed to fetch latest commit info from github"})
	}

	// create a new deployment for the app service
	dID, err := tq.CreateDeployment(h.qCtx, db.CreateDeploymentParams{
		ID:         security.GeneratePrimaryKey(),
		BranchID:   branchID,
		CommitHash: commit.Hash,
		CommitMsg:  commit.Message,
		IsCurrent:  true,
	})
	if err != nil {
		tx.Rollback()
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to create deployment"})
	}

	network, err := tq.GetProjectNetwork(h.qCtx, service.ProjectID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to get project network"})
	}

	branch, err := tq.GetDefaultBranchByServiceId(h.qCtx, service.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to get branch info"})
	}

	if err := tx.Commit(); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to create service"})
	}

	envArray, err := UnmarshalServiceEnv(&ServiceEnvByte{
		Env:          service.Env,
		BuildArgs:    service.BuildArgs,
		BuildSecrets: service.BuildSecrets,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to create service"})
	}

	// push a new deployment job to the queue
	h.Server.DeploymentQ.EnqueuePullJob(&deploymentqueue.PullJobData{
		Type:              deploymentqueue.DeployJob,
		DeploymentID:      dID,
		Token:             gh.Token,
		Url:               service.GhRepoUrl,
		Branch:            b.Branch,
		SwarmServiceName:  unique.ServiceName,
		BuildPath:         service.BuildPath,
		DockerFilePath:    service.DockerFilepath,
		DockerContextPath: service.DockerContextpath,
		DockerBuildStage:  service.DockerBuildstage,
		ImgName:           unique.ImgName,
		Env:               envArray.Env,
		BuildArgs:         envArray.BuildArgs,
		BuildSecrets:      envArray.BuildSecrets,
		IsPublic:          branch.IsPublic,
		NetworkName:       network,
	})

	return c.JSON(http.StatusOK, types.Res[uuid.UUID]{
		Message: "",
		Data:    service.ID,
	})
}

// get app service details by id
//
// route: GET /api/service/app/:id
func (h *ServiceHandler) GetAppServiceById(c *echo.Context) error {
	serviceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "invalid service id"})
	}

	service, err := h.Server.DB.Queries.GetAppServiceById(h.qCtx, serviceID)
	if err != nil {
		return c.JSON(http.StatusNotFound, types.Res[struct{}]{Message: "service not found"})
	}

	return c.JSON(http.StatusOK, types.Res[db.GetAppServiceByIdRow]{
		Message: "",
		Data:    service,
	})
}

// get branch domain and port by service id
//
// route: GET /api/service/app/domain?service_id
func (h *ServiceHandler) GetBranchDomain(c *echo.Context) error {
	q := h.Server.DB.Queries

	serviceID, err := uuid.Parse(c.QueryParam("service_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "invalid service_id"})
	}

	branches, err := q.GetBranchesDomainByServiceId(h.qCtx, serviceID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to get branch domain"})
	}

	return c.JSON(http.StatusOK, types.Res[[]db.GetBranchesDomainByServiceIdRow]{
		Message: "",
		Data:    branches,
	})
}

// get branch domain and port by service id
//
// route: GET /api/service/app/env?service_id
func (h *ServiceHandler) GetServiceEnv(c *echo.Context) error {
	q := h.Server.DB.Queries

	serviceID, err := uuid.Parse(c.QueryParam("service_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "invalid service_id"})
	}

	e, err := q.GetServiceEnv(h.qCtx, serviceID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to get branch domain"})
	}

	envString, err := UnmarshalServiceEnv(&ServiceEnvByte{
		Env:          e.Env,
		BuildArgs:    e.BuildArgs,
		BuildSecrets: e.BuildSecrets,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to get branch domain"})
	}

	return c.JSON(http.StatusOK, types.Res[GetEnvRes]{
		Message: "",
		Data: GetEnvRes{
			Env:          envString.Env,
			BuildArgs:    envString.BuildArgs,
			BuildSecrets: envString.BuildSecrets,
		},
	})
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
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "invalid domain"})
	}

	swarmService, err := q.GetSwarmServiceByBranchId(h.qCtx, b.BranchID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to get swarm service"})
	}

	// get service spec to update the labels
	inspectRes, _, err := docker.ServiceInspectWithRaw(context.Background(), swarmService, swarm.ServiceInspectOptions{})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to inspect swarm service"})
	}
	serviceV := inspectRes.Version
	spec := inspectRes.Spec

	// add domain specfic labels
	spec.Annotations.Labels[fmt.Sprintf("traefik.http.routers.%s.rule", swarmService)] = fmt.Sprintf("Host(`%s`)", b.Domain)
	spec.Annotations.Labels[fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port", swarmService)] = fmt.Sprintf("%d", b.Port)

	// update the swarm service
	if _, err := docker.ServiceUpdate(context.Background(), swarmService, serviceV, spec, swarm.ServiceUpdateOptions{}); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to update swarm service"})
	}

	// update the branch table
	if err := q.SetDomianAndPortForBranch(h.qCtx, db.SetDomianAndPortForBranchParams{
		Domain: b.Domain,
		Port:   b.Port,
		ID:     b.BranchID,
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to update domain and port"})
	}

	return c.JSON(http.StatusOK, types.Res[struct{}]{Message: "Successfully updated domain and port"})
}

// update domain and port
//
// route: PUT /api/service/app/env
func (h *ServiceHandler) UpdateAppServiceEnv(c *echo.Context) error {
	b := new(UpdateEnvReq)
	q := h.Server.DB.Queries
	docker := h.Server.Docker.Client

	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	// get all swarm service of avalable branches
	swarmServices, err := q.GetAllSwarmServiceByAppServiceId(h.qCtx, b.ServiceID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to get all swarm service"})
	}

	// update env in all the service
	for _, serviceName := range swarmServices {
		// get service spec to update the labels
		inspectRes, _, err := docker.ServiceInspectWithRaw(context.Background(), serviceName, swarm.ServiceInspectOptions{})
		if err != nil {
			return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to inspect swarm service"})
		}
		serviceV := inspectRes.Version
		spec := inspectRes.Spec
		spec.TaskTemplate.ContainerSpec.Env = b.Env

		// update the swarm service
		if _, err := docker.ServiceUpdate(context.Background(), serviceName, serviceV, spec, swarm.ServiceUpdateOptions{}); err != nil {
			return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to update swarm service"})
		}
	}

	// clear the evnironment array
	b.Env = cleanArray(b.Env)
	b.BuildArgs = cleanArray(b.BuildArgs)
	b.BuildSecrets = cleanArray(b.BuildSecrets)

	// convert into bytes
	envBytes, err := MarshalServiceEnv(&ServiceEnvArray{
		Env:          b.Env,
		BuildArgs:    b.BuildArgs,
		BuildSecrets: b.BuildSecrets,
	})
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "invalid env values"})
	}

	// update the env in the app service table
	if err := q.UpdateAppServiceEnv(h.qCtx, db.UpdateAppServiceEnvParams{
		ID:           b.ServiceID,
		Env:          envBytes.Env,
		BuildArgs:    envBytes.BuildArgs,
		BuildSecrets: envBytes.BuildSecrets,
	}); err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "Failed to update env"})
	}

	return c.JSON(http.StatusOK, types.Res[struct{}]{Message: "Successfully updated env"})
}

// scale app service
//
// route: POST /api/service/app/scale
func (h *ServiceHandler) ScaleAppService(c *echo.Context) error {
	b := new(ScaleAppServiceReq)
	q := h.Server.DB.Queries
	docker := h.Server.Docker.Client

	swarmName, err := q.GetDefaultBranchSwarmService(h.qCtx, b.ServiceId)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "error getting default branch service"})
	}

	swarmService, _, err := docker.ServiceInspectWithRaw(context.Background(), swarmName, swarm.ServiceInspectOptions{})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "error inspecting swarm service"})
	}
	version := swarmService.Version
	spec := swarmService.Spec

	spec.Mode.Replicated.Replicas = &b.Replicas

	if _, err := docker.ServiceUpdate(context.Background(), swarmName, version, spec, swarm.ServiceUpdateOptions{}); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "error updating the swarm service"})
	}

	return c.JSON(http.StatusOK, types.Res[struct{}]{Message: "successfully updated the replicas"})
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
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to get deployments"})
	}

	// arrange all ids and imgs sepratly for easy access
	dIDs := make([]uuid.UUID, len(serviceInfo))
	imgs := make([]string, len(serviceInfo))
	swarmServiceNames := make(map[string]struct{})
	for i, s := range serviceInfo {
		dIDs[i] = s.DeploymentID
		if s.Image.Valid {
			imgs[i] = s.Image.String
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
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to delete service"})
	}

	return c.JSON(http.StatusOK, types.Res[struct{}]{Message: "Successsfully deleted service"})
}
