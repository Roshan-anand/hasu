package config

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

const (
	SHUTDOWN_TIMEOUT = 30
)

// setups new http server with given handler
//
// @param h : http handler  to set the server with
func (s *Server) SetupHttp(h http.Handler) {
	s.Http = &http.Server{
		Addr:         ":" + s.Config.Port,
		Handler:      h,
		WriteTimeout: 0,
	}
}

// starts the http server
//
// @param srvErr : channel to send server errors
func (s *Server) StartServer(srvErr chan error) {
	// check if server is properly configured
	errMsg := "server config error: "
	switch {
	case s.Http == nil:
		srvErr <- fmt.Errorf("%s : http server is not initialized", errMsg)

	case s.DB == nil:
		srvErr <- fmt.Errorf("%s : database is not initialized", errMsg)
	}

	// TODO : use logger to log the info

	fmt.Println("starting server on port ", s.Http.Addr) // TODO : change it to env (konf)
	if err := s.Http.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		srvErr <- fmt.Errorf("listen serve error : %w", err)
	}
}

// shuts down the http server gracefully
func (s *Server) ShutDownServer() error {
	ctx, stop := context.WithTimeout(context.Background(), SHUTDOWN_TIMEOUT*time.Second)
	defer stop()

	// close http connections and shutdown server
	if err := s.Http.Shutdown(ctx); err != nil {
		// force close the server
		if closeErr := s.Http.Close(); closeErr != nil {
			return errors.Join(fmt.Errorf("http server close error :"), err, closeErr)
		}

		return fmt.Errorf("http server shutdown error : %w", err)
	}

	// close deployment workers
	if err := s.Services.Deployment.Stop(context.Background()); err != nil {
		return err
	}

	// close log broker workers
	if err := s.Services.LogBroker.Stop(context.Background()); err != nil {
		return err
	}

	// close database connections
	if err := s.DB.CloseDb(); err != nil {
		return err
	}

	// close badgerDB connection
	if err := s.BadgerDB.CloseDb(); err != nil {
		return err
	}

	// close docker client connection
	if err := s.Docker.CloseClient(); err != nil {
		return err
	}

	return nil
}
