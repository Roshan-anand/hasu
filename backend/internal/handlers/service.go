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

// get all services of a project
//
// route: GET /api/service?project_id=
func (h *ServiceHandler) GetAllServices(c *echo.Context) error {
	q := h.Server.DB.Queries

	projectID, err := uuid.Parse(c.QueryParam("project_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "invalid project_id"})
	}

	services, err := q.GetAllService(h.qCtx, projectID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "failed to get services"})
	}

	return c.JSON(http.StatusOK, types.Res[[]db.GetAllServiceRow]{
		Message: "",
		Data:    services,
	})
}

// get service logs
//
// route: GET /api/service/logs?branch_id
func (h *ServiceHandler) GetServiceLogs(c *echo.Context) error {
	q := h.Server.DB.Queries

	fmt.Println("trigger server olgs sse")
	branchID, err := uuid.Parse(c.QueryParam("branch_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "invalid branch_id"})
	}

	swarmService, err := q.GetSwarmServiceByBranchId(h.qCtx, branchID)
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

	// simple scanner to read the raw logs
	scanner := bufio.NewScanner(serviceLogs)
	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			sse.SendEvent("log", []byte(line))
		}
	}()

	if err := scanner.Err(); err != nil {
		log.Printf("error streaming service logs: %v", err)
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "failed to stream service logs"})
	}

	for {
		select {
		case <-c.Request().Context().Done():
			log.Printf("SSE client disconnected, ip: %v", c.RealIP())
			return nil
		}
	}
}
