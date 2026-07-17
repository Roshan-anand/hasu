package predef

import (
	"context"
	"fmt"

	"github.com/Roshan-anand/godploy/internal/lib/docker"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
)

type MountVolTarget string

const (
	PSQLMountTarget  MountVolTarget = "/var/lib/postgresql/data"
	RedisMountTarget MountVolTarget = "/data"
)

type PsqlVol string

const (
	PSQLVol  PsqlVol = "_pgdata"
	RedisVol PsqlVol = "_redisdata"
)

// creates a Docker swarm service for a predefined (PSQL/Redis) service.
//
// It creates the network if missing and deploys the service spec.
func DeployPredefService(
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

// constructs a PostgreSQL connection URL for internal use within the Docker network.
func BuildPsqlInternalURL(dbUser, dbPassword, serviceName, dbName string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:5432/%s?sslmode=disable", dbUser, dbPassword, serviceName, dbName)
}

// constructs a Redis connection URL for internal use within the Docker network.
func BuildRedisInternalURL(password, serviceName string) string {
	if password == "" {
		return fmt.Sprintf("redis://%s:6379", serviceName)
	}
	return fmt.Sprintf("redis://:%s@%s:6379", password, serviceName)
}

// creates a Docker volume for the predefined service (PSQL/Redis) to persist data.
func CreatePredefVolume(ctx context.Context, serviceName string, docker *client.Client, volType PsqlVol) (string, error) {
	volume, err := docker.VolumeCreate(ctx, volume.CreateOptions{
		Name:   serviceName + string(volType),
		Driver: "local",
	})
	if err != nil {
		return "", err
	}
	return volume.Name, nil
}
