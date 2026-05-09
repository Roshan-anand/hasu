package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/Roshan-anand/godploy/internal/config"
	logbrokerqueue "github.com/Roshan-anand/godploy/internal/jobs/logbroker/queue"
	"github.com/Roshan-anand/godploy/internal/lib"
	"github.com/Roshan-anand/godploy/internal/lib/sse"
	"github.com/Roshan-anand/godploy/internal/lib/types"
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

func InitServiceHandlers(s *config.Server) *ServiceHandler {
	return &ServiceHandler{
		Server:   s,
		Validate: validator.New(),
		qCtx:     context.Background(),
	}
}

// get all services of a organization
//
// route: GET /api/service
func (h *ServiceHandler) GetAllServices(c *echo.Context) error {
	u := c.Get(h.Server.Config.EchoCtxUserKey).(lib.AuthUser)
	q := h.Server.DB.Queries

	orgID, err := q.GetUserCurrentOrg(h.qCtx, u.Email)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res{Message: "failed to get user's current org"})
	}

	services, err := q.GetAllService(h.qCtx, orgID)
	if err != nil {
		fmt.Printf("error getting services for org_id: %v, error: %v\n", orgID, err)
		return c.JSON(http.StatusInternalServerError, types.Res{Message: "failed to get services"})
	}

	return c.JSON(http.StatusOK, services)
}

// get all service deployment jobs
//
// route: GET /api/service/deployment?service_id
func (h *ServiceHandler) GetServiceDeployments(c *echo.Context) error {
	q := h.Server.DB.Queries

	// TODO : inlcude org_id to get all deployments of the org / service based on the query params.

	serviceID, err := uuid.Parse(c.QueryParam("service_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res{Message: "invalid service_id"})
	}

	deployemnts, err := q.GetDeploymentsByServiceID(h.qCtx, serviceID)
	if err != nil {
		fmt.Printf("error getting deployments for service_id: %v, error: %v\n", serviceID, err)
		return c.JSON(http.StatusInternalServerError, types.Res{Message: "failed to get deployments"})
	}

	return c.JSON(http.StatusOK, deployemnts)
}

// delete service deployment by deployment id
//
// route: DELETE /api/service/deployment
func (h *ServiceHandler) DeleteServiceDeployment(c *echo.Context) error {
	b := new(DeploymentReq)
	q := h.Server.DB.Queries

	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	dyp, err := q.GetDeploymentImgByID(h.qCtx, b.DeploymentID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res{Message: "Failed to get deployment"})
	}

	if err := q.DeleteDeploymentByID(h.qCtx, b.DeploymentID); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res{Message: "failed to delete deployment"})
	}

	h.Server.Docker.RemoveImages([]string{dyp.ImageName.String})
	h.Server.BadgerDB.DeleteAllLogsByDeploymentID([]uuid.UUID{b.DeploymentID})

	return c.JSON(http.StatusOK, types.Res{Message: "deployment deleted successfully"})
}

// subscribe to service deployment logs event
//
// route: GET /api/service/deployment/logs?deployment_id
func (h *ServiceHandler) SubscribeServiceDeploymentLogs(c *echo.Context) error {
	q := h.Server.DB.Queries
	dID, err := uuid.Parse(c.QueryParam("deployment_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res{Message: "invalid deployment_id"})
	}

	w := c.Response()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	sse := sse.NewSSE(w)

	status, err := q.GetDeploymentStatus(h.qCtx, dID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res{Message: "failed to get deployment status"})
	}

	userID := lib.GeneratePrimaryKey()

	// if deployment is successful or failed, then stream logs from badgerDB
	if status == types.DeploymentReady || status == types.DeploymentError {
		if err := h.Server.BadgerDB.StreamAllLogsByDeploymentID(dID, sse); err != nil {
			return c.JSON(http.StatusInternalServerError, types.Res{Message: "failed to stream logs"})
		}
		return nil
	}

	// subscribe to log broker queue to get real-time logs of the deployment
	h.Server.LogBrokerQ.SubscribeLogs(userID, &logbrokerqueue.Subscriber{
		SSE:          sse,
		DeploymentID: dID,
	})

	for {
		select {
		case <-c.Request().Context().Done():
			log.Printf("SSE client disconnected, ip: %v", c.RealIP())
			h.Server.LogBrokerQ.UnsubscribeLogs(userID)
			return nil
		}
	}
}
