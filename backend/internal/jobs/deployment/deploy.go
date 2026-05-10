package deploymentjob

import (
	"context"
	"fmt"

	deploymentqueue "github.com/Roshan-anand/godploy/internal/jobs/deployment/queue"
	logbrokerqueue "github.com/Roshan-anand/godploy/internal/jobs/logbroker/queue"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
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

			fmt.Println("DeployWorker: started working ...")

			replicas := uint64(1)
			_, err := w.Server.Docker.Client.ServiceCreate(context.Background(), client.ServiceCreateOptions{
				Spec: swarm.ServiceSpec{

					Annotations: swarm.Annotations{
						Name: d.SwarmServiceName,
						Labels: map[string]string{
							"traefik.enable": "true",
							fmt.Sprintf("traefik.http.routers.%s.rule", d.SwarmServiceName):                      "Host(`portfolio.godploy.localhost`)",
							fmt.Sprintf("traefik.http.routers.%s.entrypoints", d.SwarmServiceName):               "websecure",
							fmt.Sprintf("traefik.http.routers.%s.tls", d.SwarmServiceName):                       "true",
							fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port", d.SwarmServiceName): "80",
							"traefik.constraint-label": "head-proxy",
						},
					},

					TaskTemplate: swarm.TaskSpec{
						ContainerSpec: &swarm.ContainerSpec{
							Image: d.ImgName,
							TTY:   true,
						},

						RestartPolicy: &swarm.RestartPolicy{
							Condition: swarm.RestartPolicyConditionAny,
						},

						Networks: []swarm.NetworkAttachmentConfig{
							{
								Target: "godploy_traefik_proxy",
							},
						},
					},

					Mode: swarm.ServiceMode{
						Replicated: &swarm.ReplicatedService{
							Replicas: &replicas,
						},
					},
				},
			})
			if err != nil {
				fmt.Printf("DeployWorker: error creating service: %v\n", err)
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
				Message:      "successfully deployed",
			})

			fmt.Printf("DeployWorker: finished working ...")
		case <-ctx.Done():
			fmt.Println("DeployWorker: context cancelled, exiting")
			return
		}
	}
}
