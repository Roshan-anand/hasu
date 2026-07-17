package services

import (
	deployjob "github.com/Roshan-anand/godploy/internal/jobs/deployment"
	"github.com/Roshan-anand/godploy/internal/jobs/logbroker"
	"github.com/Roshan-anand/godploy/internal/lib/database"
	"github.com/Roshan-anand/godploy/internal/lib/docker"
)

type Services struct {
	Deployment *deployjob.DeploymentService
	LogBroker  *logbroker.LogBrokerService
}

func NewServices(db *database.DataBase, docker *docker.DockerClient, badger *database.BadgerDB) *Services {
	logBrokerService := logbroker.NewLogBrokerService(db.Queries, badger)
	deploymentService := deployjob.NewDeploymentService(db, docker, logBrokerService, badger)
	return &Services{
		Deployment: deploymentService,
		LogBroker:  logBrokerService,
	}
}
