package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Roshan-anand/godploy/internal/config"
	"github.com/Roshan-anand/godploy/internal/db"
	"github.com/Roshan-anand/godploy/internal/lib/security"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/api/types/volume"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

type ServiceReq struct {
	ServiceId uuid.UUID `json:"service_id" validate:"required"`
}

type CreatePsqlServiceReq struct {
	ProjectID  uuid.UUID `json:"project_id" validate:"required"`
	Name       string    `json:"name" validate:"required"`
	DbName     string    `json:"db_name" validate:"required"`
	DbUser     string    `json:"db_user" validate:"required"`
	DbPassword string    `json:"db_password" validate:"required"`
	Image      string    `json:"image" validate:"required"`
}

type UpdatePsqlServiceReq struct {
	ServiceID  uuid.UUID `json:"service_id" validate:"required"`
	DbName     string    `json:"db_name" validate:"required"`
	DbUser     string    `json:"db_user" validate:"required"`
	DbPassword string    `json:"db_password" validate:"required"`
}

type DeletePsqlServiceReq struct {
	ServiceId uuid.UUID `json:"service_id" validate:"required"`
	KeepData  bool      `json:"keep_data"`
}

type PsqlServiceHandler struct {
	Server   *config.Server
	Validate *validator.Validate
	qCtx     context.Context
}

func InitPsqlServiceHandlers(s *config.Server) *PsqlServiceHandler {
	return &PsqlServiceHandler{
		Server:   s,
		Validate: validator.New(),
		qCtx:     context.Background(),
	}
}

func isPsqlImage(image string) bool {
	// TODO : improve checking logic
	return strings.Contains(image, "postgres")
}

func buildPsqlInternalURL(dbUser, dbPassword, serviceName, dbName string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:5432/%s", dbUser, dbPassword, serviceName, dbName)
}

// create a new psql service
//
// route: POST /api/service/psql
func (h *ServiceHandler) CreatePsqlService(c *echo.Context) error {
	q := h.Server.DB.Queries
	b := new(CreatePsqlServiceReq)
	docker := h.Server.Docker.Client

	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	// check if image is a postgres image
	if !isPsqlImage(b.Image) {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "Invalid image. Only postgres images are allowed"})
	}

	// check if service name already exists in the organization
	if exists, err := q.ServiceNameExists(h.qCtx, db.ServiceNameExistsParams{
		ProjectID: b.ProjectID,
		Name:      b.Name,
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to check service name"})
	} else if exists {
		return c.JSON(http.StatusConflict, types.Res[struct{}]{Message: "Service name already exists"})
	}

	network, err := q.GetProjectNetwork(h.qCtx, b.ProjectID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to fetch project network"})
	}

	serviceName := fmt.Sprintf("%s-%s", b.Name, security.GenerateRandomID(6))

	// create volume for the psql service
	volume, err := docker.VolumeCreate(h.qCtx, volume.CreateOptions{
		Name:   serviceName + "_pgdata",
		Driver: "local",
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to create volume"})
	}

	// create network if not exist
	if err := h.Server.Docker.CreateNetwork(network); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to create network"})
	}

	// config swarm service spec
	spec := swarm.ServiceSpec{
		Annotations: swarm.Annotations{
			Name: serviceName,
		},

		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: &swarm.ContainerSpec{
				Image: b.Image,

				Env: []string{
					"POSTGRES_PASSWORD=" + b.DbPassword,
					"POSTGRES_USER=" + b.DbUser,
					"POSTGRES_DB=" + b.DbName,
				},

				Mounts: []mount.Mount{
					{
						Type:   mount.TypeVolume,
						Source: volume.Name,
						Target: "/var/lib/postgresql/data",
					},
				},
			},

			Networks: []swarm.NetworkAttachmentConfig{
				{
					Target: network,
				},
			},

			RestartPolicy: &swarm.RestartPolicy{
				Condition: swarm.RestartPolicyConditionAny,
			},
		},
	}

	// depoly the service
	_, err = docker.ServiceCreate(h.qCtx, spec, swarm.ServiceCreateOptions{})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to deploy service"})
	}

	internalUrl := buildPsqlInternalURL(b.DbUser, b.DbPassword, serviceName, b.DbName)

	serviceID, err := h.Server.DB.Queries.CreatePsqlService(h.qCtx, db.CreatePsqlServiceParams{
		ID:               security.GeneratePrimaryKey(),
		ProjectID:        b.ProjectID,
		Type:             types.PsqlServiceType,
		SwarmServiceName: serviceName,
		Name:             b.Name,
		DbName:           b.DbName,
		DbUser:           b.DbUser,
		DbPassword:       b.DbPassword, // TODO : make is hased
		InternalUrl:      internalUrl,
		Image:            b.Image,
		Volume:           volume.Name,
	})
	if err != nil {
		fmt.Println("error creating service in db :", err)
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to create service"})
	}

	return c.JSON(http.StatusOK, types.Res[uuid.UUID]{
		Message: "",
		Data:    serviceID,
	})
}

// get psql service details by id
//
// route: GET /api/service/psql/:id
func (h *ServiceHandler) GetPsqlServiceById(c *echo.Context) error {
	serviceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "invalid service id"})
	}

	service, err := h.Server.DB.Queries.GetPsqlServiceById(h.qCtx, serviceID)
	if err != nil {
		return c.JSON(http.StatusNotFound, types.Res[struct{}]{Message: "service not found"})
	}

	return c.JSON(http.StatusOK, types.Res[db.GetPsqlServiceByIdRow]{
		Message: "",
		Data:    service,
	})
}

// update psql service details
//
// route: PUT /api/service/psql
func (h *ServiceHandler) UpdatePsqlServiceDetails(c *echo.Context) error {
	q := h.Server.DB.Queries

	b := new(UpdatePsqlServiceReq)
	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	service, err := q.GetPsqlServiceById(h.qCtx, b.ServiceID)
	if err != nil {
		return c.JSON(http.StatusNotFound, types.Res[struct{}]{Message: "service not found"})
	}

	internalUrl := buildPsqlInternalURL(b.DbUser, b.DbPassword, service.SwarmServiceName, b.DbName)

	if err := q.UpdatePsqlServiceDetails(h.qCtx, db.UpdatePsqlServiceDetailsParams{
		DbName:      b.DbName,
		DbUser:      b.DbUser,
		DbPassword:  b.DbPassword,
		InternalUrl: internalUrl,
		ID:          b.ServiceID,
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to update service details"})
	}

	return c.JSON(http.StatusOK, types.Res[struct{}]{Message: "Successfully updated service details"})
}

// redeploy psql service with saved details
//
// route: POST /api/service/psql/redeploy
func (h *ServiceHandler) RedeployPsqlService(c *echo.Context) error {
	q := h.Server.DB.Queries
	docker := h.Server.Docker.Client

	b := new(ServiceReq)
	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	service, err := q.GetPsqlServiceById(h.qCtx, b.ServiceId)
	if err != nil {
		return c.JSON(http.StatusNotFound, types.Res[struct{}]{Message: "service not found"})
	}

	inspectRes, _, err := docker.ServiceInspectWithRaw(h.qCtx, service.SwarmServiceName, swarm.ServiceInspectOptions{})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to inspect swarm service"})
	}

	serviceV := inspectRes.Version
	spec := inspectRes.Spec
	if spec.TaskTemplate.ContainerSpec == nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Missing container spec"})
	}

	spec.TaskTemplate.ContainerSpec.Env = []string{
		"POSTGRES_PASSWORD=" + service.DbPassword,
		"POSTGRES_USER=" + service.DbUser,
		"POSTGRES_DB=" + service.DbName,
	}

	if _, err := docker.ServiceUpdate(h.qCtx, service.SwarmServiceName, serviceV, spec, swarm.ServiceUpdateOptions{}); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to redeploy service"})
	}

	return c.JSON(http.StatusOK, types.Res[struct{}]{Message: "Successfully started redeploy"})
}

// stops and delete the psql service
//
// route: DELETE /api/service/psql
func (h *ServiceHandler) DeletePsqlService(c *echo.Context) error {
	docker := h.Server.Docker.Client
	q := h.Server.DB.Queries

	b := new(DeletePsqlServiceReq)

	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	service, err := q.GetPsqlServiceById(h.qCtx, b.ServiceId)
	if err != nil {
		return c.JSON(http.StatusConflict, types.Res[struct{}]{Message: "Failed to fetch service details"})
	}

	// check and stop the service if it is running
	if s, _, _ := docker.ServiceInspectWithRaw(h.qCtx, service.SwarmServiceName, swarm.ServiceInspectOptions{}); s.ID != "" {
		if err := docker.ServiceRemove(h.qCtx, service.SwarmServiceName); err != nil {
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
			Type:           types.PSQLPredefServiceType,
		}); err != nil {
			tx.Rollback()
			fmt.Println("err ", err)
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

	if err := tq.DeletePsqlService(h.qCtx, b.ServiceId); err != nil {
		tx.Rollback()
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to create service"})
	}

	if err := tx.Commit(); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to commit transaction"})
	}

	return c.JSON(http.StatusOK, types.Res[struct{}]{
		Message: "Successsfully deleted service",
	})
}
