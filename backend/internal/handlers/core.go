package handlers

import "github.com/Roshan-anand/godploy/internal/config"

type Handler struct {
	Health       *HealthHandler
	Auth         *AuthHandler
	Profile      *ProfileHandler
	Service      *ServiceHandler
	PsqlService  *PsqlServiceHandler
	RedisService *RedisServiceHandler
	Git          *GitHandler
	Org          *OrgHandler
	Project      *ProjectHandler
	Deployment   *DeploymentHandler
	Volume       *VolumeHandler
	Instance     *InstanceHandler
}

func NewHandeler(srv *config.Server) *Handler {
	return &Handler{
		Health:       InitHealthHandlers(srv),
		Auth:         InitAuthHandlers(srv),
		Profile:      InitProfileHandlers(srv),
		Service:      InitServiceHandlers(srv),
		PsqlService:  InitPsqlServiceHandlers(srv),
		RedisService: InitRedisServiceHandlers(srv),
		Git:          InitGitHandlers(srv),
		Org:          InitOrgHandlers(srv),
		Project:      InitProjectHandlers(srv),
		Deployment:   InitDeploymentHandlers(srv),
		Volume:       InitVolumeHandlers(srv),
		Instance:     InitInstanceHandlers(srv),
	}
}
