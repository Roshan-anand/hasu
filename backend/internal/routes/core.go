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

	// initialize profile api routes
	profile := protected.Group("/profile")
	profile.GET("", h.Profile.GetProfile)
	profile.PUT("", h.Profile.UpdateProfile)
	profile.PUT("/password", h.Profile.ChangePassword)

	// initialize org api routes
	org := protected.Group("/org")
	org.GET("", h.Org.GetAllOrgs)
	org.POST("", h.Org.CreateOrg)
	org.PUT("/rename", h.Org.RenameOrg)
	org.DELETE("", h.Org.DeleteOrg)
	org.POST("/switch", h.Org.SwitchOrg)
	org.GET("/projects", h.Org.GetOrgProjects)
	org.GET("/volumes", h.Org.GetOrgVolumes)
	org.PUT("/transfer-volume", h.Org.TransferVolume)

	// initialize project api routes
	project := protected.Group("/project")
	project.GET("", h.Project.GetAllProject)
	project.POST("", h.Project.CreateProject)
	project.DELETE("", h.Project.DeleteProject)
	project.PUT("/transfer", h.Project.TransferProject)
	project.PUT("/rename", h.Project.RenameProject)

	instance := protected.Group("/instance")
	instance.GET("", h.Instance.GetAllInstance)
	instance.PUT("/rename", h.Instance.RenameInstance)

	volume := protected.Group("/volume")
	volume.GET("", h.Service.GetAllVolume)
	volume.GET("/:type", h.Service.GetAllVolumeByType)
	volume.PATCH("", h.Service.RenameVolume)
	volume.DELETE("", h.Service.DeleteVolume)

	// initialize service api routes
	service := protected.Group("/service")
	service.GET("/all", h.Service.GetAllServices)
	service.GET("/:name", h.Service.GetServiceID)
	service.GET("/deployment", h.Deployment.GetServiceDeployments)
	service.DELETE("/deployment", h.Deployment.DeleteServiceDeployment)
	service.GET("/deployment/logs", h.Deployment.SubscribeServiceDeploymentLogs)
	service.GET("/logs", h.Service.GetServiceLogs)
	service.POST("/stop", h.Service.StopPredefService)
	service.POST("/start", h.Service.StartPredefService)

	psql := service.Group("/psql")
	psql.GET("/:id", h.Service.GetPsqlServiceById)
	psql.POST("", h.Service.CreatePsqlService)
	psql.PUT("", h.Service.UpdatePsqlServiceDetails)
	psql.POST("/redeploy", h.Service.RedeployPsqlService)
	psql.DELETE("", h.Service.DeletePsqlService)

	redis := service.Group("/redis")
	redis.GET("/:id", h.Service.GetRedisServiceById)
	redis.POST("", h.Service.CreateRedisService)
	redis.PUT("", h.Service.UpdateRedisServiceDetails)
	redis.POST("/redeploy", h.Service.RedeployRedisService)
	redis.DELETE("", h.Service.DeleteRedisService)

	app := service.Group("/app")
	app.GET("/:id", h.Service.GetAppServiceById)
	app.POST("", h.Service.CreateAppService)
	app.DELETE("", h.Service.DeleteAppService)
	app.GET("/domain", h.Service.GetDomainPort)
	app.PUT("/domain", h.Service.UpdateAppServiceDomain)
	app.GET("/env", h.Service.GetServiceEnv)
	app.PUT("/env", h.Service.UpdateAppServiceEnv)
	app.GET("/settings", h.Service.GetAppServiceSettings)
	app.POST("/scale", h.Service.ScaleAppService)
	app.POST("/pause", h.Service.PauseAppService)
	app.POST("/resume", h.Service.ResumeAppService)
	app.POST("/rebuild", h.Deployment.RebuildAppService)
	app.POST("/rollback", h.Deployment.RollbackAppService)

	gh := protected.Group("/provider/github")
	gh.GET("/app/create", h.Git.CreateGithubApp)
	gh.GET("/app/list", h.Git.GetAllGithubApps)
	gh.DELETE("/app", h.Git.DeleteGithubApp)
	gh.GET("/repo/list", h.Git.GetGithubRepoList)
	gh.GET("/pr/list", h.Git.GetGithubPRList)
	gh.GET("/pr/instance", h.Git.GetGithubPRListByInstance)
	ghPublic := public.Group("/provider/github")
	ghPublic.GET("/app/callback", h.Git.CreateGithubAppCallback)
	ghPublic.GET("/app/setup", h.Git.SetupGithubApp)
	ghPublic.POST("/webhook", h.Git.GithubWebhook)

	return e, nil
}
