package docker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/build"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
)

type DockerClient struct {
	Client *client.Client
}

func InitDockerClient() (*DockerClient, error) {
	c, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}

	ctx, cancle := context.WithTimeout(context.Background(), time.Second*5)
	defer cancle()

	p, err := c.Ping(ctx)
	if err != nil {
		if closeErr := c.Close(); closeErr != nil {
			return nil, errors.Join(err, closeErr)
		}
		return nil, err
	}

	fmt.Println("connected docker :", p.APIVersion)

	// initialize swarm mode if not initialized
	if _, err = c.SwarmInspect(context.Background()); err != nil {
		if _, err := c.SwarmInit(context.Background(), swarm.InitRequest{
			AdvertiseAddr: "127.0.0.1",
			ListenAddr:    "0.0.0.0",
		}); err != nil {
			return nil, fmt.Errorf("failed to initialize swarm mode : %w", err)
		}
	}

	return &DockerClient{Client: c}, nil
}

func (d *DockerClient) CloseClient() error {
	fmt.Println("closing docker client connection")
	return d.Client.Close()
}

// helper function to remove multiple image
func (d *DockerClient) RemoveImages(imgs []string) error {
	var error error

	for _, img := range imgs {
		_, err := d.Client.ImageRemove(context.Background(), img, image.RemoveOptions{
			Force:         true,
			PruneChildren: true,
		})
		if err != nil {
			error = err
		}
	}

	// remove all build cache
	if _, err := d.Client.BuildCachePrune(context.Background(), build.CachePruneOptions{
		All: true,
	}); err != nil {
		error = err
	}

	return error
}

// helper function to remove multiple services
func (d *DockerClient) RemoveServices(SwarmService map[string]struct{}) error {
	var error error
	for s := range SwarmService {
		if err := d.Client.ServiceRemove(context.Background(), s); err != nil {
			error = err
		}
	}
	return error
}

// helper function to create network if not exist
func (d *DockerClient) CreateNetwork(networkName string) error {
	_, err := d.Client.NetworkInspect(context.Background(), networkName, network.InspectOptions{})
	if err != nil {
		if errdefs.IsNotFound(err) {
			// create network if not exist
			_, err := d.Client.NetworkCreate(context.Background(), networkName, network.CreateOptions{
				Driver:     "overlay",
				Scope:      "swarm",
				Attachable: true,
			})
			if err != nil {
				fmt.Printf("error creating network: %v\n", err)
				return err
			}
		} else {
			fmt.Printf("error inspecting network: %v\n", err)
			return err
		}
	}
	return nil
}

// helper function to remove networks
func (d *DockerClient) RemoveNetworks(networks []string) error {
	var error error
	for _, n := range networks {
		if err := d.Client.NetworkRemove(context.Background(), n); err != nil {
			error = err
		}
	}
	return error
}

// helper function to remove a Docker volumes
func (d *DockerClient) RemoveVolumes(volumeName []string) error {
	var error error
	for _, v := range volumeName {
		if err := d.Client.VolumeRemove(context.Background(), v, true); err != nil {
			error = err
		}
	}
	return error
}
