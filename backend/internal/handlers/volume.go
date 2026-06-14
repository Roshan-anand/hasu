package handlers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/Roshan-anand/godploy/internal/config"
	"github.com/Roshan-anand/godploy/internal/db"
	"github.com/Roshan-anand/godploy/internal/lib/auth"
	typesLib "github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

type DeleteVolumeReq struct {
	Volumes []string `json:"volumes" validate:"required"`
}

type RenameVolumeReq struct {
	ID          uuid.UUID `json:"id" validate:"required"`
	DisplayName string    `json:"display_name" validate:"required"`
}

// OrphanVolumeWithUsage extends the DB model with Docker volume usage info.
type OrphanVolumeWithUsage struct {
	db.OrphanVolume
	SizeBytes int64 `json:"size_bytes"`
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

// get all orphan volumes with Docker volume usage (size)
//
// route: GET /api/volume?org_id
func (h *ServiceHandler) GetAllVolume(c *echo.Context) error {
	q := h.Server.DB.Queries
	docker := h.Server.Docker.Client

	orgID, err := uuid.Parse(c.QueryParam("org_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, typesLib.Res[struct{}]{Message: "invalid org id"})
	}

	volumes, err := q.GetAllOrphanVolumesByOrgID(h.qCtx, orgID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.JSON(http.StatusNoContent, typesLib.Res[[]struct{}]{Message: "no volumes available", Data: []struct{}{}})
		}
		fmt.Println("error getting volumes:", err)
		return c.JSON(http.StatusInternalServerError, typesLib.Res[struct{}]{Message: "internal server error"})
	}

	// Enrich each volume with its Docker usage info (size)
	enriched := make([]OrphanVolumeWithUsage, 0, len(volumes))
	for _, v := range volumes {
		info := OrphanVolumeWithUsage{OrphanVolume: v}
		dvol, err := docker.VolumeInspect(context.Background(), v.Volume)
		if err == nil && dvol.UsageData != nil {
			info.SizeBytes = dvol.UsageData.Size
		}
		enriched = append(enriched, info)
	}

	fmt.Printf("enriched volumes: %+v\n", enriched) // Debug log to check enriched data

	return c.JSON(http.StatusOK, typesLib.Res[[]OrphanVolumeWithUsage]{Message: "successfully get volumes", Data: enriched})
}

// get all orphan volumes by type
//
// route: GET /api/volume/:type?org_id
func (h *ServiceHandler) GetAllVolumeByType(c *echo.Context) error {
	q := h.Server.DB.Queries

	typeParam := typesLib.PredefServiceType(c.Param("type"))

	orgID, err := uuid.Parse(c.QueryParam("org_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, typesLib.Res[struct{}]{Message: "invalid org id"})
	}

	volumes, err := q.GetOrphanVolumeByType(h.qCtx, db.GetOrphanVolumeByTypeParams{
		OrganizationID: orgID,
		Type:           typeParam,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.JSON(http.StatusNoContent, typesLib.Res[struct{}]{Message: "no volumes available"})
		}
		fmt.Println("error getting volumes:", err)
		return c.JSON(http.StatusInternalServerError, typesLib.Res[struct{}]{Message: "internal server error"})
	}

	return c.JSON(http.StatusOK, typesLib.Res[[]db.OrphanVolume]{Message: "successfully get volumes", Data: volumes})
}

// delete selected orphan volumes
//
// route: DELETE /api/volume
func (h *ServiceHandler) DeleteVolume(c *echo.Context) error {
	u := c.Get(h.Server.Config.EchoCtxUserKey).(auth.AuthUser)
	b := new(DeleteVolumeReq)
	q := h.Server.DB.Queries
	docker := h.Server.Docker.Client

	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	org, err := q.GetCurrentOrg(h.qCtx, u.Email)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, typesLib.Res[struct{}]{Message: "Failed to fetch organization id"})
	}

	for _, vol := range b.Volumes {
		if err := docker.VolumeRemove(context.Background(), vol, true); err != nil {
			return c.JSON(http.StatusInternalServerError, typesLib.Res[struct{}]{Message: "error deleting volume :" + vol})
		}
		if err := q.DeleteOrphanVolume(h.qCtx, db.DeleteOrphanVolumeParams{
			Volume:         vol,
			OrganizationID: org.ID,
		}); err != nil {
			return c.JSON(http.StatusInternalServerError, typesLib.Res[struct{}]{Message: "error deleting volume :" + vol})
		}
	}

	return c.JSON(http.StatusOK, typesLib.Res[struct{}]{
		Message: "succesfully remove volumes",
	})
}

// rename an orphan volume's display name
//
// route: PATCH /api/volume
func (h *ServiceHandler) RenameVolume(c *echo.Context) error {
	b := new(RenameVolumeReq)
	q := h.Server.DB.Queries

	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	vol, err := q.GetOrphanVolumeById(h.qCtx, b.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.JSON(http.StatusNotFound, typesLib.Res[struct{}]{Message: "volume not found"})
		}
		fmt.Println("error fetching volume:", err)
		return c.JSON(http.StatusInternalServerError, typesLib.Res[struct{}]{Message: "Failed to fetch volume"})
	}

	if err := q.UpdateOrphanVolumeName(h.qCtx, db.UpdateOrphanVolumeNameParams{
		ID:             vol.ID,
		DisplayName:    b.DisplayName,
		OrganizationID: vol.OrganizationID,
	}); err != nil {
		fmt.Println("error renaming volume:", err)
		return c.JSON(http.StatusInternalServerError, typesLib.Res[struct{}]{Message: "Failed to rename volume"})
	}

	return c.JSON(http.StatusOK, typesLib.Res[struct{}]{
		Message: "Volume renamed successfully",
	})
}
