package config

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/moby/moby/client"
)

type DockerClient struct {
	Client *client.Client
}

func InitDockerClient() (*DockerClient, error) {
	c, err := client.New(client.FromEnv)
	if err != nil {
		return nil, err
	}

	ctx, cancle := context.WithTimeout(context.Background(), time.Second*5)
	defer cancle()

	p, err := c.Ping(ctx, client.PingOptions{})
	if err != nil {
		if closeErr := c.Close(); closeErr != nil {
			return nil, errors.Join(err, closeErr)
		}
		return nil, err
	}

	fmt.Println("connected docker :", p.APIVersion)

	// initialize swarm mode if not initialized
	if _, err = c.SwarmInspect(context.Background(), client.SwarmInspectOptions{}); err != nil {
		if _, err := c.SwarmInit(context.Background(), client.SwarmInitOptions{
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
func (d *DockerClient) RemoveImages(imgs []string) {
	fmt.Printf("removing images: %v\n", imgs)
	for _, img := range imgs {
		_, err := d.Client.ImageRemove(context.Background(), img, client.ImageRemoveOptions{
			Force:         true,
			PruneChildren: true,
		})
		if err != nil {
			fmt.Printf("failed to remove image %s : %v\n", img, err)
		} else {
			fmt.Printf("removed image %s\n", img)
		}
	}
}
