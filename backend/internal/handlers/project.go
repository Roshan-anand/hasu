package handlers

import (
	"context"
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
	b := new(CreateOrgReq)
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

	// create a new project
	project, err := q.CreateProject(h.qCtx, db.CreateProjectParams{
		ID:             security.GeneratePrimaryKey(),
		Name:           b.Name,
		NetworkName:    b.Name + "_network",
		OrganizationID: org.ID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to create project"})
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
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{
			Message: "internal server error",
		})
	}

	return c.JSON(http.StatusOK, types.Res[[]db.Project]{
		Message: "",
		Data:    projects,
	})
}

// delete a project
//
// route: DELETE /api/project
func (h *ProjectHandler) DeleteProject(c *echo.Context) error {
	b := new(ProjectReq)
	q := h.Server.DB.Queries

	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	// check if project has any services running
	if exists, err := q.CheckProjectHasServices(h.qCtx, b.ProjectID); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "internal server error"})
	} else if exists {
		return c.JSON(http.StatusConflict, types.Res[struct{}]{Message: "Cannot delete project with active services"})
	}

	if err := q.DeleteProject(h.qCtx, b.ProjectID); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to delete project"})
	}

	return c.JSON(http.StatusOK, types.Res[struct{}]{Message: "project deleted successfully"})
}
