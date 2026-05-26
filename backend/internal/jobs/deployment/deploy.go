package deploymentjob

import (
	"context"
	"fmt"

	deploymentqueue "github.com/Roshan-anand/godploy/internal/jobs/deployment/queue"
	logbrokerqueue "github.com/Roshan-anand/godploy/internal/jobs/logbroker/queue"
	"github.com/Roshan-anand/godploy/internal/lib/types"
	"github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/network"
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
				Msg: getTitle("Deploying  the service " + d.SwarmServiceName),
			})

			// create network if not exist
			_, err := docker.NetworkInspect(context.Background(), d.NetworkName, network.InspectOptions{})
			if err != nil {
				if errdefs.IsNotFound(err) {
					// create network if not exist
					_, err := docker.NetworkCreate(context.Background(), d.NetworkName, network.CreateOptions{
						Driver:     "overlay",
						Scope:      "swarm",
						Attachable: true,
					})
					if err != nil {
						fmt.Printf("DeployWorker: error creating network: %v\n", err)
						l.EndLogs(&logbrokerqueue.EndLogData{
							DeploymentID: d.DeploymentID,
							Status:       types.DeploymentError,
							Message:      err.Error(),
						})
						continue
					}
				} else {
					fmt.Printf("DeployWorker: error inspecting network: %v\n", err)
					l.EndLogs(&logbrokerqueue.EndLogData{
						DeploymentID: d.DeploymentID,
						Status:       types.DeploymentError,
						Message:      err.Error(),
					})
					continue
				}
			}

			replicas := uint64(1)
			// config swarm service spec
			spec := swarm.ServiceSpec{
				Annotations: swarm.Annotations{
					Name: d.SwarmServiceName,
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
							Target: d.NetworkName,
						},
					},
				},

				Mode: swarm.ServiceMode{
					Replicated: &swarm.ReplicatedService{
						Replicas: &replicas,
					},
				},
			}

			// if env avalable
			if len(d.Env) > 0 {
				spec.TaskTemplate.ContainerSpec.Env = d.Env
			}

			// if the service is public connect to traefik
			if d.IsPublic {
				spec.TaskTemplate.Networks = append(spec.TaskTemplate.Networks, swarm.NetworkAttachmentConfig{
					Target: "godploy_traefik_proxy",
				})

				spec.Annotations.Labels = map[string]string{
					"traefik.enable": "true",
					fmt.Sprintf("traefik.http.routers.%s.entrypoints", d.SwarmServiceName):               "websecure",
					fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port", d.SwarmServiceName): "80",
					fmt.Sprintf("traefik.http.routers.%s.tls.certresolver", d.SwarmServiceName):          "le",
					"traefik.constraint-label": "head-proxy",
				}
			}

			_, err = docker.ServiceCreate(context.Background(), spec, swarm.ServiceCreateOptions{})
			if err != nil {
				fmt.Printf("DeployWorker: error creating service: %v\n", err)
				l.EndLogs(&logbrokerqueue.EndLogData{
					DeploymentID: d.DeploymentID,
					Status:       types.DeploymentError,
					Message:      err.Error(),
				})
				continue
			}

			fmt.Println("finished deploying :", d.SwarmServiceName)
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
