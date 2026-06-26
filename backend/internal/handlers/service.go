package handlers

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/Roshan-anand/godploy/internal/config"
	"github.com/Roshan-anand/godploy/internal/db"
	"github.com/Roshan-anand/godploy/internal/lib/sse"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/swarm"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

type ServiceHandler struct {
	Server   *config.Server
	Validate *validator.Validate
	qCtx     context.Context
}

type DeploymentReq struct {
	DeploymentID uuid.UUID `json:"deployment_id" validate:"required"`
}

type GetAllServicesRes struct {
	db.GetAllServiceRow
	Replicas int32 `json:"replicas"`
}

type PredefServiceReq struct {
	ServiceID uuid.UUID `json:"service_id" validate:"required"`
}

func InitServiceHandlers(s *config.Server) *ServiceHandler {
	return &ServiceHandler{
		Server:   s,
		Validate: validator.New(),
		qCtx:     context.Background(),
	}
}

// get all services of a project
//
// route: GET /api/service/all?instace_id=
func (h *ServiceHandler) GetAllServices(c *echo.Context) error {
	q := h.Server.DB.Queries

	instanceID, err := uuid.Parse(c.QueryParam("instance_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "invalid instance_id"})
	}

	// check the status of the instance — only fetch services when ready
	status, err := q.GetInstanceStatus(h.qCtx, instanceID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "invalid instance_id"})
	}

	if status != types.InstanceReady {
		var msg string
		switch status {
		case types.InstanceCreating:
			msg = "cooking"
		case types.InstanceDeleting:
			msg = "deleting"
		default:
			msg = "instance not ready"
		}
		return c.JSON(http.StatusOK, types.Res[[]GetAllServicesRes]{
			Message: msg,
		})
	}

	// get all services of the project
	allServices, err := q.GetAllService(h.qCtx, instanceID)
	if err != nil {
		fmt.Println("error getting all services: ", err)
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "failed to get services"})
	}

	services := make([]GetAllServicesRes, 0, len(allServices))

	for _, s := range allServices {
		service := GetAllServicesRes{
			GetAllServiceRow: s,
		}

		if s.Type == types.AppServiceType {
			// get live replica count from the Docker swarm service spec
			replicas := int32(0)
			swarmService, _, err := h.Server.Docker.Client.ServiceInspectWithRaw(context.Background(), s.SwarmService, swarm.ServiceInspectOptions{})
			if err == nil && swarmService.Spec.Mode.Replicated != nil && swarmService.Spec.Mode.Replicated.Replicas != nil {
				replicas = int32(*swarmService.Spec.Mode.Replicated.Replicas)
			}
			service.Replicas = replicas
		}

		services = append(services, service)
	}

	return c.JSON(http.StatusOK, types.Res[[]GetAllServicesRes]{
		Message: "",
		Data:    services,
	})
}

// get given service ID
//
// route: GET /api/service/:name?instance_id=
func (h *ServiceHandler) GetServiceID(c *echo.Context) error {
	q := h.Server.DB.Queries

	name := c.Param("name")
	if name == "" {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "service name is required"})
	}

	instanceID, err := uuid.Parse(c.QueryParam("instance_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "invalid project_id"})
	}

	// get all services of the project
	serviceID, err := q.GetServiceID(h.qCtx, db.GetServiceIDParams{
		InstanceID: instanceID,
		Name:       name,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "failed to get services"})
	}

	return c.JSON(http.StatusOK, types.Res[uuid.UUID]{
		Message: "",
		Data:    serviceID,
	})
}

// get service logs
//
// route: GET /api/service/logs?service_id
func (h *ServiceHandler) GetServiceLogs(c *echo.Context) error {
	q := h.Server.DB.Queries

	serviceID, err := uuid.Parse(c.QueryParam("service_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "invalid service id"})
	}

	swarmService, err := q.GetSwarmServiceByServiceId(h.qCtx, serviceID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "failed to get swarm service"})
	}

	serviceLogs, err := h.Server.Docker.Client.ServiceLogs(context.Background(), swarmService, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: true,
		Follow:     true,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "failed to get service logs"})
	}
	defer serviceLogs.Close()

	// setup sse headers
	w := c.Response()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	sse := sse.NewSSE(w)

	// TODO : for now to test logs, we have set TTY=true when deploying the service in the settings which results in simple singel output.
	// we hve to think a way to handle TTY=false logs. at that time it send multiplexed output from both stdout and stderr which is better way to show the logs.

	reader := bufio.NewReader(serviceLogs)

	streamErr := make(chan error, 1)

	go func() {
		for {
			line, err := reader.ReadString('\n')
			if len(line) > 0 {
				sse.SendEvent("log", []byte(line))
			}

			if err != nil {
				fmt.Printf("error streaming service logs: %v", err)
				streamErr <- err
				return
			}
		}
	}()

	for {
		select {
		case err := <-streamErr:
			if err != nil {
				return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "failed to stream service logs"})
			}
			return nil

		case <-c.Request().Context().Done():
			log.Printf("SSE client disconnected, ip: %v", c.RealIP())
			return nil
		}
	}
}

// StopPredefService — stops a predefined (PSQL/Redis) service
//
// route: POST /api/service/stop
func (h *ServiceHandler) StopPredefService(c *echo.Context) error {
	b := new(PredefServiceReq)
	q := h.Server.DB.Queries
	docker := h.Server.Docker.Client

	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	service, err := q.GetPredefSwarmServiceById(h.qCtx, b.ServiceID)
	if err != nil {
		return c.JSON(http.StatusNotFound, types.Res[struct{}]{Message: "service not found"})
	}

	// inspect the current swarm service
	swarmService, _, err := docker.ServiceInspectWithRaw(context.Background(), service.SwarmService, swarm.ServiceInspectOptions{})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "error inspecting swarm service"})
	}
	version := swarmService.Version
	spec := swarmService.Spec

	replicas := uint64(0)
	spec.Mode.Replicated.Replicas = &replicas

	if _, err := docker.ServiceUpdate(context.Background(), service.SwarmService, version, spec, swarm.ServiceUpdateOptions{}); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "error stopping swarm service"})
	}

	// update status in both tables — one will be a no-op (no matching id)
	q.UpdatePsqlServiceStatus(h.qCtx, db.UpdatePsqlServiceStatusParams{
		Status: types.PredefServicePaused,
		ID:     b.ServiceID,
	})
	q.UpdateRedisServiceStatus(h.qCtx, db.UpdateRedisServiceStatusParams{
		Status: types.PredefServicePaused,
		ID:     b.ServiceID,
	})

	return c.JSON(http.StatusOK, types.Res[struct{}]{
		Message: "service stopped",
	})
}

// StartPredefService — starts a predefined (PSQL/Redis) service
//
// route: POST /api/service/start
func (h *ServiceHandler) StartPredefService(c *echo.Context) error {
	b := new(PredefServiceReq)
	q := h.Server.DB.Queries
	docker := h.Server.Docker.Client

	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	service, err := q.GetPredefSwarmServiceById(h.qCtx, b.ServiceID)
	if err != nil {
		return c.JSON(http.StatusNotFound, types.Res[struct{}]{Message: "service not found"})
	}

	// inspect the current swarm service
	swarmService, _, err := docker.ServiceInspectWithRaw(context.Background(), service.SwarmService, swarm.ServiceInspectOptions{})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "error inspecting swarm service"})
	}
	version := swarmService.Version
	spec := swarmService.Spec

	replicas := uint64(1)
	spec.Mode.Replicated.Replicas = &replicas

	if _, err := docker.ServiceUpdate(context.Background(), service.SwarmService, version, spec, swarm.ServiceUpdateOptions{}); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "error starting swarm service"})
	}

	// update status in both tables — one will be a no-op (no matching id)
	q.UpdatePsqlServiceStatus(h.qCtx, db.UpdatePsqlServiceStatusParams{
		Status: types.PredefServiceRunning,
		ID:     b.ServiceID,
	})
	q.UpdateRedisServiceStatus(h.qCtx, db.UpdateRedisServiceStatusParams{
		Status: types.PredefServiceRunning,
		ID:     b.ServiceID,
	})

	return c.JSON(http.StatusOK, types.Res[struct{}]{
		Message: "service started",
	})
}
