package handlers

import (
	"context"
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
		return c.JSON(http.StatusInternalServerError, types.Res{
			Message: "internal server error",
		})
	}

	return c.JSON(http.StatusOK, orgs)
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
		return c.JSON(http.StatusInternalServerError, types.Res{Message: "internal server error"})
	} else if !isAdmin {
		return c.JSON(http.StatusForbidden, types.Res{Message: "admin access required"})
	}

	if exists, err := q.CheckOrgExists(h.qCtx, db.CheckOrgExistsParams{
		UserEmail: u.Email,
		OrgName:   b.Name,
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res{Message: "internal server error"})
	} else if exists {
		return c.JSON(http.StatusConflict, types.Res{Message: "Organization with this name already exists"})
	}

	org, err := q.CreateOrg(h.qCtx, db.CreateOrgParams{
		ID:   security.GeneratePrimaryKey(),
		Name: b.Name,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res{Message: "Failed to create organization"})
	}

	if err := q.LinkUserNOrg(h.qCtx, db.LinkUserNOrgParams{
		UserEmail:      u.Email,
		OrganizationID: org.ID,
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res{Message: "Failed to link user to organization"})
	}

	return c.JSON(http.StatusOK, org)
}

// delete an organization for admin users
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
		return c.JSON(http.StatusInternalServerError, types.Res{Message: "internal server error"})
	}

	switch {
	case user.OrgID == b.OrgID:
		return c.JSON(http.StatusConflict, types.Res{Message: "Cannot delete the current organization"})
	case user.Role != types.AdminRole:
		return c.JSON(http.StatusForbidden, types.Res{Message: "admin access required"})
	}

	if err := q.DeleteOrg(h.qCtx, b.OrgID); err != nil {
		fmt.Printf("Error deleting organization: %v\n", err)
		return c.JSON(http.StatusInternalServerError, types.Res{Message: "Failed to delete organization"})
	}

	return c.JSON(http.StatusOK, types.Res{Message: "Organization deleted successfully"})
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
		return c.JSON(http.StatusInternalServerError, types.Res{Message: "internal server error"})
	} else if !exists {
		return c.JSON(http.StatusForbidden, types.Res{Message: "User does not have access to the organization"})
	}

	user, err := q.GetUserByEmail(h.qCtx, u.Email)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res{Message: "Failed to get user"})
	}

	if err := q.UpdateCurrentOrg(h.qCtx, db.UpdateCurrentOrgParams{
		CurrentOrgID: b.OrgID,
		ID:           user.ID,
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res{Message: "Failed to switch organization"})
	}

	org, err := q.GetOrgById(h.qCtx, b.OrgID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res{Message: "Failed to get organization"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":   org.ID,
		"name": org.Name,
	})
}
