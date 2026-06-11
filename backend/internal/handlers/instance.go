package handlers

import (
	"context"
	"net/http"

	"github.com/Roshan-anand/godploy/internal/config"
	"github.com/Roshan-anand/godploy/internal/db"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

type InstanceHandler struct {
	Server   *config.Server
	Validate *validator.Validate
	qCtx     context.Context
}

func InitInstanceHandlers(s *config.Server) *InstanceHandler {
	return &InstanceHandler{
		Server:   s,
		Validate: validator.New(),
		qCtx:     context.Background(),
	}
}

// get all organizations accessible to the authenticated user
//
// route: GET /api/instance?project=&org_id=
func (h *InstanceHandler) GetAllInstance(c *echo.Context) error {
	q := h.Server.DB.Queries

	project := c.QueryParam("project")
	if project == "" {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{
			Message: "invalid project",
		})
	}

	orgID, err := uuid.Parse(c.QueryParam("org_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{
			Message: "invalid org_id",
		})
	}

	instances, err := q.GetAllInstance(h.qCtx, db.GetAllInstanceParams{
		OrganizationID: orgID,
		Project:        project,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{
			Message: "internal server error",
		})
	}

	return c.JSON(http.StatusOK, types.Res[[]db.GetAllInstanceRow]{
		Message: "",
		Data:    instances,
	})
}

type RenameInstanceReq struct {
	InstanceID uuid.UUID `json:"instance_id" validate:"required"`
	ProjectID  uuid.UUID `json:"project_id" validate:"required"`
	Name       string    `json:"name" validate:"required,min=3"`
}

// rename a project instance
//
// route: PUT /api/instance/rename
func (h *InstanceHandler) RenameInstance(c *echo.Context) error {
	b := new(RenameInstanceReq)

	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	q := h.Server.DB.Queries

	// check if instance name already exists in the project
	if exists, err := q.CheckInstanceExists(h.qCtx, db.CheckInstanceExistsParams{
		ProjectID:    b.ProjectID,
		InstanceName: b.Name,
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "internal server error"})
	} else if exists {
		return c.JSON(http.StatusConflict, types.Res[struct{}]{Message: "Instance with this name already exists in the project"})
	}

	instance, err := q.RenameInstance(h.qCtx, db.RenameInstanceParams{
		Name: b.Name,
		ID:   b.InstanceID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to rename instance"})
	}

	return c.JSON(http.StatusOK, types.Res[db.RenameInstanceRow]{Message: "", Data: instance})
}
