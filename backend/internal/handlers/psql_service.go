package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/Roshan-anand/godploy/internal/config"
	"github.com/Roshan-anand/godploy/internal/db"
	"github.com/Roshan-anand/godploy/internal/lib/security"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
)

type ServiceReq struct {
	ServiceId uuid.UUID `json:"service_id" validate:"required"`
}

type PsqlServiceHandler struct {
	Server   *config.Server
	Validate *validator.Validate
	qCtx     context.Context
}

type CreatePsqlServiceReq struct {
	OrgID      uuid.UUID `json:"org_id" validate:"required"`
	Name       string    `json:"name" validate:"required"`
	DbName     string    `json:"db_name" validate:"required"`
	DbUser     string    `json:"db_user" validate:"required"`
	DbPassword string    `json:"db_password" validate:"required"`
	Image      string    `json:"image" validate:"required"`
}

func InitPsqlServiceHandlers(s *config.Server) *PsqlServiceHandler {
	return &PsqlServiceHandler{
		Server:   s,
		Validate: validator.New(),
		qCtx:     context.Background(),
	}
}

// create a new psql service
//
// route: POST /api/service/psql
func (h *ServiceHandler) CreatePsqlService(c *echo.Context) error {
	q := h.Server.DB.Queries
	b := new(CreatePsqlServiceReq)

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

	serviceName := fmt.Sprintf("%s-%s", b.Name, b.OrgID)

	service, err := h.Server.DB.Queries.CreatePsqlService(h.qCtx, db.CreatePsqlServiceParams{
		ID:               security.GeneratePrimaryKey(),
		OrganizationID:   b.OrgID,
		Type:             types.PsqlServiceType,
		SwarmServiceName: serviceName,
		Name:             b.Name,
		DbName:           b.DbName,
		DbUser:           b.DbUser,
		DbPassword:       b.DbPassword, // TODO : make is hased
		InternalUrl:      "",           // TODO : create internal URl
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res{Message: "Failed to create service"})
	}

	return c.JSON(http.StatusOK, service)
}

// get psql service details by id
//
// route: GET /api/service/psql/:id
func (h *ServiceHandler) GetPsqlServiceById(c *echo.Context) error {
	serviceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res{Message: "invalid service id"})
	}

	service, err := h.Server.DB.Queries.GetPsqlServiceById(h.qCtx, serviceID)
	if err != nil {
		return c.JSON(http.StatusNotFound, types.Res{Message: "service not found"})
	}

	return c.JSON(http.StatusOK, service)
}

// deploy the psql service to docker swarm
//
// route: POST /api/service/psql/deploy
func (h *ServiceHandler) DeployPsqlService(c *echo.Context) error {
	docker := h.Server.Docker.Client
	q := h.Server.DB.Queries

	b := new(ServiceReq)

	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	service, err := q.GetPsqlServiceById(h.qCtx, b.ServiceId)
	if err != nil {
		return c.JSON(http.StatusNotFound, types.Res{Message: "service not found"})
	}

	// create a volume for the service
	vlName := service.SwarmServiceName + "_pgdata"
	docker.VolumeCreate(h.qCtx, client.VolumeCreateOptions{
		Name:   vlName,
		Driver: "local",
	})

	replicas := uint64(2)

	spec := client.ServiceCreateOptions{
		Spec: swarm.ServiceSpec{

			Annotations: swarm.Annotations{
				Name: service.SwarmServiceName,
			},

			TaskTemplate: swarm.TaskSpec{
				ContainerSpec: &swarm.ContainerSpec{
					Image: service.ImageID, // TODO validate if image accepts image_id

					Env: []string{
						"POSTGRES_PASSWORD=" + service.DbPassword,
						"POSTGRES_USER=" + service.DbUser,
						"POSTGRES_DB=" + service.DbName,
					},

					Mounts: []mount.Mount{
						{
							Type:   mount.TypeVolume,
							Source: vlName,
							Target: "/var/lib/postgresql/data",
						},
					},
				},

				RestartPolicy: &swarm.RestartPolicy{
					Condition: swarm.RestartPolicyConditionAny,
				},
			},

			Mode: swarm.ServiceMode{
				Replicated: &swarm.ReplicatedService{
					Replicas: &replicas,
				},
			},
		},
	}

	// depoly the service
	sRes, err := docker.ServiceCreate(h.qCtx, spec)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res{Message: "Failed to deploy service"})
	}

	// update the service ID
	if err := q.SetPsqlSwarmServiceId(h.qCtx, db.SetPsqlSwarmServiceIdParams{
		SwarmServiceID: sql.NullString{
			String: sRes.ID,
			Valid:  true,
		},
		ID: service.ID,
	}); err != nil {
		docker.ServiceRemove(h.qCtx, sRes.ID, client.ServiceRemoveOptions{})
		return c.JSON(http.StatusInternalServerError, types.Res{Message: "Failed to update service with service id"})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"id": sRes.ID,
	})
}

// stop the psql service
//
// route: POST /api/service/psql/stop
func (h *ServiceHandler) StopPsqlService(c *echo.Context) error {
	docker := h.Server.Docker.Client
	q := h.Server.DB.Queries

	b := new(ServiceReq)

	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	service, err := q.GetPsqlServiceById(h.qCtx, b.ServiceId)
	if err != nil {
		return c.JSON(http.StatusNotFound, types.Res{Message: "service not found"})
	}

	if _, err := docker.ServiceRemove(h.qCtx, service.SwarmServiceID.String, client.ServiceRemoveOptions{}); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res{Message: "error removing service"})
	}

	return c.JSON(http.StatusOK, types.Res{Message: "successfully removed the service"})
}

// stops and delete the psql service
//
// route: DELETE /api/service/psql
func (h *ServiceHandler) DeletePsqlService(c *echo.Context) error {
	docker := h.Server.Docker.Client
	q := h.Server.DB.Queries

	b := new(ServiceReq)

	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	service, err := q.GetPsqlServiceById(h.qCtx, b.ServiceId)
	if err != nil {
		return c.JSON(http.StatusConflict, types.Res{Message: "Failed to fetch service details"})
	}

	// check and stop the service if it is running
	if s, _ := docker.ServiceInspect(h.qCtx, service.SwarmServiceID.String, client.ServiceInspectOptions{}); s.Service.ID != "" {
		if _, err := docker.ServiceRemove(h.qCtx, service.SwarmServiceID.String, client.ServiceRemoveOptions{}); err != nil {
			return c.JSON(http.StatusInternalServerError, types.Res{Message: fmt.Sprintln("error removing service :", err)})
		}
	}

	if err := q.DeletePsqlService(h.qCtx, b.ServiceId); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res{Message: "Failed to create service"})
	}

	return c.JSON(http.StatusOK, types.Res{
		Message: "Successsfully deleted service",
	})
}
