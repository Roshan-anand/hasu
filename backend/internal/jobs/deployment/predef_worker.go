package deployjob

import (
	"context"

	"github.com/Roshan-anand/godploy/internal/lib/docker"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/swarm"
)

type MountVolTarget string

const (
	PSQLMountTarget  MountVolTarget = "/var/lib/postgresql/data"
	RedisMountTarget MountVolTarget = "/data"
)

// DeployPredefinedService creates a Docker swarm service for a predefined (PSQL/Redis) service.
// It creates the network if missing and deploys the service spec.
func DeployPredefinedService(
	ctx context.Context,
	dockerClient *docker.DockerClient,
	network string,
	serviceName string,
	image string,
	env []string,
	volumeName string,
	mountTarget MountVolTarget,
) error {
	if err := dockerClient.CreateNetwork(network); err != nil {
		return err
	}

	spec := swarm.ServiceSpec{
		Annotations: swarm.Annotations{
			Name: serviceName,
		},
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: &swarm.ContainerSpec{
				Image: image,
				Env:   env,
				Mounts: []mount.Mount{
					{
						Type:   mount.TypeVolume,
						Source: volumeName,
						Target: string(mountTarget),
					},
				},
			},
			Networks: []swarm.NetworkAttachmentConfig{
				{
					Target: network,
				},
			},
			RestartPolicy: &swarm.RestartPolicy{
				Condition: swarm.RestartPolicyConditionAny,
			},
		},
	}

	_, err := dockerClient.Client.ServiceCreate(ctx, spec, swarm.ServiceCreateOptions{})
	return err
}
