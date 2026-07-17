package handlers

import (
	"context"
	"database/sql"
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

type ProfileHandler struct {
	Server   *config.Server
	Validate *validator.Validate
	qCtx     context.Context
}

type ProfileRes struct {
	ID        uuid.UUID      `json:"id"`
	Name      string         `json:"name"`
	Email     string         `json:"email"`
	Role      types.UserRole `json:"role"`
	Avatar    string         `json:"avatar"`
	CreatedAt string         `json:"created_at"`
}

type UpdateProfileReq struct {
	Name   string `json:"name" validate:"required,min=3,max=50"`
	Email  string `json:"email" validate:"required,email"`
	Avatar string `json:"avatar"`
}

type ChangePasswordReq struct {
	OldPassword string `json:"old_password" validate:"required,min=8,max=15"`
	NewPassword string `json:"new_password" validate:"required,min=8,max=15"`
}

func InitProfileHandlers(s *config.Server) *ProfileHandler {
	return &ProfileHandler{
		Server:   s,
		Validate: validator.New(),
		qCtx:     context.Background(),
	}
}

// GetProfile returns the authenticated user's profile
//
// route: GET /api/profile
func (h *ProfileHandler) GetProfile(c *echo.Context) error {
	u, ok := c.Get(h.Server.Config.EchoCtxUserKey).(auth.AuthUser)
	if !ok {
		return c.JSON(http.StatusUnauthorized, types.Res[struct{}]{Message: "unauthorized"})
	}

	q := h.Server.DB.Queries

	user, err := q.GetUserByEmail(h.qCtx, u.Email)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Internal Server Error"})
	}

	profile, err := q.GetUserProfile(h.qCtx, user.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Internal Server Error"})
	}

	return c.JSON(http.StatusOK, types.Res[ProfileRes]{
		Message: "",
		Data: ProfileRes{
			ID:        profile.ID,
			Name:      profile.Name,
			Email:     profile.Email,
			Role:      profile.Role,
			Avatar:    profile.Avatar.String,
			CreatedAt: profile.CreatedAt.Format("2006-01-02T15:04:05Z"),
		},
	})
}

// UpdateProfile updates the authenticated user's profile (name, email, avatar)
//
// route: PUT /api/profile
func (h *ProfileHandler) UpdateProfile(c *echo.Context) error {
	u, ok := c.Get(h.Server.Config.EchoCtxUserKey).(auth.AuthUser)
	if !ok {
		return c.JSON(http.StatusUnauthorized, types.Res[struct{}]{Message: "unauthorized"})
	}

	b := new(UpdateProfileReq)
	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	q := h.Server.DB.Queries

	user, err := q.GetUserByEmail(h.qCtx, u.Email)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Internal Server Error"})
	}

	if err := q.UpdateUserProfile(h.qCtx, db.UpdateUserProfileParams{
		ID:    user.ID,
		Name:  b.Name,
		Email: b.Email,
		Avatar: sql.NullString{
			Valid:  true,
			String: b.Avatar,
		},
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Internal Server Error"})
	}

	return c.JSON(http.StatusOK, types.Res[struct{}]{Message: "Profile updated successfully"})
}

// ChangePassword updates the authenticated user's password after validating the old password
//
// route: PUT /api/profile/password
func (h *ProfileHandler) ChangePassword(c *echo.Context) error {
	u, ok := c.Get(h.Server.Config.EchoCtxUserKey).(auth.AuthUser)
	if !ok {
		return c.JSON(http.StatusUnauthorized, types.Res[struct{}]{Message: "unauthorized"})
	}

	b := new(ChangePasswordReq)
	if Res := BindAndValidate(b, c, h.Validate); Res != nil {
		return c.JSON(http.StatusBadRequest, Res)
	}

	q := h.Server.DB.Queries

	user, err := q.GetUserByEmail(h.qCtx, u.Email)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Internal Server Error"})
	}

	// validate old password
	if !security.CheckPasswordHash(b.OldPassword, user.HashPass) {
		return c.JSON(http.StatusUnauthorized, types.Res[struct{}]{Message: "invalid credentials"})
	}

	// hash new password
	newHash, err := security.HashPassword(b.NewPassword)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Internal Server Error"})
	}

	if err := q.UpdateUserPassword(h.qCtx, db.UpdateUserPasswordParams{
		ID:       user.ID,
		HashPass: newHash,
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "Internal Server Error"})
	}

	return c.JSON(http.StatusOK, types.Res[struct{}]{Message: "Password changed successfully"})
}
