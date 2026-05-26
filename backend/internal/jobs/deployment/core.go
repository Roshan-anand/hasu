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

// starts all the deployment workers
func (w *worker) StartAllDeploymentWorker() {
	q := w.Server.DeploymentQ
	go w.PullWorker(context.Background(), q.PullQueue)
	go w.BuildWorker(context.Background(), q.BuildQueue)
	go w.DeployWorker(context.Background(), q.DeployQueue)
	go w.ReDeployWorker(context.Background(), q.RedeployQueue)
}
