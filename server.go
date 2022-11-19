package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type WebServer struct {
	log   *zap.Logger
	store DatabaseStore
	http.Server
}

const genericWebServerTimeout = 15 * time.Second

// startServer prepares and executes web-server control loop
func startServer() int {

	// init database
	db, closeDB, err := NewDatabaseStore()
	if err != nil {
		fmt.Println("Error: unable to initialize database:", err)
		return 1
	}
	defer closeDB()

	// init web-server struct
	server, err := NewWebServer(db)
	if err != nil {
		fmt.Println("Error: unable to initialize web-server:", err)
		return 1
	}
	defer server.log.Sync()

	// catch graceful termination signals from system
	var errGroup errgroup.Group
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	// stop server on graceful termination
	errGroup.Go(func() error {
		<-ctx.Done()
		if err := server.Stop(); err != nil {
			server.log.Info("Error stopping the server", zap.Error(err))
			return err
		}
		return nil
	})

	// run server
	if err := server.Start(); err != nil {
		server.log.Info("Error starting the server", zap.Error(err))
	}

	// wait for graceful termination to complete
	if err := errGroup.Wait(); err != nil {
		return 1
	}
	return 0
}

// Start starts web-server
func (srv *WebServer) Start() error {
	srv.log.Info("Starting the server", zap.String("address", srv.Addr))

	// start to listen for inbound connections
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("error starting the server: %w", err)
	}
	return nil
}

// Stop stops web-server
func (srv *WebServer) Stop() error {
	srv.log.Info("Stopping the server")

	// we have 20 seconds to drain connections
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// initiate server shutdown with timeout
	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("error stopping the server: %w", err)
	}
	return nil
}

// NewWebServer initialize web-server struct
func NewWebServer(store DatabaseStore) (*WebServer, error) {

	// fetch variables from environment variables
	port, err := getIntOrFail("SERVE_PORT")
	if err != nil {
		return nil, err
	}

	host := getStringOrDefault("SERVE_HOST", "localhost")
	addr := net.JoinHostPort(host, strconv.Itoa(port))

	// init logger
	log, err := createLogger()
	if err != nil {
		return nil, err
	}

	// prepend log messages with release version tag
	if len(tagRelease) == 0 {
		tagRelease = "dev"
	}
	log = log.With(zap.String("release", tagRelease))

	// init and return webserver struct
	server := &WebServer{
		log: log,
		Server: http.Server{
			Addr:              addr,
			ReadTimeout:       genericWebServerTimeout,
			ReadHeaderTimeout: genericWebServerTimeout,
			WriteTimeout:      genericWebServerTimeout,
			IdleTimeout:       genericWebServerTimeout,
			ErrorLog:          zap.NewStdLog(log),
		},
		store: store,
	}

	// register routes
	server.initRoutes()

	return server, nil
}

// createLogger prepares stdout logging facility based on release type
func createLogger() (*zap.Logger, error) {

	// fetch logging facility class from environment variable
	logEnv := getStringOrDefault("SERVE_LOG_ENV", "development")

	// pick logging facility class
	switch logEnv {
	case "production":
		return zap.NewProduction()
	case "development":
		return zap.NewDevelopment()
	default:
		return zap.NewNop(), nil
	}
}
