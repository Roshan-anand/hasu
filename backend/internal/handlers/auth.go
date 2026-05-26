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

const (
	MAX_PASS_COUNT = 15
	MIN_PASS_COUNT = 8
)

type AuthHandler struct {
	Server   *config.Server
	Validate *validator.Validate
	qCtx     context.Context
}

type RegisterReq struct {
	Name     string `json:"name" validate:"required,min=3,max=50"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=15"`
	OrgName  string `json:"org_name" validate:"required,min=3,max=50"`
}

type LoginReq struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=15"`
}

type AuthRes struct {
	Name    string    `json:"name"`
	Email   string    `json:"email"`
	OrgId   uuid.UUID `json:"org_id"`
	OrgName string    `json:"org_name"`
}

func InitAuthHandlers(s *config.Server) *AuthHandler {
	return &AuthHandler{
		Server:   s,
		Validate: validator.New(),
		qCtx:     context.Background(),
	}
}

// check if user is authenticated
//
// route: GET /api/auth/user
func (h *AuthHandler) AuthUser(c *echo.Context) error {
	u, ok := c.Get(h.Server.Config.EchoCtxUserKey).(auth.AuthUser)
	q := h.Server.DB.Queries

	if !ok {
		exists, err := q.AdminExists(h.qCtx)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Internal Sever Error"})
		}

		if !exists {
			return c.JSON(http.StatusForbidden, types.Res[struct{}]{Message: "No admin registered"})
		}
		return c.JSON(http.StatusUnauthorized, types.Res[struct{}]{Message: "Unauthorized"})
	}

	org, err := q.GetCurrentOrg(h.qCtx, u.Email)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Internal Sever Error"})
	}

	return c.JSON(http.StatusOK, types.Res[AuthRes]{Message: "", Data: AuthRes{
		Name:    u.Name,
		Email:   u.Email,
		OrgId:   org.ID,
		OrgName: org.Name,
	}})
}

// register a new application
//
// route: POST /api/auth/register
func (h *AuthHandler) AppRegiter(c *echo.Context) error {
	b := new(RegisterReq)

	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	q := h.Server.DB.Queries

	// check if admin user already exists
	if exist, err := q.AdminExists(h.qCtx); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Internal Server Error"})
	} else if exist {
		return c.JSON(http.StatusBadRequest, types.Res[struct{}]{Message: "Admin User Already Exists"})
	}

	// hash password
	hPass, err := security.HashPassword(b.Password)
	if err != nil {
		fmt.Println(err)
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Internal Server Error"})
	}

	// start a db transaction
	tx, err := h.Server.DB.Pool.BeginTx(context.Background(), nil)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Internal Server Error"})
	}
	tq := q.WithTx(tx)

	// create organization first (user needs orgId at insert time)
	org, err := tq.CreateOrg(h.qCtx, db.CreateOrgParams{
		ID:   security.GeneratePrimaryKey(),
		Name: b.OrgName,
	})
	if err != nil {
		tx.Rollback()
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Internal Server Error"})
	}

	// register new admin user
	uId, err := tq.CreateUser(h.qCtx, db.CreateUserParams{
		ID:           security.GeneratePrimaryKey(),
		Name:         b.Name,
		Email:        b.Email,
		HashPass:     hPass,
		Role:         types.AdminRole,
		CurrentOrgID: org.ID,
	})
	if err != nil {
		tx.Rollback()
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Internal Server Error"})
	}

	// link user with organization
	if err := tq.LinkUserNOrg(h.qCtx, db.LinkUserNOrgParams{
		UserEmail:      b.Email,
		OrganizationID: org.ID,
	}); err != nil {
		tx.Rollback()
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Internal Server Error"})
	}

	if err := tx.Commit(); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Internal Server Error"})
	}

	// set cookies
	auth.SetSessionCookies(h.Server, c, uId)
	auth.SetJwtCookie(h.Server, c, auth.AuthUser{Email: b.Email, Name: b.Name, Role: types.AdminRole})

	return c.JSON(http.StatusOK, types.Res[AuthRes]{Message: "", Data: AuthRes{
		Name:    b.Name,
		Email:   b.Email,
		OrgId:   org.ID,
		OrgName: b.OrgName,
	}})
}

// login user
//
// route: POST /api/auth/login
func (h *AuthHandler) AppLogin(c *echo.Context) error {
	b := new(LoginReq)

	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	// get the user
	u, err := h.Server.DB.Queries.GetUserByEmail(h.qCtx, b.Email)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, types.Res[struct{}]{Message: "user not found"})
	}

	// check password
	if !security.CheckPasswordHash(b.Password, u.HashPass) {
		return c.JSON(http.StatusUnauthorized, types.Res[struct{}]{Message: "invalid credentials"})
	}

	// set cookies
	auth.SetSessionCookies(h.Server, c, u.ID)
	auth.SetJwtCookie(h.Server, c, auth.AuthUser{Email: u.Email, Name: u.Name, Role: u.Role})

	return c.JSON(http.StatusOK, types.Res[AuthRes]{Message: "", Data: AuthRes{
		Name:    u.Name,
		Email:   u.Email,
		OrgId:   u.ID,
		OrgName: u.OrgName,
	}})
}
