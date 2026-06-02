package handlers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/Roshan-anand/godploy/internal/config"
	"github.com/Roshan-anand/godploy/internal/db"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

type DeleteVolumeReq struct {
	Volumes []string `json:"volumes" validate:"required"`
}

type VolumeHandler struct {
	Server   *config.Server
	Validate *validator.Validate
	qCtx     context.Context
}

func InitVolumeHandlers(s *config.Server) *VolumeHandler {
	return &VolumeHandler{
		Server:   s,
		Validate: validator.New(),
		qCtx:     context.Background(),
	}
}

// get all orphan volumes
//
// route: GET /api/volume?org_id
func (h *ServiceHandler) GetAllVolume(c *echo.Context) error {
	q := h.Server.DB.Queries

	orgID, err := uuid.Parse(c.QueryParam("org_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "invalid org id"})
	}

	volumes, err := q.GetAllOrphanVolumesByOrgID(h.qCtx, orgID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.JSON(http.StatusNoContent, types.Res[struct{}]{Message: "no volumes available"})
		}
		fmt.Println("error getting volumes:", err)
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "internal server error"})
	}

	return c.JSON(http.StatusOK, types.Res[[]db.OrphanVolume]{Message: "successfully get volumes", Data: volumes})
}

// delete selected orphan volumes
//
// route: DELETE /api/volume
func (h *ServiceHandler) DeleteVolume(c *echo.Context) error {
	b := new(DeleteVolumeReq)
	q := h.Server.DB.Queries
	docker := h.Server.Docker.Client

	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	for _, vol := range b.Volumes {
		if err := docker.VolumeRemove(context.Background(), vol, true); err != nil {
			return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "error deleting volume :" + vol})
		}
		if err := q.DeleteOrphanVolume(h.qCtx, vol); err != nil {
			return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "error deleting volume :" + vol})
		}
	}

	return c.JSON(http.StatusOK, types.Res[struct{}]{
		Message: "succesfully remove volumes",
	})
}
