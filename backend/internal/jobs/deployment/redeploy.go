package deploymentjob

import (
	"context"
	"fmt"

	deploymentqueue "github.com/Roshan-anand/godploy/internal/jobs/deployment/queue"
	logbrokerqueue "github.com/Roshan-anand/godploy/internal/jobs/logbroker/queue"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/moby/moby/client"
)

// responsible for pulling code and storing it local
func (w *worker) ReDeployWorker(ctx context.Context, data chan *deploymentqueue.RedeployJobData) {
	for {
		select {
		case d, ok := <-data:
			if !ok {
				fmt.Println("DeployWorker: data channel closed, exiting")
				return
			}

			docker := w.Server.Docker.Client

			// get the swarm service spec
			res, err := docker.ServiceInspect(w.qCtx, d.SwarmServiceName, client.ServiceInspectOptions{})
			if err != nil {
				fmt.Printf("DeployWorker: error inspecting service: %v\n", err)
				w.Server.LogBrokerQ.EndLogs(&logbrokerqueue.EndLogData{
					DeploymentID: d.DeploymentID,
					Status:       types.DeploymentError,
					Message:      err.Error(),
				})
				continue
			}
			version := res.Service.Version
			spec := res.Service.Spec

			// update the image and env
			spec.TaskTemplate.ContainerSpec.Image = d.ImgName
			if len(d.Env) > 0 {
				spec.TaskTemplate.ContainerSpec.Env = d.Env
			}

			// update the service with the new spec
			if _, err := docker.ServiceUpdate(w.qCtx, d.SwarmServiceName, client.ServiceUpdateOptions{
				Version: version,
				Spec:    spec,
			}); err != nil {
				fmt.Printf("DeployWorker: error updating service: %v\n", err)
				w.Server.LogBrokerQ.EndLogs(&logbrokerqueue.EndLogData{
					DeploymentID: d.DeploymentID,
					Status:       types.DeploymentError,
					Message:      err.Error(),
				})
				continue
			}

			// end the logs
			w.Server.LogBrokerQ.EndLogs(&logbrokerqueue.EndLogData{
				DeploymentID: d.DeploymentID,
				Status:       types.DeploymentReady,
				Message:      "successfully re deployed",
			})

			fmt.Printf("DeployWorker: finished working ...")
		case <-ctx.Done():
			fmt.Println("DeployWorker: context cancelled, exiting")
			return
		}
	}
}
