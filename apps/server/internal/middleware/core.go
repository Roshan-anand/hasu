package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Roshan-anand/godploy/internal/config"
	"github.com/Roshan-anand/godploy/internal/lib/auth"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

type Middlewares struct {
	Server *config.Server
	Ctx    context.Context
}

// return new middlewares instance
func NewMiddlewares(s *config.Server) *Middlewares {
	return &Middlewares{Server: s, Ctx: context.Background()}
}

// global middleware cors applicable to all routes.
// in dev, AllowCredentials is required so the browser accepts cross-origin responses that set cookies.
func (m *Middlewares) GlobalMiddlewareCors() echo.MiddlewareFunc {
	cfg := middleware.CORSConfig{
		AllowOrigins:     []string{m.Server.Config.WebUrl},
		AllowCredentials: true,
	}

	return middleware.CORSWithConfig(cfg)
}

// global middleware user applicable to all routes
func (m *Middlewares) GlobalMiddlewareUser(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		unAuthErr := types.Res[struct{}]{
			Message: "unauthorized access"}
		secret := m.Server.Config.JwtSecret

		// checks for the JWT
		jwt, err := c.Cookie(m.Server.Config.SessionDataName)
		if err == nil {
			claims, err := auth.VerifyJWT(jwt.Value, secret)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, unAuthErr)
			}

			// set user in context
			c.Set(m.Server.Config.EchoCtxUserKey, claims.AuthUser)
			return next(c)
		}

		// check for session token
		token, err := c.Cookie(m.Server.Config.SessionTokenName)
		if err == nil {
			sData, err := m.Server.DB.Queries.GetSessionByToken(m.Ctx, token.Value)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, unAuthErr)
			}

			// session expired then remove
			if time.Now().After(sData.ExpiresAt) {
				if err := m.Server.DB.Queries.RemoveSessionByUID(context.Background(), sData.ID); err != nil {
					fmt.Println("remove session by uid error:", err)
				}
				return c.JSON(http.StatusUnauthorized, unAuthErr)
			}

			// if expire date is les then 30% then extend expiry
			diff := sData.ExpiresAt.Sub(time.Now())
			if diff < auth.SESSION_DATA_EXPIRY_DAY*30/100 {
				auth.SetSessionCookies(m.Server, c, sData.ID)
			}

			u := auth.AuthUser{
				Email: sData.Email,
				Name:  sData.Name,
				Role:  sData.Role,
			}

			// set new jwt cookie
			auth.SetJwtCookie(m.Server, c, u)

			// set user in context
			c.Set(m.Server.Config.EchoCtxUserKey, u)
			return next(c)
		}

		// check if path is /api/user
		if c.Path() == "/api/auth/user" {
			return next(c)
		}

		// no auth found
		return c.JSON(http.StatusUnauthorized, unAuthErr)
	}
}
