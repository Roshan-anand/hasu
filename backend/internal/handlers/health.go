package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"

	"github.com/Roshan-anand/godploy/internal/config"
	"github.com/Roshan-anand/godploy/internal/db"
	"github.com/Roshan-anand/godploy/internal/lib/security"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

type HealthHandler struct {
	Server   *config.Server
	Validate *validator.Validate
	qCtx     context.Context
}

type UrlReq struct {
	Url string `json:"url" validate:"required"`
}

func InitHealthHandlers(s *config.Server) *HealthHandler {
	return &HealthHandler{
		Server:   s,
		Validate: validator.New(),
		qCtx:     context.Background(),
	}
}

// to check server health and connectivity with database and other dependencies
//
// route: GET /api/health
func (h *HealthHandler) HealthCheck(c *echo.Context) error {
	switch {
	case h.Server.DB == nil:
		return c.JSON(500, types.Res{Message: "database not initialized"})
	case h.Server.BadgerDB == nil:
		return c.JSON(500, types.Res{Message: "badger database not initialized"})
	case h.Server.Docker == nil:
		return c.JSON(500, types.Res{Message: "docker client not initialized"})
	}

	return c.JSON(200, types.Res{Message: "ok"})
}

// TODO : remove this router for production
type ghAppReq struct {
	Name           string    `json:"name" validate:"required"`
	OrgID          uuid.UUID `json:"organization_id" validate:"required"`
	AppID          string    `json:"app_id" validate:"required"`
	InstallationID string    `json:"installation_id" validate:"required"`
	PemKey         string    `json:"pem_key" validate:"required"`
	WebhookSecret  string    `json:"webhook_secret" validate:"required"`
}

func (h *HealthHandler) SetGhApp(c *echo.Context) error {
	b := new(ghAppReq)
	q := h.Server.DB.Queries

	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	// parse string to int for app id
	appId, err := strconv.ParseInt(b.AppID, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res{Message: "invalid app id"})
	}

	insId, err := strconv.ParseInt(b.InstallationID, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res{Message: "invalid installation id"})
	}

	ghAppId, err := q.CreateGithubApp(h.qCtx, db.CreateGithubAppParams{
		ID:             security.GeneratePrimaryKey(),
		Name:           b.Name,
		OrganizationID: b.OrgID,
		AppID:          appId,
		PemKey:         b.PemKey,
		WebhookSecret:  b.WebhookSecret,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res{Message: "failed to create github app"})
	}

	if err := q.InsertInstallationID(h.qCtx, db.InsertInstallationIDParams{
		AppID: ghAppId,
		InstallationID: sql.NullInt64{
			Valid: true,
			Int64: insId,
		},
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res{Message: "failed to insert installation id"})
	}

	return c.JSON(http.StatusOK, types.Res{Message: "github app created successfully"})
}
