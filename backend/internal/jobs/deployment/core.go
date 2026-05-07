package deploymentjob

import (
	"context"

	"github.com/Roshan-anand/godploy/internal/config"
)

type worker struct {
	Server *config.Server
	qCtx   context.Context
}

func NewJob(s *config.Server) *worker {
	return &worker{
		Server: s,
		qCtx:   context.Background(),
	}
}
