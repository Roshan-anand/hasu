package handlers

import (
	"context"
	"net/http"

	"github.com/Roshan-anand/godploy/internal/config"
	"github.com/Roshan-anand/godploy/internal/db"
	deployjob "github.com/Roshan-anand/godploy/internal/jobs/deployment"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

// PreviewHandler exposes preview instance lifecycle endpoints.
type PreviewHandler struct {
	Server   *config.Server
	qCtx     context.Context
	Validate *validator.Validate
}

// CreatePreviewRequest is the body for preview creation.
type CreatePreviewRequest struct {
	ProjectID      uuid.UUID           `json:"project_id" validate:"required"`
	Name           string              `json:"name" validate:"required"`
	PRNumber       int                 `json:"pr_number"`
	RepoID         int                 `json:"repo_id"`
	GitSourceType  types.GitSourceType `json:"git_source_type" validate:"required,oneof=pr branch"`
	GitSourceValue string              `json:"git_source_value" validate:"required"`
	EnvCopy        bool                `json:"env_copy"`
}

// DeletePreviewRequest is the body for preview deletion.
type DeletePreviewRequest struct {
	PreviewID uuid.UUID `json:"preview_id" validate:"required"`
}

// InitPreviewHandlers creates a new PreviewHandler.
func InitPreviewHandlers(s *config.Server) *PreviewHandler {
	return &PreviewHandler{
		Server:   s,
		Validate: validator.New(),
		qCtx:     context.Background(),
	}
}

// CreatePreview spins up a preview instance from a PR or branch.
// route: POST /api/instance/preview
func (h *PreviewHandler) CreatePreview(c *echo.Context) error {
	req := new(CreatePreviewRequest)
	if Res := BindAndValidate(req, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	if err := h.Server.Services.Deployment.AssignCreatePreview(h.qCtx, &deployjob.CreatePreviewJobParams{
		ProjectID:      req.ProjectID,
		Name:           req.Name,
		PRNumber:       req.PRNumber,
		RepoID:         req.RepoID,
		GitSourceType:  req.GitSourceType,
		GitSourceValue: req.GitSourceValue,
		EnvCopy:        req.EnvCopy,
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
	req := new(DeletePreviewRequest)
	if Res := BindAndValidate(req, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	if err := h.Server.Services.Deployment.DeletePreview(h.qCtx, req.PreviewID); err != nil {
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
