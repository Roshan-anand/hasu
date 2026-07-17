package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Roshan-anand/godploy/internal/config"
	"github.com/Roshan-anand/godploy/internal/db"
	"github.com/Roshan-anand/godploy/internal/lib/auth"
	predef "github.com/Roshan-anand/godploy/internal/lib/predef_utils"
	"github.com/Roshan-anand/godploy/internal/lib/security"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

type CreateRedisServiceReq struct {
	InstanceID uuid.UUID `json:"instance_id" validate:"required"`
	Name       string    `json:"name" validate:"required"`
	Password   string    `json:"password"`
	Image      string    `json:"image" validate:"required"`
	Volume     string    `json:"volume"`
}

type UpdateRedisServiceReq struct {
	ServiceID uuid.UUID `json:"service_id" validate:"required"`
	Password  string    `json:"password"`
}

type DeleteRedisServiceReq struct {
	ServiceId uuid.UUID `json:"service_id" validate:"required"`
	KeepData  bool      `json:"keep_data"`
}

type RedisServiceHandler struct {
	Server   *config.Server
	Validate *validator.Validate
	qCtx     context.Context
}

func InitRedisServiceHandlers(s *config.Server) *RedisServiceHandler {
	return &RedisServiceHandler{
		Server:   s,
		Validate: validator.New(),
		qCtx:     context.Background(),
	}
}

func isRedisImage(image string) bool {
	return strings.Contains(image, "redis")
}

// create a new redis service
//
// route: POST /api/service/redis
func (h *ServiceHandler) CreateRedisService(c *echo.Context) error {
	u := c.Get(h.Server.Config.EchoCtxUserKey).(auth.AuthUser)
	q := h.Server.DB.Queries
	b := new(CreateRedisServiceReq)
	docker := h.Server.Docker.Client

	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	// check if image is a redis image
	if !isRedisImage(b.Image) {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "Invalid image. Only redis images are allowed"})
	}

	// check if service name already exists
	if exists, err := q.ServiceNameExists(h.qCtx, db.ServiceNameExistsParams{
		InstanceID: b.InstanceID,
		Name:       b.Name,
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to check service name"})
	} else if exists {
		return c.JSON(http.StatusConflict, types.Res[struct{}]{Message: "Service name already exists"})
	}

	org, err := q.GetCurrentOrg(h.qCtx, u.Email)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to fetch organization id"})
	}

	serviceName := fmt.Sprintf("%s-%s", b.Name, security.GenerateRandomID(3, true))

	// if user selects orphan volume then claim it or else create a new volume
	var volumeName string
	if b.Volume == "" {
		volumeName, err = predef.CreatePredefVolume(h.qCtx, serviceName, docker, predef.RedisVol)
		if err != nil {
			fmt.Println("err :", err)
			return fmt.Errorf("failed to create redis volume: %w", err)
		}
	} else {
		row, err := q.ClaimOrphanVolume(h.qCtx, db.ClaimOrphanVolumeParams{
			Volume:         b.Volume,
			OrganizationID: org.ID,
		})
		if err != nil {
			return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to claim orphan volume"})
		}

		if row == 0 {
			return c.JSON(http.StatusConflict, types.Res[struct{}]{Message: "Volume is not available"})
		}

		volumeName = b.Volume
	}

	network, err := q.GetInstanceNetwork(h.qCtx, b.InstanceID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to fetch project network"})
	}

	env := []string{}
	if b.Password != "" {
		env = append(env, "REDIS_PASSWORD="+b.Password)
	}

	if err := predef.DeployPredefService(h.qCtx, h.Server.Docker, network, serviceName, b.Image, env, volumeName, predef.RedisMountTarget); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to deploy redis service"})
	}

	internalURL := predef.BuildRedisInternalURL(b.Password, serviceName)

	service, err := q.CreateRedisService(h.qCtx, db.CreateRedisServiceParams{
		ID:           security.GeneratePrimaryKey(),
		InstanceID:   b.InstanceID,
		Type:         types.RedisServiceType,
		Status:       types.PredefServiceRunning,
		SwarmService: serviceName,
		Name:         b.Name,
		Password:     b.Password,
		InternalUrl:  internalURL,
		Image:        b.Image,
		Volume:       volumeName,
	})
	if err != nil {
		fmt.Println("error creating service in db :", err)
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to create service"})
	}

	return c.JSON(http.StatusOK, types.Res[db.CreateRedisServiceRow]{
		Message: "",
		Data:    service,
	})
}

// get redis service details by id
//
// route: GET /api/service/redis/:id
func (h *ServiceHandler) GetRedisServiceById(c *echo.Context) error {
	serviceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "invalid service id"})
	}

	service, err := h.Server.DB.Queries.GetRedisServiceById(h.qCtx, serviceID)
	if err != nil {
		return c.JSON(http.StatusNotFound, types.Res[struct{}]{Message: "service not found"})
	}

	return c.JSON(http.StatusOK, types.Res[db.GetRedisServiceByIdRow]{
		Message: "",
		Data:    service,
	})
}

// update redis service details
//
// route: PUT /api/service/redis
func (h *ServiceHandler) UpdateRedisServiceDetails(c *echo.Context) error {
	q := h.Server.DB.Queries

	b := new(UpdateRedisServiceReq)
	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	service, err := q.GetRedisServiceById(h.qCtx, b.ServiceID)
	if err != nil {
		return c.JSON(http.StatusNotFound, types.Res[struct{}]{Message: "service not found"})
	}

	internalURL := predef.BuildRedisInternalURL(b.Password, service.SwarmService)

	if err := q.UpdateRedisServiceDetails(h.qCtx, db.UpdateRedisServiceDetailsParams{
		Password:    b.Password,
		InternalUrl: internalURL,
		ID:          b.ServiceID,
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to update service details"})
	}

	return c.JSON(http.StatusOK, types.Res[struct{}]{Message: "Successfully updated service details"})
}

// redeploy redis service with saved details
//
// route: POST /api/service/redis/redeploy
func (h *ServiceHandler) RedeployRedisService(c *echo.Context) error {
	q := h.Server.DB.Queries
	docker := h.Server.Docker.Client

	b := new(ServiceReq)
	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	service, err := q.GetRedisServiceById(h.qCtx, b.ServiceId)
	if err != nil {
		return c.JSON(http.StatusNotFound, types.Res[struct{}]{Message: "service not found"})
	}

	inspectRes, _, err := docker.ServiceInspectWithRaw(h.qCtx, service.SwarmService, swarm.ServiceInspectOptions{})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to inspect swarm service"})
	}

	serviceV := inspectRes.Version
	spec := inspectRes.Spec
	if spec.TaskTemplate.ContainerSpec == nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Missing container spec"})
	}

	spec.TaskTemplate.ContainerSpec.Env = []string{}
	if service.Password != "" {
		spec.TaskTemplate.ContainerSpec.Env = append(spec.TaskTemplate.ContainerSpec.Env, "REDIS_PASSWORD="+service.Password)
	}

	if _, err := docker.ServiceUpdate(h.qCtx, service.SwarmService, serviceV, spec, swarm.ServiceUpdateOptions{}); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to redeploy service"})
	}

	return c.JSON(http.StatusOK, types.Res[struct{}]{Message: "Successfully started redeploy"})
}

// stops and delete the redis service
//
// route: DELETE /api/service/redis
func (h *ServiceHandler) DeleteRedisService(c *echo.Context) error {
	docker := h.Server.Docker.Client
	q := h.Server.DB.Queries

	b := new(DeleteRedisServiceReq)

	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	service, err := q.GetRedisServiceById(h.qCtx, b.ServiceId)
	if err != nil {
		return c.JSON(http.StatusConflict, types.Res[struct{}]{Message: "Failed to fetch service details"})
	}

	// check and stop the service if it is running
	if s, _, _ := docker.ServiceInspectWithRaw(h.qCtx, service.SwarmService, swarm.ServiceInspectOptions{}); s.ID != "" {
		if err := docker.ServiceRemove(h.qCtx, service.SwarmService); err != nil {
			return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: fmt.Sprintln("error removing service :", err)})
		}
	}

	tx, err := h.Server.DB.Pool.BeginTx(context.Background(), nil)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to start transaction"})
	}
	tq := q.WithTx(tx)

	// create an orphan volume record if user wants to keep data
	if b.KeepData {
		if err := tq.CreateOrphanVolume(h.qCtx, db.CreateOrphanVolumeParams{
			ID:             security.GeneratePrimaryKey(),
			OrganizationID: service.OrganizationID,
			Volume:         service.Volume,
			Type:           types.RedisPredefServiceType,
		}); err != nil {
			tx.Rollback()
			return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to create orphan volume record"})
		}
	} else {
		go func(volumeName string) {
			// incase service removal task is still in progress so retry every 7 sec to remove the volume
			for i := 0; i < 10; i++ {
				time.Sleep(8 * time.Second)
				if err := docker.VolumeRemove(context.Background(), volumeName, true); err != nil {
					fmt.Println("error removing volume in background:", err)
				} else {
					fmt.Println("successfully removed volume in background:", volumeName)
					break
				}
			}
		}(service.Volume)
	}

	// clean up incoming dependency records where this service is the target
	if err := tq.DeleteIncomingDependencies(h.qCtx, b.ServiceId); err != nil {
		tx.Rollback()
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to clean up dependency records"})
	}

	if err := tq.DeleteRedisService(h.qCtx, b.ServiceId); err != nil {
		tx.Rollback()
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to delete service"})
	}

	if err := tx.Commit(); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to commit transaction"})
	}

	return c.JSON(http.StatusOK, types.Res[struct{}]{
		Message: "Successfully deleted service",
	})
}
