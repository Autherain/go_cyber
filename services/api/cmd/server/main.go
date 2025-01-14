package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Autherain/go_cyber/environment"
	"github.com/Autherain/go_cyber/internal/health"
	"github.com/Autherain/go_cyber/internal/logger"
	"github.com/Autherain/go_cyber/server"
	"github.com/jirenius/go-res"
)

const serviceName = "api"

func main() {
	// Parse environment variables
	variables := environment.Parse()

	// Initialize logger
	log := logger.NewLogger(logger.Config{
		Format:    variables.LogFormat,
		Level:     variables.LogLevel,
		AddSource: variables.LogSource,
	})
	slog.SetDefault(log.SlogLogger())

	// Initialize NATS connection
	natsConn := environment.MustInitNATSConn(variables)
	defer natsConn.Close()

	// Initialize service first
	service := res.NewService(serviceName)
	service.SetLogger(log)
	service.SetInChannelSize(variables.ServiceInChannelSize)
	service.SetWorkerCount(variables.ServiceWorkerCount)

	// Initialize health checker
	healthChecker := health.New(
		natsConn,
		health.NewVersionInfo(variables.Env),
		health.WithNATSCheck(natsConn),
		health.WithInterval(variables.HealthCheckInterval),
		health.WithTimeout(variables.HealthCheckTimeout),
		health.WithSubject(variables.HealthCheckSubject),
		health.WithServiceName(serviceName),
	)

	// Create server with all dependencies
	srv := server.New(
		server.WithService(service),
		server.WithLogger(log),
		server.WithHealthChecker(healthChecker),
		server.WithShutdownTimeout(variables.ShutdownTimeout),
	)

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		log.Info("Received shutdown signal", "signal", sig)
		cancel()
	}()

	// Start server
	if err := srv.Start(ctx, natsConn); err != nil {
		log.Error("Server error", "error", err)
		os.Exit(1)
	}
}
