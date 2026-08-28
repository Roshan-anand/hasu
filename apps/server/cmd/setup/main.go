package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/Roshan-anand/hasu/internal/lib/docker"
)

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// to setup local development env
func main() {
	docker, err := docker.InitDockerClient()
	if err != nil {
		fmt.Println(err)
		return
	}

	host := docker.Client.DaemonHost()
	os.Setenv("DOCKER_HOST", host)

	switch os.Args[1] {
	case "setup":
		if err := runCommand("docker", "stack", "deploy", "-c", "../../docker/compose.traefik-dev.yml", "hasu"); err != nil {
			fmt.Println("failed to setup traefik stack :", err)
			return
		}
		if err := runCommand("docker", "compose", "-f", "../../docker/compose.dev.yml", "build"); err != nil {
			fmt.Println("failed to build hasu backend image :", err)
			return
		}
	case "dev-start":
		if err := runCommand("docker", "compose", "-p", "hasu", "-f", "../../docker/compose.dev.yml", "up", "--watch"); err != nil {
			fmt.Println("failed to setup hasu stack :", err)
			return
		}
	case "dev-stop":
		if err := runCommand("docker", "compose", "-p", "hasu", "-f", "../../docker/compose.dev.yml", "down"); err != nil {
			fmt.Println("failed to stop hasu stack :", err)
			return
		}
	case "test-backend":
		if err := runCommand("docker", "compose", "-f", "../../docker/compose.dev.yml", "run", "--rm", "server", "go", "test", "-v", "./..."); err != nil {
			fmt.Println("failed to stop hasu stack :", err)
			return
		}
	case "server-logs":
		if err := runCommand("docker", "compose", "-p", "hasu", "-f", "../../docker/compose.dev.yml", "logs", "-f", "server"); err != nil {
			fmt.Println("failed to fetch hasu backend logs :", err)
			return
		}
	case "web-logs":
		if err := runCommand("docker", "compose", "-p", "hasu", "-f", "../../docker/compose.dev.yml", "logs", "-f", "web"); err != nil {
			fmt.Println("failed to fetch hasu frontend logs :", err)
			return
		}
	case "traefik-logs":
		if err := runCommand("docker", "service", "logs", "-f", "hasu_traefik"); err != nil {
			fmt.Println("failed to fetch traefik logs :", err)
			return
		}
	default:
		fmt.Println("invalid command")
	}
}
