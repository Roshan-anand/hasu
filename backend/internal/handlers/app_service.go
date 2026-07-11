package handlers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Roshan-anand/godploy/internal/db"
	deployjob "github.com/Roshan-anand/godploy/internal/jobs/deployment"
	"github.com/Roshan-anand/godploy/internal/lib/docker"
	"github.com/Roshan-anand/godploy/internal/lib/security"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/Roshan-anand/godploy/internal/lib/utils"
	"github.com/docker/docker/api/types/swarm"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

type DockerBuildReq struct {
	FilePath    string `json:"file_path"`
	ContextPath string `json:"context_path"`
	BuildStage  string `json:"build_stage"`
}

type CreateAppServiceReq struct {
	InstanceID    uuid.UUID         `json:"instance_id" validate:"required"`
	Name          string            `json:"name" validate:"required,min=3,max=50"`
	GitProvider   types.GitProvider `json:"git_provider" validate:"required"`
	GhAppID       int64             `json:"gh_app_id" validate:"required"`
	GhRepoID      int64             `json:"gh_repo_id" validate:"required"`
	DefaultBranch string            `json:"default_branch" validate:"required"`
	BuildPath     string            `json:"build_path" validate:"required"`
	WatchPath     string            `json:"watch_path" validate:"required"`
	Env           []string          `json:"env"`
	BuildSecrets  []string          `json:"build_secrets"`
	DockerBuild   *DockerBuildReq   `json:"docker_build"`
	Public        bool              `json:"public"`
	Port          int32             `json:"port"`
}

type CreatePreviewAppServiceReq struct {
	ServiceID uuid.UUID `json:"service_id" validate:"required"`
	Branch    string    `json:"branch" validate:"required"`
}

type UpdateDomainReq struct {
	ServiceID uuid.UUID `json:"service_id" validate:"required"`
	Domain    string    `json:"domain"`
	Port      int32     `json:"port"`
	IsPublic  bool      `json:"is_public"`
}

type UpdateEnvReq struct {
	ServiceID    uuid.UUID `json:"service_id" validate:"required"`
	Env          []string  `json:"env" validate:"required"`
	BuildSecrets []string  `json:"build_secrets" validate:"required"`
}

type GetEnvRes struct {
	Env          []string `json:"env" validate:"required"`
	BuildSecrets []string `json:"build_secrets" validate:"required"`
}

type GetAppServiceByIdRes struct {
	ID           uuid.UUID              `json:"id"`
	Name         string                 `json:"name"`
	GhRepoName   string                 `json:"gh_repo_name"`
	GhRepoUrl    string                 `json:"gh_repo_url"`
	IsPublic     bool                   `json:"is_public"`
	Branch       string                 `json:"branch"`
	SwarmService string                 `json:"swarm_service"`
	Domain       string                 `json:"domain"`
	InternalUrl  string                 `json:"internal_url"`
	Port         int32                  `json:"port"`
	CreatedAt    time.Time              `json:"created_at"`
	Replicas     int32                  `json:"replicas"`
	Status       types.DeploymentStatus `json:"status"`
	CommitMsg    string                 `json:"commit_msg"`
}

type AppServiceSettingsRes struct {
	Domain            string `json:"domain"`
	Port              int32  `json:"port"`
	IsPublic          bool   `json:"is_public"`
	Replicas          int32  `json:"replicas"`
	BuildPath         string `json:"build_path"`
	WatchPath         string `json:"watch_path"`
	DockerFilepath    string `json:"docker_filepath"`
	DockerContextpath string `json:"docker_contextpath"`
	DockerBuildstage  string `json:"docker_buildstage"`
}

type UpdateAppServiceBuildSettingsReq struct {
	ServiceID         uuid.UUID `json:"service_id" validate:"required"`
	BuildPath         string    `json:"build_path"`
	WatchPath         string    `json:"watch_path"`
	DockerFilepath    string    `json:"docker_filepath"`
	DockerContextpath string    `json:"docker_contextpath"`
	DockerBuildstage  string    `json:"docker_buildstage"`
}

type ScaleAppServiceReq struct {
	ServiceId uuid.UUID `json:"service_id" validate:"required"`
	Replicas  uint64    `json:"replicas" validate:"required"`
}

type PauseAppServiceReq struct {
	ServiceID uuid.UUID `json:"service_id" validate:"required"`
}

type ResumeAppServiceReq struct {
	ServiceID uuid.UUID `json:"service_id" validate:"required"`
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
		Name:       b.Name,
		InstanceID: b.InstanceID,
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to check service name"})
	} else if exists {
		return c.JSON(http.StatusConflict, types.Res[struct{}]{Message: "Service name already exists"})
	}

	ghData, err := GetGitHubDeployData(q, b.GhAppID, b.GhRepoID, b.DefaultBranch)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "failed to fetch github data"})
	}

	url, err := utils.GetUrltHostNPath(ghData.RepoURL)
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "invalid repository url"})
	}

	// used as unique image and service name
	unique := docker.GenerateServiceAndImgName(b.Name, b.DefaultBranch)

	// clear the evnironment array
	b.Env = utils.CleanArray(b.Env)
	b.BuildSecrets = utils.CleanArray(b.BuildSecrets)

	// convert into bytes
	envByte, err := utils.MarshalServiceEnv(&utils.ServiceEnvArray{
		Env:          b.Env,
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

	// create app internal url (accessible within the same Docker network)
	internalURL := fmt.Sprintf("http://%s:%d", unique.ServiceName, b.Port)

	// create a new service
	service, err := tq.CreateAppService(h.qCtx, db.CreateAppServiceParams{
		ID:                security.GeneratePrimaryKey(),
		InstanceID:        b.InstanceID,
		Type:              types.AppServiceType,
		Name:              b.Name,
		GitProvider:       b.GitProvider,
		GhAppID:           b.GhAppID,
		GhRepoID:          b.GhRepoID,
		GhRepoName:        ghData.RepoFullName,
		GhRepoUrl:         url,
		BuildPath:         b.BuildPath,
		WatchPath:         b.WatchPath,
		Env:               envByte.Env,
		BuildSecrets:      envByte.BuildSecrets,
		DockerFilepath:    b.DockerBuild.FilePath,
		DockerContextpath: b.DockerBuild.ContextPath,
		DockerBuildstage:  b.DockerBuild.BuildStage,
		IsPublic:          b.Public,
		Branch:            b.DefaultBranch,
		SwarmService:      unique.ServiceName,
		Port:              b.Port,
		InternalUrl:       internalURL,
	})
	if err != nil {
		tx.Rollback()
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to create service"})
	}

	// create a new deployment for the app service
	dID, err := tq.CreateDeployment(h.qCtx, db.CreateDeploymentParams{
		ID:         security.GeneratePrimaryKey(),
		ServiceID:  service.ID,
		CommitHash: ghData.CommitHash,
		CommitMsg:  ghData.CommitMsg,
		IsCurrent:  false,
	})
	if err != nil {
		tx.Rollback()
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to create deployment"})
	}

	if err := tx.Commit(); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to create service"})
	}

	// push a new deployment job to the queue
	if err := h.Server.Services.Deployment.AssignDeploy(context.Background(), &deployjob.DeploymentServiceParams{
		DeploymentID:      dID,
		InstanceID:        b.InstanceID,
		ServiceID:         service.ID,
		Token:             ghData.Token,
		Url:               url,
		Branch:            b.DefaultBranch,
		SwarmService:      unique.ServiceName,
		BuildPath:         b.BuildPath,
		DockerFilePath:    b.DockerBuild.FilePath,
		DockerContextPath: b.DockerBuild.ContextPath,
		DockerBuildStage:  b.DockerBuild.BuildStage,
		ImgName:           unique.ServiceName,
		Env:               b.Env,
		BuildSecrets:      b.BuildSecrets,
		IsPublic:          b.Public,
		GitProvider:       b.GitProvider,
	}, nil); err != nil {
		fmt.Println("error assigning deploy job:", err)
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to create deployment"})
	}

	return c.JSON(http.StatusOK, types.Res[db.CreateAppServiceRow]{
		Message: "",
		Data:    service,
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
		if errors.Is(err, sql.ErrNoRows) {
			fmt.Println("service not found ", err)
			return c.JSON(http.StatusNotFound, types.Res[struct{}]{Message: "service not found"})
		}
		fmt.Println("error getting service by id:", err)
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to get service"})
	}

	// get live replica count from the Docker swarm service spec
	replicas := int32(0)
	swarmService, _, err := h.Server.Docker.Client.ServiceInspectWithRaw(context.Background(), service.SwarmService, swarm.ServiceInspectOptions{})
	if err == nil && swarmService.Spec.Mode.Replicated != nil && swarmService.Spec.Mode.Replicated.Replicas != nil {
		replicas = int32(*swarmService.Spec.Mode.Replicated.Replicas)
	}

	return c.JSON(http.StatusOK, types.Res[GetAppServiceByIdRes]{
		Message: "",
		Data: GetAppServiceByIdRes{
			ID:           service.ID,
			Name:         service.Name,
			GhRepoName:   service.GhRepoName,
			GhRepoUrl:    service.GhRepoUrl,
			IsPublic:     service.IsPublic,
			Branch:       service.Branch,
			SwarmService: service.SwarmService,
			Domain:       service.Domain.String,
			InternalUrl:  service.InternalUrl,
			Port:         service.Port,
			CreatedAt:    service.CreatedAt,
			Replicas:     replicas,
			Status:       service.Status,
			CommitMsg:    service.CommitMsg.String,
		},
	})
}

// get domain and port of the service
//
// route: GET /api/service/app/domain?service_id
func (h *ServiceHandler) GetDomainPort(c *echo.Context) error {
	q := h.Server.DB.Queries

	serviceID, err := uuid.Parse(c.QueryParam("service_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "invalid service_id"})
	}

	branches, err := q.GetDomainAndPortByServiceId(h.qCtx, serviceID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to get branch domain"})
	}

	return c.JSON(http.StatusOK, types.Res[db.GetDomainAndPortByServiceIdRow]{
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

	envString, err := utils.UnmarshalServiceEnv(&utils.ServiceEnvByte{
		Env:          e.Env,
		BuildSecrets: e.BuildSecrets,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to get branch domain"})
	}

	return c.JSON(http.StatusOK, types.Res[GetEnvRes]{
		Message: "",
		Data: GetEnvRes{
			Env:          envString.Env,
			BuildSecrets: envString.BuildSecrets,
		},
	})
}

// update domain, port and visibility
//
// route: PUT /api/service/app/domain
func (h *ServiceHandler) UpdateAppServiceDomain(c *echo.Context) error {
	b := new(UpdateDomainReq)
	q := h.Server.DB.Queries
	docker := h.Server.Docker.Client

	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	// when making private, clear the domain
	if !b.IsPublic {
		b.Domain = ""
	}

	// validate domain only when public
	if b.IsPublic && b.Domain != "" {
		if !strings.HasPrefix(b.Domain, "https://") {
			b.Domain = "https://" + b.Domain
		}
		u, err := url.Parse(b.Domain)
		if err != nil || u.Hostname() == "" {
			fmt.Println("host name:", u.Hostname())
			fmt.Println("host :", u.Host)
			fmt.Println("paht :", u.Path)

			return c.JSON(http.StatusBadRequest,
				types.Res[struct{}]{Message: "invalid domain"})
		}

		b.Domain = u.Host
	}

	swarmService, err := q.GetSwarmServiceByServiceId(h.qCtx, b.ServiceID)
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

	// set traefik enable based on visibility (preserves routing config, just toggles on/off)
	spec.Annotations.Labels["traefik.enable"] = "false"
	if b.IsPublic {
		spec.Annotations.Labels["traefik.enable"] = "true"
	}

	spec.Annotations.Labels[fmt.Sprintf("traefik.http.routers.%s.rule", swarmService)] = fmt.Sprintf("Host(`%s`)", b.Domain)
	spec.Annotations.Labels[fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port", swarmService)] = fmt.Sprintf("%d", b.Port)

	// update the swarm service
	if _, err := docker.ServiceUpdate(context.Background(), swarmService, serviceV, spec, swarm.ServiceUpdateOptions{}); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to update swarm service"})
	}

	// update the app service in database
	if err := q.UpdateDomianAndPort(h.qCtx, db.UpdateDomianAndPortParams{
		Domain:    sql.NullString{String: b.Domain, Valid: b.Domain != ""},
		Port:      b.Port,
		IsPublic:  b.IsPublic,
		ServiceID: b.ServiceID,
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to update domain and port"})
	}

	return c.JSON(http.StatusOK, types.Res[struct{}]{Message: "Successfully updated domain and port"})
}

// update env
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
	swarmService, err := q.GetSwarmServiceByServiceId(h.qCtx, b.ServiceID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to get all swarm service"})
	}

	// update env of the service
	inspectRes, _, err := docker.ServiceInspectWithRaw(context.Background(), swarmService, swarm.ServiceInspectOptions{})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to inspect swarm service"})
	}
	serviceV := inspectRes.Version
	spec := inspectRes.Spec

	// clear the evnironment array
	b.Env = utils.CleanArray(b.Env)
	b.BuildSecrets = utils.CleanArray(b.BuildSecrets)

	// convert into bytes
	envBytes, err := utils.MarshalServiceEnv(&utils.ServiceEnvArray{
		Env:          b.Env,
		BuildSecrets: b.BuildSecrets,
	})
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "invalid env values"})
	}

	// update the env in the app service table
	if err := q.UpdateAppServiceEnv(h.qCtx, db.UpdateAppServiceEnvParams{
		ID:           b.ServiceID,
		Env:          envBytes.Env,
		BuildSecrets: envBytes.BuildSecrets,
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to update env"})
	}

	// Resolve dependency values.
	resolvedEnv := deployjob.MergeDependencyEnv(q, b.ServiceID, b.Env)
	spec.TaskTemplate.ContainerSpec.Env = resolvedEnv

	// update the swarm service
	if _, err := docker.ServiceUpdate(context.Background(), swarmService, serviceV, spec, swarm.ServiceUpdateOptions{}); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to update swarm service"})
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

	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	// check if service is part of production instance
	if isProduction, err := q.CheckServiceIsProduction(h.qCtx, b.ServiceId); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "error checking service instance"})
	} else if !isProduction {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "service is not part of production instance"})
	}

	swarmName, err := q.GetSwarmServiceByServiceId(h.qCtx, b.ServiceId)
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

// get domain, port, visibility, replica count
//
// route: GET /api/service/app/settings?service_id=
func (h *ServiceHandler) GetAppServiceSettings(c *echo.Context) error {
	q := h.Server.DB.Queries

	serviceID, err := uuid.Parse(c.QueryParam("service_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "invalid service_id"})
	}

	settings, err := q.GetAppServiceSettings(h.qCtx, serviceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.JSON(http.StatusNotFound, types.Res[struct{}]{Message: "service not found"})
		}
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "failed to get settings"})
	}

	// get live replica count from the Docker swarm service spec
	replicas := int32(0)
	swarmName, err := q.GetSwarmServiceByServiceId(h.qCtx, serviceID)
	if err == nil {
		swarmService, _, err := h.Server.Docker.Client.ServiceInspectWithRaw(context.Background(), swarmName, swarm.ServiceInspectOptions{})
		if err == nil && swarmService.Spec.Mode.Replicated != nil && swarmService.Spec.Mode.Replicated.Replicas != nil {
			replicas = int32(*swarmService.Spec.Mode.Replicated.Replicas)
		}
	}

	return c.JSON(http.StatusOK, types.Res[AppServiceSettingsRes]{
		Message: "",
		Data: AppServiceSettingsRes{
			Domain:            settings.Domain.String,
			Port:              settings.Port,
			IsPublic:          settings.IsPublic,
			Replicas:          replicas,
			BuildPath:         settings.BuildPath,
			WatchPath:         settings.WatchPath,
			DockerFilepath:    settings.DockerFilepath,
			DockerContextpath: settings.DockerContextpath,
			DockerBuildstage:  settings.DockerBuildstage,
		},
	})
}

// UpdateAppServiceBuildSettings — updates build-related settings (build_path, watch_path, docker filepath/contextpath/buildstage)
//
// route: PUT /api/service/app/settings
func (h *ServiceHandler) UpdateAppServiceBuildSettings(c *echo.Context) error {
	b := new(UpdateAppServiceBuildSettingsReq)

	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	if err := h.Server.DB.Queries.UpdateAppServiceBuildSettings(h.qCtx, db.UpdateAppServiceBuildSettingsParams{
		BuildPath:         b.BuildPath,
		WatchPath:         b.WatchPath,
		DockerFilepath:    b.DockerFilepath,
		DockerContextpath: b.DockerContextpath,
		DockerBuildstage:  b.DockerBuildstage,
		ID:                b.ServiceID,
	}); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.JSON(http.StatusNotFound, types.Res[struct{}]{Message: "service not found"})
		}
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "failed to update build settings"})
	}

	return c.JSON(http.StatusOK, types.Res[struct{}]{Message: "build settings updated"})
}

// PauseAppService — sets swarm replicas to 0, marks current deployment as paused
//
// route: POST /api/service/app/pause
func (h *ServiceHandler) PauseAppService(c *echo.Context) error {
	b := new(PauseAppServiceReq)
	q := h.Server.DB.Queries
	docker := h.Server.Docker.Client

	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	currentDep, err := q.GetCurrentDeploymentByServiceId(h.qCtx, b.ServiceID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "error getting current deployment"})
	}

	swarmName, err := q.GetSwarmServiceByServiceId(h.qCtx, b.ServiceID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "error getting swarm service"})
	}

	swarmService, _, err := docker.ServiceInspectWithRaw(context.Background(), swarmName, swarm.ServiceInspectOptions{})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "error inspecting swarm service"})
	}
	version := swarmService.Version
	spec := swarmService.Spec

	zero := uint64(0)
	spec.Mode.Replicated.Replicas = &zero

	if _, err := docker.ServiceUpdate(context.Background(), swarmName, version, spec, swarm.ServiceUpdateOptions{}); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "error pausing swarm service"})
	}

	if err := q.UpdateDeploymentStatus(h.qCtx, db.UpdateDeploymentStatusParams{
		Status: types.DeploymentPaused,
		ID:     currentDep.ID,
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "error persisting paused status"})
	}

	return c.JSON(http.StatusOK, types.Res[struct{}]{Message: "service paused"})
}

// restore replicas to 1, mark deployment as ready
//
// route: POST /api/service/app/resume
func (h *ServiceHandler) ResumeAppService(c *echo.Context) error {
	b := new(ResumeAppServiceReq)
	q := h.Server.DB.Queries
	docker := h.Server.Docker.Client

	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	currentDep, err := q.GetCurrentDeploymentByServiceId(h.qCtx, b.ServiceID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "error getting current deployment"})
	}

	swarmName, err := q.GetSwarmServiceByServiceId(h.qCtx, b.ServiceID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "error getting swarm service"})
	}

	swarmService, _, err := docker.ServiceInspectWithRaw(context.Background(), swarmName, swarm.ServiceInspectOptions{})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "error inspecting swarm service"})
	}
	version := swarmService.Version
	spec := swarmService.Spec

	resumeReplicas := uint64(1)
	spec.Mode.Replicated.Replicas = &resumeReplicas

	if _, err := docker.ServiceUpdate(context.Background(), swarmName, version, spec, swarm.ServiceUpdateOptions{}); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "error resuming swarm service"})
	}

	if err := q.UpdateDeploymentStatus(h.qCtx, db.UpdateDeploymentStatusParams{
		Status: types.DeploymentReady,
		ID:     currentDep.ID,
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "error persisting running status"})
	}

	return c.JSON(http.StatusOK, types.Res[struct{}]{Message: "service resumed"})
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

	serviceInfo, err := q.GetAllSwarmServiceAndImgByAppServiceId(h.qCtx, b.ServiceId)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to get deployments"})
	}

	// arrange all ids and imgs sepratly for easy access
	dIDs := make([]uuid.UUID, len(serviceInfo))
	imgs := make([]string, len(serviceInfo))
	SwarmServices := make(map[string]struct{})

	// because all the deployment image have same parent swarm service
	SwarmServices[serviceInfo[0].SwarmService] = struct{}{}

	for i, s := range serviceInfo {
		dIDs[i] = s.DeploymentID
		if s.Image.Valid {
			imgs[i] = s.Image.String
		}
	}

	// stop all the services running and remove all the images
	go func() {
		h.Server.Docker.RemoveServices(SwarmServices)
		h.Server.Docker.RemoveImages(imgs)
	}()

	// delete all logs related to the service deployments
	go h.Server.BadgerDB.DeleteAllLogsByDeploymentID(dIDs)

	// clean up incoming dependency records where this service is the target
	if err := q.DeleteIncomingDependencies(h.qCtx, b.ServiceId); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to clean up dependency records"})
	}

	// delete the app service (outgoing dependencies cascade via FK)
	if err := h.Server.DB.Queries.DeleteAppService(h.qCtx, b.ServiceId); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to delete service"})
	}

	return c.JSON(http.StatusOK, types.Res[struct{}]{Message: "Successsfully deleted service"})
}
