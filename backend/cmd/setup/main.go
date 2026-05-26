package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/Roshan-anand/godploy/internal/config"
)

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// to setup local development env
func main() {
	docker, err := config.InitDockerClient()
	if err != nil {
		fmt.Println(err)
		return
	}

	host := docker.Client.DaemonHost()
	os.Setenv("DOCKER_HOST", host)

	switch os.Args[1] {
	case "setup":
		if err := runCommand("docker", "stack", "deploy", "-c", "../docker/compose.traefik-dev.yaml", "godploy"); err != nil {
			fmt.Println("failed to setup traefik stack :", err)
			return
		}
		if err := runCommand("docker", "compose", "-f", "../docker/compose.dev.yaml", "build"); err != nil {
			fmt.Println("failed to build godploy backend image :", err)
			return
		}
	case "dev-start":
		if err := runCommand("docker", "compose", "-p", "godploy", "-f", "../docker/compose.dev.yaml", "up", "--watch"); err != nil {
			fmt.Println("failed to setup godploy stack :", err)
			return
		}
	case "server-start":
		if err := runCommand("docker", "compose", "-p", "godploy", "-f", "../docker/compose.dev.yaml", "up", "server", "--watch"); err != nil {
			fmt.Println("failed to setup godploy stack :", err)
			return
		}
	case "dev-stop":
		if err := runCommand("docker", "compose", "-p", "godploy", "-f", "../docker/compose.dev.yaml", "down"); err != nil {
			fmt.Println("failed to stop godploy stack :", err)
			return
		}
	case "test-backend":
		if err := runCommand("docker", "compose", "-f", "../docker/compose.dev.yaml", "run", "--rm", "server", "go", "test", "-v", "./..."); err != nil {
			fmt.Println("failed to stop godploy stack :", err)
			return
		}
	case "server-logs":
		if err := runCommand("docker", "compose", "-p", "godploy", "-f", "../docker/compose.dev.yaml", "logs", "-f", "server"); err != nil {
			fmt.Println("failed to fetch godploy backend logs :", err)
			return
		}
	case "web-logs":
		if err := runCommand("docker", "compose", "-p", "godploy", "-f", "../docker/compose.dev.yaml", "logs", "-f", "web"); err != nil {
			fmt.Println("failed to fetch godploy frontend logs :", err)
			return
		}
	case "traefik-logs":
		if err := runCommand("docker", "service", "logs", "-f", "godploy_traefik"); err != nil {
			fmt.Println("failed to fetch traefik logs :", err)
			return
		}
	default:
		fmt.Println("invalid command")
	}
}
