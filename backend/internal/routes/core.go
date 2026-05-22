package routes

import (
	"github.com/Roshan-anand/godploy/frontend"
	"github.com/Roshan-anand/godploy/internal/config"
	"github.com/Roshan-anand/godploy/internal/handlers"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/Roshan-anand/godploy/internal/middleware"
	"github.com/labstack/echo/v5"
)

// setup all routes
func SetupRoutes(srv *config.Server) (*echo.Echo, error) {
	h := handlers.NewHandeler(srv)
	m := middleware.NewMiddlewares(srv)
	e := echo.New()

	// initialize static file serving route
	uiFs, err := frontend.GetEmbedFS()
	if err != nil {
		return nil, err
	}
	e.StaticFS("/", uiFs)

	if srv.Config.AppEnv == types.DevMode {
		e.Use(m.GlobalMiddlewareCors())
	}

	api := e.Group("/api")
	public := api.Group("")
	protected := api.Group("")
	protected.Use(m.GlobalMiddlewareUser)

	// public routes
	public.GET("/health", h.Health.HealthCheck)
	public.POST("/sample", h.Health.SetGhApp)

	// initialize auth api routes
	auth := public.Group("/auth")
	auth.GET("/user", h.Auth.AuthUser, m.GlobalMiddlewareUser)
	auth.POST("/register", h.Auth.AppRegiter)
	auth.POST("/login", h.Auth.AppLogin)

	// initialize org api routes
	org := protected.Group("/org")
	org.GET("", h.Org.GetAllOrgs)
	org.POST("", h.Org.CreateOrg)
	org.DELETE("", h.Org.DeleteOrg)
	org.POST("/switch", h.Org.SwitchOrg)

	// initialize service api routes
	service := protected.Group("/service")
	service.GET("", h.Service.GetAllServices)
	service.GET("/deployment", h.Service.GetServiceDeployments)
	service.DELETE("/deployment", h.Service.DeleteServiceDeployment)
	service.GET("/deployment/logs", h.Service.SubscribeServiceDeploymentLogs)
	service.GET("/logs", h.Service.GetServiceLogs)

	psql := service.Group("/psql")
	psql.GET("/:id", h.Service.GetPsqlServiceById)
	psql.POST("", h.Service.CreatePsqlService)
	psql.DELETE("", h.Service.DeletePsqlService)
	psql.POST("/deploy", h.Service.DeployPsqlService)
	psql.POST("/stop", h.Service.StopPsqlService)

	app := service.Group("/app")
	app.GET("/:id", h.Service.GetAppServiceById)
	app.POST("", h.Service.CreateAppService)
	app.DELETE("", h.Service.DeleteAppService)
	app.GET("/domain", h.Service.GetBranchDomain)
	app.PUT("/domain", h.Service.UpdateAppServiceDomain)
	app.GET("/env", h.Service.GetServiceEnv)
	app.PUT("/env", h.Service.UpdateAppServiceEnv)
	app.POST("/rebuild", h.Service.RebuildAppService)
	app.POST("/rollback", h.Service.RollbackAppService)

	gh := protected.Group("/provider/github")
	gh.GET("/app/create", h.Git.CreateGithubApp)
	gh.GET("/app/list", h.Git.GetAllGithubApps)
	gh.DELETE("/app", h.Git.DeleteGithubApp)
	gh.GET("/repo/list", h.Git.GetGithubRepoList)
	ghPublic := public.Group("/provider/github")
	ghPublic.GET("/app/callback", h.Git.CreateGithubAppCallback)
	ghPublic.GET("/app/setup", h.Git.SetupGithubApp)
	ghPublic.POST("/webhook", h.Git.GithubWebhook)

	return e, nil
}
