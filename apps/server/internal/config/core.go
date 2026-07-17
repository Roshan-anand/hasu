package config

import (
	"net/http"

	"github.com/Roshan-anand/godploy/internal/lib/database"
	"github.com/Roshan-anand/godploy/internal/lib/docker"
	services "github.com/Roshan-anand/godploy/internal/service"
)

// server holds the global configuration for the application
type Server struct {
	Http     *http.Server
	DB       *database.DataBase
	BadgerDB *database.BadgerDB
	Config   *Config
	Docker   *docker.DockerClient
	Services *services.Services
}

// creates a new server instance
func NewServer(cfg *Config) (*Server, error) {
	// connect DB, Redis, Docker client etc. here and add them to the server struct

	// initialize database connection
	db, err := database.InitDb(cfg.SqliteDir)
	if err != nil {
		return nil, err
	}

	// initialize badgerDB connection
	badger, err := database.InitBadgerDB(cfg.BadgerDir)
	if err != nil {
		return nil, err
	}

	//initialize docker client
	docker, err := docker.InitDockerClient()
	if err != nil {
		return nil, err
	}

	services := services.NewServices(db, docker, badger)

	return &Server{
		DB:       db,
		BadgerDB: badger,
		Config:   cfg,
		Docker:   docker,
		Services: services,
	}, nil
}
