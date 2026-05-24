package handlers

import "github.com/Roshan-anand/godploy/internal/config"

type Handler struct {
	Health      *HealthHandler
	Auth        *AuthHandler
	Service     *ServiceHandler
	PsqlService *PsqlServiceHandler
	Git         *GitHandler
	Org         *OrgHandler
	Project     *ProjectHandler
}

func NewHandeler(srv *config.Server) *Handler {
	return &Handler{
		Health:      InitHealthHandlers(srv),
		Auth:        InitAuthHandlers(srv),
		Service:     InitServiceHandlers(srv),
		PsqlService: InitPsqlServiceHandlers(srv),
		Git:         InitGitHandlers(srv),
		Org:         InitOrgHandlers(srv),
		Project:     InitProjectHandlers(srv),
	}
}
