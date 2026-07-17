package handlers

import (
	"context"
	"errors"
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
	"github.com/mattn/go-sqlite3"
)

type OrgHandler struct {
	Server   *config.Server
	Validate *validator.Validate
	qCtx     context.Context
}

type CreateOrgReq struct {
	Name string `json:"name" validate:"required,min=3"`
}

type OrgReq struct {
	OrgID uuid.UUID `json:"org_id" validate:"required"`
}

type RenameOrgReq struct {
	OrgID uuid.UUID `json:"org_id" validate:"required"`
	Name  string    `json:"name" validate:"required,min=3"`
}

type SwitchOrgRes struct {
	OrgID uuid.UUID `json:"id"`
	Name  string    `json:"name"`
}

type TransferVolumeReq struct {
	VolumeID    uuid.UUID `json:"volume_id" validate:"required"`
	TargetOrgID uuid.UUID `json:"target_org_id" validate:"required"`
	SourceOrgID uuid.UUID `json:"source_org_id" validate:"required"`
}

func InitOrgHandlers(s *config.Server) *OrgHandler {
	return &OrgHandler{
		Server:   s,
		Validate: validator.New(),
		qCtx:     context.Background(),
	}
}

// get all organizations accessible to the authenticated user
//
// route: GET /api/org
func (h *OrgHandler) GetAllOrgs(c *echo.Context) error {
	u := c.Get(h.Server.Config.EchoCtxUserKey).(auth.AuthUser)

	orgs, err := h.Server.DB.Queries.GetAllOrg(h.qCtx, u.Email)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{
			Message: "internal server error",
		})
	}

	return c.JSON(http.StatusOK, types.Res[[]db.GetAllOrgRow]{
		Message: "",
		Data:    orgs,
	})
}

// create a new organization and link it to the authenticated user
//
// route: POST /api/org
func (h *OrgHandler) CreateOrg(c *echo.Context) error {
	u := c.Get(h.Server.Config.EchoCtxUserKey).(auth.AuthUser)
	b := new(CreateOrgReq)

	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	q := h.Server.DB.Queries
	if isAdmin, err := q.IsUserAdmin(h.qCtx, u.Email); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "internal server error"})
	} else if !isAdmin {
		return c.JSON(http.StatusForbidden, types.Res[struct{}]{Message: "admin access required"})
	}

	tx, err := h.Server.DB.Pool.BeginTx(context.Background(), nil)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to start transaction"})
	}
	tq := q.WithTx(tx)

	org, err := tq.CreateOrg(h.qCtx, db.CreateOrgParams{
		ID:   security.GeneratePrimaryKey(),
		Name: b.Name,
	})
	if err != nil {
		tx.Rollback()
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			return c.JSON(http.StatusConflict, types.Res[struct{}]{Message: "Organization with this name already exists"})
		}
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to create organization"})
	}

	if err := tq.LinkUserNOrg(h.qCtx, db.LinkUserNOrgParams{
		UserEmail:      u.Email,
		OrganizationID: org.ID,
	}); err != nil {
		tx.Rollback()
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to link user to organization"})
	}

	if err := tx.Commit(); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to commit transaction"})
	}

	return c.JSON(http.StatusOK, types.Res[db.CreateOrgRow]{Message: "", Data: org})
}

// get projects for an organization (used for delete warning display)
//
// route: GET /api/org/projects?org_id=
func (h *OrgHandler) GetOrgProjects(c *echo.Context) error {
	orgID, err := uuid.Parse(c.QueryParam("org_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "invalid organization id"})
	}

	projects, err := h.Server.DB.Queries.GetProjectsByOrgId(h.qCtx, orgID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "internal server error"})
	}

	return c.JSON(http.StatusOK, types.Res[[]db.GetProjectsByOrgIdRow]{Message: "", Data: projects})
}

// rename an organization
//
// route: PUT /api/org/rename
func (h *OrgHandler) RenameOrg(c *echo.Context) error {
	u := c.Get(h.Server.Config.EchoCtxUserKey).(auth.AuthUser)
	b := new(RenameOrgReq)

	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	q := h.Server.DB.Queries

	// check if user has access to the org
	if exists, err := q.CheckUserOrgExists(h.qCtx, db.CheckUserOrgExistsParams{
		UserEmail:      u.Email,
		OrganizationID: b.OrgID,
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "internal server error"})
	} else if !exists {
		return c.JSON(http.StatusForbidden, types.Res[struct{}]{Message: "User does not have access to the organization"})
	}

	org, err := q.RenameOrg(h.qCtx, db.RenameOrgParams{
		Name: b.Name,
		ID:   b.OrgID,
	})
	if err != nil {
		if h.Server.DB.IsUniqueConstraintError(err) {
			return c.JSON(http.StatusConflict, types.Res[struct{}]{Message: "Organization with this name already exists"})
		}
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to rename organization"})
	}

	return c.JSON(http.StatusOK, types.Res[db.RenameOrgRow]{Message: "", Data: org})
}

// delete an organization for admin users
// Validates no services are running, cleans up Docker networks + orphan volumes, then deletes the org
//
// route: DELETE /api/org
func (h *OrgHandler) DeleteOrg(c *echo.Context) error {
	u := c.Get(h.Server.Config.EchoCtxUserKey).(auth.AuthUser)
	b := new(OrgReq)

	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	q := h.Server.DB.Queries
	user, err := q.GetUserByEmail(h.qCtx, u.Email)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "internal server error"})
	}

	switch {
	case user.OrgID == b.OrgID:
		return c.JSON(http.StatusConflict, types.Res[struct{}]{Message: "Cannot delete the current organization"})
	case user.Role != types.AdminRole:
		return c.JSON(http.StatusForbidden, types.Res[struct{}]{Message: "admin access required"})
	}

	// single query to get all instances across all projects with their service count
	instances, err := q.GetAllInstancesByOrgId(h.qCtx, b.OrgID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to get organization instances"})
	}

	// check if any instance has running services
	for _, inst := range instances {
		if inst.ServiceCount > 0 {
			return c.JSON(http.StatusConflict, types.Res[struct{}]{
				Message: fmt.Sprintf("Instance \"%s\" in project \"%s\" has %d service(s) running. Remove all services before deleting the organization.", inst.Name, inst.ProjectName, inst.ServiceCount),
			})
		}
	}

	// collect and remove all Docker networks from all projects
	projects, err := q.GetProjectsByOrgId(h.qCtx, b.OrgID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to get organization projects"})
	}

	for _, project := range projects {
		networks, err := q.GetAllNetworksByProjectId(h.qCtx, project.ID)
		if err != nil {
			continue
		}
		go h.Server.Docker.RemoveNetworks(networks)
	}

	// get and remove all orphan volumes for the org
	orphanVolumes, err := q.GetAllOrphanVolumesByOrgID(h.qCtx, b.OrgID)
	if err != nil {
		fmt.Printf("Error fetching orphan volumes: %v\n", err)
	}

	for _, vol := range orphanVolumes {
		if err := h.Server.Docker.RemoveVolumes([]string{vol.Volume}); err != nil {
			fmt.Printf("Warning: failed to remove volume %s: %v\n", vol.Volume, err)
		}
		// delete from DB (will also be handled by CASCADE, but explicit is cleaner)
		if err := q.DeleteOrphanVolume(h.qCtx, db.DeleteOrphanVolumeParams{
			Volume:         vol.Volume,
			OrganizationID: b.OrgID,
		}); err != nil {
			fmt.Printf("Warning: failed to delete orphan volume record %s: %v\n", vol.Volume, err)
		}
	}

	// delete the org — FK CASCADE handles projects → instances → services → deployments
	if err := q.DeleteOrg(h.qCtx, b.OrgID); err != nil {
		fmt.Printf("Error deleting organization: %v\n", err)
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to delete organization"})
	}

	return c.JSON(http.StatusOK, types.Res[struct{}]{Message: "Organization deleted successfully"})
}

// get all orphan volumes for an organization (used in delete confirmation UI)
//
// route: GET /api/org/volumes?org_id=
func (h *OrgHandler) GetOrgVolumes(c *echo.Context) error {
	orgID, err := uuid.Parse(c.QueryParam("org_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "invalid organization id"})
	}

	volumes, err := h.Server.DB.Queries.GetOrphanVolumesByOrgId(h.qCtx, orgID)
	if err != nil {
		fmt.Printf("Error fetching orphan volumes: %v\n", err)
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "internal server error"})
	}

	return c.JSON(http.StatusOK, types.Res[[]db.OrphanVolume]{Message: "", Data: volumes})
}

// transfer an orphan volume to another organization
//
// route: PUT /api/org/transfer-volume
func (h *OrgHandler) TransferVolume(c *echo.Context) error {
	u := c.Get(h.Server.Config.EchoCtxUserKey).(auth.AuthUser)
	b := new(TransferVolumeReq)

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

	// transfer the orphan volume (source org provided by the caller, e.g. the org being deleted)
	if err := q.TransferOrphanVolume(h.qCtx, db.TransferOrphanVolumeParams{
		OrganizationID:   b.TargetOrgID,
		ID:               b.VolumeID,
		OrganizationID_2: b.SourceOrgID,
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to transfer volume"})
	}

	return c.JSON(http.StatusOK, types.Res[struct{}]{Message: "Volume transferred successfully"})
}

// switch the authenticated user's current organization
//
// route: POST /api/org/switch
func (h *OrgHandler) SwitchOrg(c *echo.Context) error {
	u := c.Get(h.Server.Config.EchoCtxUserKey).(auth.AuthUser)
	b := new(OrgReq)

	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	q := h.Server.DB.Queries

	// check if the user is part of the organization they are trying to switch to
	if exists, err := q.CheckUserOrgExists(context.Background(), db.CheckUserOrgExistsParams{
		UserEmail:      u.Email,
		OrganizationID: b.OrgID,
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "internal server error"})
	} else if !exists {
		return c.JSON(http.StatusForbidden, types.Res[struct{}]{Message: "User does not have access to the organization"})
	}

	user, err := q.GetUserByEmail(h.qCtx, u.Email)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to get user"})
	}

	if err := q.UpdateCurrentOrg(h.qCtx, db.UpdateCurrentOrgParams{
		CurrentOrgID: b.OrgID,
		ID:           user.ID,
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to switch organization"})
	}

	org, err := q.GetOrgById(h.qCtx, b.OrgID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Failed to get organization"})
	}

	return c.JSON(http.StatusOK, types.Res[db.GetOrgByIdRow]{
		Message: "",
		Data:    org,
	})
}
