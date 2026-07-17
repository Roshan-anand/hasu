package main

import (
	"context"
	"fmt"
	"log"
	"os/signal"
	"syscall"

	"github.com/Roshan-anand/godploy/internal/config"
	"github.com/Roshan-anand/godploy/internal/routes"
	"github.com/joho/godotenv"
)

// create and configure the server
func createServer() (*config.Server, error) {

	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// validate public ip
	// if cfg.AppEnv == types.ProdMode {
	// 	cfg.ServerUrl = utils.GetPublicUrl()
	// } else if !utils.ValidatePublicUrl(cfg.ServerUrl) {
	// 	return nil, fmt.Errorf("invalid server url: %s", cfg.ServerUrl)
	// }

	// create server instance
	s, err := config.NewServer(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize server: %w", err)
	}

	// setup all routes
	r, err := routes.SetupRoutes(s)
	if err != nil {
		return nil, fmt.Errorf("failed to setup routes: %w", err)
	}

	// start deployment workers
	s.Services.Deployment.Start(context.Background(), cfg.CodeStoreDir)

	// start log broker workers
	s.Services.LogBroker.Start(context.Background())

	// TODO : before server exits store the info of the pending works from the chan
	// clean up deployments that were interrupted by a previous server stop before
	// any new deployment work can begin.
	if err := s.Services.Deployment.CleanupInterruptedDeployments(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to cleanup interrupted deployments: %w", err)
	}

	s.SetupHttp(r) // setup http server with routes

	return s, nil
}

// starts the server
//
// listens for terminate or interrupt signals to shutdown the server gracefully
func runServer(server *config.Server) error {

	// context to listen for terminate or interrupt signals
	notify, cancle := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancle()

	srvErr := make(chan error, 1)
	defer close(srvErr)

	go server.StartServer(srvErr) // start the server

	// graceful shutdown on terminate or interrupt signal
	select {
	case <-notify.Done():
		if err := server.ShutDownServer(); err != nil {
			return err
		}
	case err := <-srvErr:
		return err
	}

	return nil
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, loading environment variables from system")
	}

	server, err := createServer()
	if err != nil {
		log.Fatal("failed to create server config: ", err)
		return
	}

	if err := runServer(server); err != nil {
		log.Fatal("failed to run server: ", err)
		return
	}
}
