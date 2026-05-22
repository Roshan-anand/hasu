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
	"github.com/Roshan-anand/godploy/internal/lib/gh"
	"github.com/Roshan-anand/godploy/internal/lib/security"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/moby/moby/client"
)

type DockerBuildReq struct {
	FilePath    string `json:"file_path"`
	ContextPath string `json:"context_path"`
	BuildStage  string `json:"build_stage"`
}

type CreateAppServiceReq struct {
	OrgID        uuid.UUID       `json:"org_id" validate:"required"`
	Name         string          `json:"name" validate:"required,min=3,max=50"`
	GitProvider  string          `json:"git_provider" validate:"required"`
	GhAppID      int64           `json:"gh_app_id" validate:"required"`
	GhRepoID     int64           `json:"gh_repo_id" validate:"required"`
	BuildPath    string          `json:"build_path" validate:"required"`
	WatchPath    string          `json:"watch_path" validate:"required"`
	Env          []string        `json:"env"`
	BuildArgs    []string        `json:"build_args"`
	BuildSecrets []string        `json:"build_secrets"`
	DockerBuild  *DockerBuildReq `json:"docker_build"`
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
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to check service name"})
	} else if exists {
		return c.JSON(http.StatusConflict, types.Res[struct{}]{Message: "Service name already exists"})
	}

	// get the github app details
	ghApp, err := q.GetGhAppByAppId(h.qCtx, b.GhAppID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "invalid github app"})
		}
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "failed to verify github app"})
	}

	ghClient, err := gh.CreateGithubClient(context.Background(), ghApp.AppID, ghApp.InstallationID.Int64, ghApp.PemKey)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "failed to create github client"})
	}

	// get the github repository details
	repo, _, err := ghClient.Repositories.GetByID(context.Background(), b.GhRepoID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "failed to fetch repository info from github"})
	}

	repoName := repo.GetFullName()
	repoURL := repo.GetHTMLURL()
	defaultBranch := repo.GetDefaultBranch()
	owner := repo.GetOwner().GetLogin()
	repoShortName := repo.GetName()

	// get the latest commit info of the default branch
	commits, _, err := ghClient.Repositories.ListCommits(context.Background(), owner, repoShortName, &github.CommitsListOptions{
		SHA: defaultBranch,
		ListOptions: github.ListOptions{
			PerPage: 1,
			Page:    1,
		},
	})
	if err != nil || len(commits) == 0 {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "failed to fetch latest commit info from github"})
	}

	latestCommitHash := commits[0].GetSHA()
	latestCommitMsg := commits[0].GetCommit().GetMessage()

	// parse url
	u, err := url.Parse(repoURL)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "failed to parse repository url"})
	}
	url := u.Host + u.Path

	// used as uniquecontainer name and code storing path
	serviceName := fmt.Sprintf("%s-%s-%s", b.Name, defaultBranch, security.GenerateRandomID(6))
	imgName := strings.ToLower(fmt.Sprintf("%s-%s-dyp_%s", b.Name, defaultBranch, security.GenerateRandomID(6)))

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
	q = q.WithTx(tx)

	// create a new service
	service, err := q.CreateAppService(h.qCtx, db.CreateAppServiceParams{
		ID:                security.GeneratePrimaryKey(),
		OrganizationID:    b.OrgID,
		Type:              types.AppServiceType,
		Name:              b.Name,
		GitProvider:       b.GitProvider,
		GhAppID:           ghApp.AppID,
		GhRepoID:          b.GhRepoID,
		GhRepoName:        repoName,
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
	branchID, err := q.CreateAppServiceBranch(h.qCtx, db.CreateAppServiceBranchParams{
		ID:               security.GeneratePrimaryKey(),
		IsDefaultBranch:  true,
		BranchName:       defaultBranch,
		SwarmServiceName: serviceName,
		ServiceID:        service.ID,
	})
	if err != nil {
		tx.Rollback()
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to create service branch"})
	}

	// create a new deployment for the app service
	dID, err := q.CreateDeployment(h.qCtx, db.CreateDeploymentParams{
		ID:         security.GeneratePrimaryKey(),
		BranchID:   branchID,
		CommitHash: latestCommitHash,
		CommitMsg:  latestCommitMsg,
		IsCurrent:  true,
	})
	if err != nil {
		tx.Rollback()
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to create deployment"})
	}

	if err := tx.Commit(); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to create service"})
	}

	// get gh token
	token, err := gh.GetGhToken(ghApp.AppID, ghApp.InstallationID.Int64, ghApp.PemKey)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to get github token"})
	}

	fmt.Println("env :", len(b.Env), b.Env)

	// push a new deployment job to the queue
	h.Server.DeploymentQ.EnqueuePullJob(&deploymentqueue.PullJobData{
		Type:              deploymentqueue.DeployJob,
		DeploymentID:      dID,
		Token:             token,
		Url:               url,
		Branch:            defaultBranch,
		SwarmServiceName:  serviceName,
		BuildPath:         b.BuildPath,
		DockerFilePath:    b.DockerBuild.FilePath,
		DockerContextPath: b.DockerBuild.ContextPath,
		DockerBuildStage:  b.DockerBuild.BuildStage,
		ImgName:           imgName,
		Env:               b.Env,
		BuildArgs:         b.BuildArgs,
		BuildSecrets:      b.BuildSecrets,
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
		fmt.Printf("Failed to get service by id: %v\n", err)
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
	inspectRes, err := docker.ServiceInspect(context.Background(), swarmService, client.ServiceInspectOptions{})
	if err != nil {
		fmt.Printf("Failed to inspect swarm service: %v\n", err)
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to inspect swarm service"})
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
		inspectRes, err := docker.ServiceInspect(context.Background(), serviceName, client.ServiceInspectOptions{})
		if err != nil {
			fmt.Printf("Failed to inspect swarm service: %v\n", err)
			return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to inspect swarm service"})
		}
		serviceV := inspectRes.Service.Meta.Version
		spec := inspectRes.Service.Spec
		spec.TaskTemplate.ContainerSpec.Env = b.Env

		// update the swarm service
		if _, err := docker.ServiceUpdate(context.Background(), serviceName, client.ServiceUpdateOptions{
			Version: serviceV,
			Spec:    spec,
		}); err != nil {
			fmt.Printf("Failed to update swarm service: %v\n", err)
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
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to delete service"})
	}

	return c.JSON(http.StatusOK, types.Res[struct{}]{Message: "Successsfully deleted service"})
}
