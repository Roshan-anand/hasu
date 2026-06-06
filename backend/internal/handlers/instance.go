package handlers

import (
	"context"
	"fmt"
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

	fmt.Println("triggerget all instace")

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
