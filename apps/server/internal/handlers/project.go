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

type TransferProjectReq struct {
	ProjectID   uuid.UUID `json:"project_id" validate:"required"`
	TargetOrgID uuid.UUID `json:"target_org_id" validate:"required"`
}

type RenameProjectReq struct {
	ProjectID uuid.UUID `json:"project_id" validate:"required"`
	OrgID     uuid.UUID `json:"org_id" validate:"required"`
	Name      string    `json:"name" validate:"required,min=3"`
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
		if h.Server.DB.IsUniqueConstraintError(err) {
			return c.JSON(http.StatusConflict, types.Res[struct{}]{Message: "Project with this name already exists in the organization"})
		}
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
		Status:       types.InstanceReady,
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
// TODO : also remove all preview instances
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

	go h.Server.Docker.RemoveNetworks(networks)

	if err := q.DeleteProject(h.qCtx, b.ProjectID); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to delete project"})
	}

	return c.JSON(http.StatusOK, types.Res[struct{}]{Message: "project deleted successfully"})
}

// transfer a project to another organization
//
// route: PUT /api/project/transfer
func (h *ProjectHandler) TransferProject(c *echo.Context) error {
	u := c.Get(h.Server.Config.EchoCtxUserKey).(auth.AuthUser)
	b := new(TransferProjectReq)

	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	q := h.Server.DB.Queries

	// validate that the user has access to the target org
	if exists, err := q.CheckUserOrgExists(h.qCtx, db.CheckUserOrgExistsParams{
		UserEmail:      u.Email,
		OrganizationID: b.TargetOrgID,
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "internal server error"})
	} else if !exists {
		return c.JSON(http.StatusForbidden, types.Res[struct{}]{Message: "User does not have access to the target organization"})
	}

	// transfer the project
	if err := q.TransferProject(h.qCtx, db.TransferProjectParams{
		OrganizationID: b.TargetOrgID,
		ID:             b.ProjectID,
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to transfer project"})
	}

	return c.JSON(http.StatusOK, types.Res[struct{}]{Message: "Project transferred successfully"})
}

// rename a project
//
// route: PUT /api/project/rename
func (h *ProjectHandler) RenameProject(c *echo.Context) error {
	b := new(RenameProjectReq)

	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	q := h.Server.DB.Queries

	project, err := q.RenameProject(h.qCtx, db.RenameProjectParams{
		Name: b.Name,
		ID:   b.ProjectID,
	})
	if err != nil {
		if h.Server.DB.IsUniqueConstraintError(err) {
			return c.JSON(http.StatusConflict, types.Res[struct{}]{Message: "Project with this name already exists in the organization"})
		}
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to rename project"})
	}

	return c.JSON(http.StatusOK, types.Res[db.RenameProjectRow]{Message: "", Data: project})
}
