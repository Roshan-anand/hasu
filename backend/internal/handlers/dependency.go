package handlers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/Roshan-anand/godploy/internal/config"
	"github.com/Roshan-anand/godploy/internal/db"
	"github.com/Roshan-anand/godploy/internal/lib/security"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

var envKeyRegex = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// --- Request/Response types ---

type CreateDependencyReq struct {
	SourceServiceID uuid.UUID `json:"source_service_id" validate:"required"`
	TargetServiceID uuid.UUID `json:"target_service_id" validate:"required"`
	TargetCol       string    `json:"target_col" validate:"required"`
	EnvKey          string    `json:"env_key" validate:"required"`
}

type UpdateDependencyReq struct {
	TargetServiceID uuid.UUID `json:"target_service_id" validate:"required"`
	TargetCol       string    `json:"target_col" validate:"required"`
	EnvKey          string    `json:"env_key" validate:"required"`
}

type ServiceDependencyRes struct {
	ID                uuid.UUID `json:"id"`
	SourceServiceID   uuid.UUID `json:"source_service_id"`
	TargetServiceID   uuid.UUID `json:"target_service_id"`
	TargetServiceName string    `json:"target_service_name"`
	TargetServiceType string    `json:"target_service_type"`
	TargetCol         string    `json:"target_col"`
	EnvKey            string    `json:"env_key"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type ListDependenciesRes struct {
	Dependencies []ServiceDependencyRes `json:"dependencies"`
}

type DependencyTargetRes struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	ServiceType string    `json:"service_type"`
	AllowedCols []string  `json:"allowed_cols"`
}

type ListDependencyTargetsRes struct {
	Targets []DependencyTargetRes `json:"targets"`
}

type DependencyHandler struct {
	Server   *config.Server
	Validate *validator.Validate
	qCtx     context.Context
}

// InitDependencyHandlers creates a new handler instance for dependency management.
func InitDependencyHandlers(s *config.Server) *DependencyHandler {
	return &DependencyHandler{
		Server:   s,
		Validate: validator.New(),
		qCtx:     context.Background(),
	}
}

// allowed target columns per service type
func allowedTargetCols(serviceType string) []string {
	switch serviceType {
	case "app":
		return []string{"internal_url", "domain", "name"}
	case "psql":
		return []string{"internal_url", "db_name", "db_user", "db_password", "name"}
	case "redis":
		return []string{"internal_url", "password", "name"}
	}
	return nil
}

func containsString(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}

// validateSameInstance ensures target exists in the same instance as source.
func (h *DependencyHandler) validateSameInstance(sourceServiceID, targetServiceID uuid.UUID) error {
	q := h.Server.DB.Queries

	source, err := q.GetAppServiceOnly(h.qCtx, sourceServiceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("source service not found")
		}
		return err
	}

	targets, err := q.GetDependencyTargets(h.qCtx, db.GetDependencyTargetsParams{
		InstanceID: source.InstanceID,
		ID:         sourceServiceID,
	})
	if err != nil {
		return err
	}

	for _, t := range targets {
		if t.ID == targetServiceID {
			return nil
		}
	}
	return fmt.Errorf("target service not found in same instance")
}

// list all dependencies for a source service
//
// route: GET /api/service/app/dependencies?service_id=
func (h *DependencyHandler) GetServiceDependencies(c *echo.Context) error {
	q := h.Server.DB.Queries

	serviceID, err := uuid.Parse(c.QueryParam("service_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "invalid service_id"})
	}

	rows, err := q.GetServiceDependencies(h.qCtx, serviceID)
	if err != nil {
		fmt.Println("error getting dependencies:", err)
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "failed to get dependencies"})
	}

	deps := make([]ServiceDependencyRes, 0, len(rows))
	for _, r := range rows {
		deps = append(deps, ServiceDependencyRes{
			ID:                r.ID,
			SourceServiceID:   r.SourceServiceID,
			TargetServiceID:   r.TargetServiceID,
			TargetServiceName: r.TargetServiceName,
			TargetServiceType: r.TargetServiceType,
			TargetCol:         r.TargetCol,
			EnvKey:            r.EnvKey,
			CreatedAt:         r.CreatedAt,
			UpdatedAt:         r.UpdatedAt,
		})
	}

	return c.JSON(http.StatusOK, types.Res[ListDependenciesRes]{
		Data: ListDependenciesRes{Dependencies: deps},
	})
}

// link source service to target with env key mapping
//
// route: POST /api/service/app/dependencies
func (h *DependencyHandler) CreateServiceDependency(c *echo.Context) error {
	req := new(CreateDependencyReq)
	if Res := BindAndValidate(req, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	// env key format validation
	if !envKeyRegex.MatchString(req.EnvKey) {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "invalid env_key format"})
	}

	// self-dependency check
	if req.SourceServiceID == req.TargetServiceID {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "cannot depend on itself"})
	}

	// same-instance validation
	if err := h.validateSameInstance(req.SourceServiceID, req.TargetServiceID); err != nil {
		fmt.Println("create dependency validation error:", err)
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: err.Error()})
	}

	// target column validation per service type
	q := h.Server.DB.Queries
	source, err := q.GetAppServiceOnly(h.qCtx, req.SourceServiceID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "failed to validate source service"})
	}

	targets, err := q.GetDependencyTargets(h.qCtx, db.GetDependencyTargetsParams{
		InstanceID: source.InstanceID,
		ID:         req.SourceServiceID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "failed to validate target service"})
	}

	var targetType string
	for _, t := range targets {
		if t.ID == req.TargetServiceID {
			targetType = t.ServiceType
			break
		}
	}

	if targetType == "" {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "target service not found"})
	}

	if !containsString(allowedTargetCols(targetType), req.TargetCol) {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "invalid target column for service type"})
	}

	// domain-specific validation
	if req.TargetCol == "domain" {
		app, err := q.GetAppServiceOnly(h.qCtx, req.TargetServiceID)
		if err != nil || !app.IsPublic || !app.Domain.Valid {
			return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "domain not available for target service"})
		}
	}

	dep, err := q.CreateServiceDependency(h.qCtx, db.CreateServiceDependencyParams{
		ID:              security.GeneratePrimaryKey(),
		SourceServiceID: req.SourceServiceID,
		TargetServiceID: req.TargetServiceID,
		TargetCol:       req.TargetCol,
		EnvKey:          req.EnvKey,
	})
	if err != nil {
		fmt.Println("error creating dependency:", err)
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "failed to create dependency"})
	}

	// resolve target name for response
	targetName := ""
	for _, t := range targets {
		if t.ID == dep.TargetServiceID {
			targetName = t.Name
			break
		}
	}

	return c.JSON(http.StatusCreated, types.Res[ServiceDependencyRes]{
		Message: "dependency created",
		Data: ServiceDependencyRes{
			ID:                dep.ID,
			SourceServiceID:   dep.SourceServiceID,
			TargetServiceID:   dep.TargetServiceID,
			TargetServiceName: targetName,
			TargetServiceType: targetType,
			TargetCol:         dep.TargetCol,
			EnvKey:            dep.EnvKey,
			CreatedAt:         dep.CreatedAt,
			UpdatedAt:         dep.UpdatedAt,
		},
	})
}

// list eligible target services in same instance
//
// route: GET /api/service/app/dependency-targets?service_id=
func (h *DependencyHandler) GetDependencyTargets(c *echo.Context) error {
	q := h.Server.DB.Queries

	serviceID, err := uuid.Parse(c.QueryParam("service_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "invalid service_id"})
	}

	source, err := q.GetAppServiceOnly(h.qCtx, serviceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.JSON(http.StatusNotFound, types.Res[struct{}]{Message: "source service not found"})
		}
		fmt.Println("error getting source service:", err)
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "failed to get source service"})
	}

	targets, err := q.GetDependencyTargets(h.qCtx, db.GetDependencyTargetsParams{
		InstanceID: source.InstanceID,
		ID:         serviceID,
	})
	if err != nil {
		fmt.Println("error getting dependency targets:", err)
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "failed to get targets"})
	}

	res := make([]DependencyTargetRes, 0, len(targets))
	for _, t := range targets {
		cols := allowedTargetCols(t.ServiceType)
		// domain-specific filtering: only include domain if app is public and has a domain
		if t.ServiceType == "app" {
			app, err := q.GetAppServiceOnly(h.qCtx, t.ID)
			if err == nil && (!app.IsPublic || !app.Domain.Valid) {
				filtered := make([]string, 0, len(cols))
				for _, c := range cols {
					if c != "domain" {
						filtered = append(filtered, c)
					}
				}
				cols = filtered
			}
		}
		res = append(res, DependencyTargetRes{
			ID:          t.ID,
			Name:        t.Name,
			ServiceType: t.ServiceType,
			AllowedCols: cols,
		})
	}

	return c.JSON(http.StatusOK, types.Res[ListDependencyTargetsRes]{
		Data: ListDependencyTargetsRes{Targets: res},
	})
}

// modify target, column, or env key of a dependency
//
// route: PUT /api/service/app/dependencies/:id
func (h *DependencyHandler) UpdateServiceDependency(c *echo.Context) error {
	req := new(UpdateDependencyReq)
	if Res := BindAndValidate(req, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	depID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "invalid dependency id"})
	}

	// fetch existing dependency
	q := h.Server.DB.Queries
	existing, err := q.GetServiceDependencyByID(h.qCtx, depID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.JSON(http.StatusNotFound, types.Res[struct{}]{Message: "dependency not found"})
		}
		fmt.Println("error getting dependency:", err)
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "failed to get dependency"})
	}

	// env key format validation
	if !envKeyRegex.MatchString(req.EnvKey) {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "invalid env_key format"})
	}

	// self-dependency check
	if existing.SourceServiceID == req.TargetServiceID {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "cannot depend on itself"})
	}

	// same-instance validation
	if err := h.validateSameInstance(existing.SourceServiceID, req.TargetServiceID); err != nil {
		fmt.Println("update dependency validation error:", err)
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: err.Error()})
	}

	// target column validation per service type
	source, err := q.GetAppServiceOnly(h.qCtx, existing.SourceServiceID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "failed to validate source service"})
	}

	targets, err := q.GetDependencyTargets(h.qCtx, db.GetDependencyTargetsParams{
		InstanceID: source.InstanceID,
		ID:         existing.SourceServiceID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "failed to validate target service"})
	}

	var targetType string
	for _, t := range targets {
		if t.ID == req.TargetServiceID {
			targetType = t.ServiceType
			break
		}
	}

	if targetType == "" {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "target service not found"})
	}

	if !containsString(allowedTargetCols(targetType), req.TargetCol) {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "invalid target column for service type"})
	}

	// domain-specific validation
	if req.TargetCol == "domain" {
		app, err := q.GetAppServiceOnly(h.qCtx, req.TargetServiceID)
		if err != nil || !app.IsPublic || !app.Domain.Valid {
			return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "domain not available for target service"})
		}
	}

	updated, err := q.UpdateServiceDependency(h.qCtx, db.UpdateServiceDependencyParams{
		ID:              depID,
		TargetServiceID: req.TargetServiceID,
		TargetCol:       req.TargetCol,
		EnvKey:          req.EnvKey,
	})
	if err != nil {
		fmt.Println("error updating dependency:", err)
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "failed to update dependency"})
	}

	// resolve target name for response
	targetName := ""
	for _, t := range targets {
		if t.ID == updated.TargetServiceID {
			targetName = t.Name
			break
		}
	}

	return c.JSON(http.StatusOK, types.Res[ServiceDependencyRes]{
		Message: "dependency updated",
		Data: ServiceDependencyRes{
			ID:                updated.ID,
			SourceServiceID:   updated.SourceServiceID,
			TargetServiceID:   updated.TargetServiceID,
			TargetServiceName: targetName,
			TargetServiceType: targetType,
			TargetCol:         updated.TargetCol,
			EnvKey:            updated.EnvKey,
			CreatedAt:         updated.CreatedAt,
			UpdatedAt:         updated.UpdatedAt,
		},
	})
}

// remove a dependency link
//
// route: DELETE /api/service/app/dependencies/:id
func (h *DependencyHandler) DeleteServiceDependency(c *echo.Context) error {
	depID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "invalid dependency id"})
	}

	q := h.Server.DB.Queries
	if err := q.DeleteServiceDependency(h.qCtx, depID); err != nil {
		fmt.Println("error deleting dependency:", err)
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "failed to delete dependency"})
	}

	return c.JSON(http.StatusOK, types.Res[struct{}]{Message: "dependency deleted"})
}
