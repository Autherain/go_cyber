package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Autherain/go_cyber/server"

	"github.com/Autherain/go_cyber/environment"
	"github.com/Autherain/go_cyber/internal/health"
	"github.com/Autherain/go_cyber/internal/logger"
)

const (
	ServiceName = "api"
)

func main() {
	// Parse environment variables
	variables := environment.Parse()

	// Initialize logger
	logger := logger.NewLogger(logger.Config{
		Format:    variables.LogFormat,
		Level:     variables.LogLevel,
		AddSource: variables.LogSource,
	})
	slog.SetDefault(logger.SlogLogger())

	// Initialize NATS connection
	natsConn := environment.MustInitNATSConn(variables)
	defer natsConn.Close()

	// Initialize version info for health checker
	versionInfo := health.NewVersionInfo(
		variables.Env,
	)

	// Initialize health checker
	healthChecker := health.New(
		natsConn,
		versionInfo,
		health.WithNATSCheck(natsConn),
		health.WithInterval(variables.HealthCheckInterval),
		health.WithTimeout(variables.HealthCheckTimeout),
		health.WithSubject(variables.HealthCheckSubject),
		health.WithServiceName(variables.ServiceName),
	)

	// Create server configuration
	config := &server.Config{
		ServiceName:          ServiceName,
		ServiceInChannelSize: variables.ServiceInChannelSize,
		ServiceWorkerCount:   variables.ServiceWorkerCount,
		ShutdownTimeout:      variables.ShutdownTimeout,
		HealthCheckEnabled:   variables.HealthCheckEnabled,
	}

	// Create and configure server
	srv := server.New(
		config,
		server.WithLogger(logger),
		server.WithNATSConnection(natsConn),
		server.WithHealthChecker(healthChecker),
	)

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		logger.Info("Received shutdown signal", "signal", sig)
		cancel()
	}()

	// Start server
	if err := srv.Start(ctx); err != nil {
		logger.Error("Server error", "error", err)
		os.Exit(1)
	}
}
