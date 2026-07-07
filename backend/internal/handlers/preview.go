package handlers

import (
	"context"
	"net/http"

	"github.com/Roshan-anand/godploy/internal/config"
	"github.com/Roshan-anand/godploy/internal/db"
	deployjob "github.com/Roshan-anand/godploy/internal/jobs/deployment"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

// PreviewHandler exposes preview instance lifecycle endpoints.
type PreviewHandler struct {
	Server *config.Server
	qCtx   context.Context
}

// CreatePreviewRequest is the body for preview creation.
type CreatePreviewRequest struct {
	ProjectID      uuid.UUID `json:"project_id" validate:"required"`
	Name           string    `json:"name" validate:"required"`
	PRNumber       int       `json:"pr_number"`
	RepoID         int       `json:"repo_id"`
	HeadBranch     string    `json:"head_branch" validate:"required"`
	GitSourceType  string    `json:"git_source_type" validate:"required,oneof=pr branch"`
	GitSourceValue string    `json:"git_source_value" validate:"required"`
	EnvCopy        bool      `json:"env_copy"`
}

// DeletePreviewRequest is the body for preview deletion.
type DeletePreviewRequest struct {
	PreviewID uuid.UUID `json:"preview_id" validate:"required"`
}

// InitPreviewHandlers creates a new PreviewHandler.
func InitPreviewHandlers(s *config.Server) *PreviewHandler {
	return &PreviewHandler{
		Server: s,
		qCtx:   context.Background(),
	}
}

// CreatePreview spins up a preview instance from a PR or branch.
// route: POST /api/instance/preview
func (h *PreviewHandler) CreatePreview(c *echo.Context) error {
	var b CreatePreviewRequest
	if err := c.Bind(&b); err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{
			Message: "invalid request body",
		})
	}

	if err := h.Server.Services.Deployment.AssignCreatePreview(h.qCtx, &deployjob.CreatePreviewJobParams{
		ProjectID:      b.ProjectID,
		Name:           b.Name,
		PRNumber:       b.PRNumber,
		RepoID:         b.RepoID,
		HeadBranch:     b.HeadBranch,
		GitSourceType:  b.GitSourceType,
		GitSourceValue: b.GitSourceValue,
		EnvCopy:        b.EnvCopy,
	}, nil); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusAccepted, types.Res[struct{}]{
		Message: "preview creation queued",
	})
}

// DeletePreview initiates cleanup for a preview instance.
// route: DELETE /api/instance/preview
func (h *PreviewHandler) DeletePreview(c *echo.Context) error {
	var b DeletePreviewRequest
	if err := c.Bind(&b); err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{
			Message: "invalid request body",
		})
	}

	if err := h.Server.Services.Deployment.DeletePreview(h.qCtx, b.PreviewID); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, types.Res[struct{}]{
		Message: "preview deletion started",
	})
}

// ListPreviews returns all preview instances for a project.
// route: GET /api/instance/preview/list?project_id=
func (h *PreviewHandler) ListPreviews(c *echo.Context) error {
	projectID, err := uuid.Parse(c.QueryParam("project_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{
			Message: "invalid project_id",
		})
	}

	previews, err := h.Server.Services.Deployment.ListPreviews(h.qCtx, projectID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, types.Res[[]db.GetPreviewInstancesByProjectRow]{
		Message: "",
		Data:    previews,
	})
}
