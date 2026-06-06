package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Roshan-anand/godploy/internal/config"
	"github.com/Roshan-anand/godploy/internal/db"
	"github.com/Roshan-anand/godploy/internal/lib/auth"
	"github.com/Roshan-anand/godploy/internal/lib/security"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

type ProjectHandler struct {
	Server   *config.Server
	Validate *validator.Validate
	qCtx     context.Context
}

type CreateProjectReq struct {
	Name string `json:"name" validate:"required,min=3"`
}

type ProjectReq struct {
	ProjectID uuid.UUID `json:"project_id" validate:"required"`
}

type DeleteProjectReq struct {
	ProjectID uuid.UUID `json:"project_id" validate:"required"`
}

func InitProjectHandlers(s *config.Server) *ProjectHandler {
	return &ProjectHandler{
		Server:   s,
		Validate: validator.New(),
		qCtx:     context.Background(),
	}
}

// create a new project
//
// route: POST /api/project
func (h *ProjectHandler) CreateProject(c *echo.Context) error {
	u := c.Get(h.Server.Config.EchoCtxUserKey).(auth.AuthUser)
	b := new(CreateProjectReq)
	q := h.Server.DB.Queries

	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	org, err := q.GetCurrentOrg(h.qCtx, u.Email)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to get current organization"})
	}

	// check if project already exists in the organization
	if exists, err := q.CheckProjectExists(h.qCtx, db.CheckProjectExistsParams{
		OrganizationID: org.ID,
		ProjectName:    b.Name,
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to check project name"})
	} else if exists {
		return c.JSON(http.StatusConflict, types.Res[struct{}]{Message: "Project with this name already exists in the organization"})
	}

	tx, err := h.Server.DB.Pool.BeginTx(h.qCtx, nil)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to start database transaction"})
	}
	tq := q.WithTx(tx)

	// create a new project
	project, err := tq.CreateProject(h.qCtx, db.CreateProjectParams{
		ID:             security.GeneratePrimaryKey(),
		Name:           b.Name,
		OrganizationID: org.ID,
	})
	if err != nil {
		tx.Rollback()
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to create project"})
	}

	// create a network
	networkName := b.Name + "_network"
	if err := h.Server.Docker.CreateNetwork(networkName); err != nil {
		tx.Rollback()
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to create network for project"})
	}

	// create a default production instance
	if err := tq.CreateInstance(h.qCtx, db.CreateInstanceParams{
		ID:           security.GeneratePrimaryKey(),
		Name:         "production",
		ProjectID:    project.ID,
		Network:      networkName,
		IsProduction: true,
	}); err != nil {
		tx.Rollback()
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to create default instance for project"})
	}

	if err := tx.Commit(); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to commit database transaction"})
	}

	return c.JSON(http.StatusOK, types.Res[db.CreateProjectRow]{Message: "", Data: project})
}

// get all project of the organization
//
// route: GET /api/project?org_id=
func (h *ProjectHandler) GetAllProject(c *echo.Context) error {
	orgID, err := uuid.Parse(c.QueryParam("org_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "invalid organization id"})
	}

	projects, err := h.Server.DB.Queries.GetAllProjects(h.qCtx, orgID)
	if err != nil {
		fmt.Println("Error fetching projects:", err)
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{
			Message: "internal server error",
		})
	}

	return c.JSON(http.StatusOK, types.Res[[]db.GetAllProjectsRow]{
		Message: "",
		Data:    projects,
	})
}

// delete a project
//
// route: DELETE /api/project
func (h *ProjectHandler) DeleteProject(c *echo.Context) error {
	b := new(DeleteProjectReq)
	q := h.Server.DB.Queries

	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	// check if project has any services running
	if exists, err := q.CheckProjectHasService(h.qCtx, b.ProjectID); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "internal server error"})
	} else if exists {
		return c.JSON(http.StatusConflict, types.Res[struct{}]{Message: "Cannot delete project with active instance"})
	}

	// delete the project network
	networks, err := q.GetAllNetworksByProjectId(h.qCtx, b.ProjectID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to get project network"})
	}

	go h.Server.Docker.RemoveNetwork(networks)

	if err := q.DeleteProject(h.qCtx, b.ProjectID); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to delete project"})
	}

	return c.JSON(http.StatusOK, types.Res[struct{}]{Message: "project deleted successfully"})
}

// TODO : make route to shut down an instance
// by remving all the services,swarm service, volumes etc
func (h *ProjectHandler) ShutDownInstance(c *echo.Context) error {
	return nil
}
