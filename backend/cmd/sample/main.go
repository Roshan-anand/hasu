package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Roshan-anand/godploy/internal/config"
	"github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/network"
)

func main() {

	docker, err := config.InitDockerClient()
	if err != nil {
		log.Fatalf("Error initializing docker client: %v\n", err)
		return
	}

	// check if project network exist
	res, err := docker.Client.NetworkInspect(context.Background(), "red_network", network.InspectOptions{})
	if err != nil {
		// check if error is not found error
		if errdefs.IsNotFound(err) {
			fmt.Println("Network not found")
			return
		}
		fmt.Printf("DeployWorker: error inspecting network: %v\n", err)
		return
	}

	fmt.Printf("Network inspect result: %+v\n", res.Name)
}
