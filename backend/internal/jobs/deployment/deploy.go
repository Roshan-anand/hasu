package deploymentjob

import (
	"context"
	"fmt"

	deploymentqueue "github.com/Roshan-anand/godploy/internal/jobs/deployment/queue"
	logbrokerqueue "github.com/Roshan-anand/godploy/internal/jobs/logbroker/queue"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/docker/docker/api/types/swarm"
)

// responsible for pulling code and storing it local
func (w *worker) DeployWorker(ctx context.Context, data chan *deploymentqueue.DeployJobData) {
	for {
		select {
		case d, ok := <-data:
			if !ok {
				fmt.Println("DeployWorker: data channel closed, exiting")
				return
			}

			l := w.Server.LogBrokerQ
			docker := w.Server.Docker.Client

			l.PublishLog(&logbrokerqueue.PubData{
				ID:  d.DeploymentID,
				Msg: getTitle("Deploying  the service " + d.SwarmService),
			})

			// create network if not exist
			if err := w.Server.Docker.CreateNetwork(d.NetworkName); err != nil {
				fmt.Printf("DeployWorker: error creating network: %v\n", err)
				l.EndLogs(&logbrokerqueue.EndLogData{
					DeploymentID: d.DeploymentID,
					Status:       types.DeploymentError,
					Message:      err.Error(),
				})
				continue
			}

			// get the service spec
			spec := getBaseSpec(d.ImgName, d.NetworkName, d.SwarmService, d.Env, d.IsPublic)

			_, err := docker.ServiceCreate(context.Background(), *spec, swarm.ServiceCreateOptions{})
			if err != nil {
				fmt.Printf("DeployWorker: error creating service: %v\n", err)
				l.EndLogs(&logbrokerqueue.EndLogData{
					DeploymentID: d.DeploymentID,
					Status:       types.DeploymentError,
					Message:      err.Error(),
				})
				continue
			}

			fmt.Println("finished deploying :", d.SwarmService)
			// end the logs
			l.EndLogs(&logbrokerqueue.EndLogData{
				DeploymentID: d.DeploymentID,
				Status:       types.DeploymentReady,
				Message:      getTitle("successfully deployed"),
			})

		case <-ctx.Done():
			fmt.Println("DeployWorker: context cancelled, exiting")
			return
		}
	}
}
