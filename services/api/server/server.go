package server

import (
	"context"
	"sync"
	"time"

	"github.com/Autherain/go_cyber/internal/health"
	"github.com/Autherain/go_cyber/internal/logger"
	"github.com/jirenius/go-res"
	"github.com/nats-io/nats.go"
)

type Server struct {
	service       *res.Service
	natsConn      *nats.Conn
	log           *logger.Logger
	wg            sync.WaitGroup
	config        *Config
	healthChecker *health.HealthChecker
}

type Config struct {
	ServiceName          string
	ServiceInChannelSize int
	ServiceWorkerCount   int
	ShutdownTimeout      time.Duration
	HealthCheckEnabled   bool
}

func New(config *Config, options ...Option) *Server {
	server := &Server{
		config: config,
	}

	// Apply options before creating the service
	for _, option := range options {
		option(server)
	}

	// Ensure there's a default logger if none was set
	if server.log == nil {
		server.log = logger.NewDefault()
	}

	// Create service with the configured logger
	server.service = res.NewService(config.ServiceName).
		SetInChannelSize(config.ServiceInChannelSize).
		SetLogger(server.log).
		SetWorkerCount(config.ServiceWorkerCount)

	return server
}

type Option func(*Server)

func WithLogger(logger *logger.Logger) Option {
	return func(s *Server) {
		s.log = logger
	}
}

func WithNATSConnection(conn *nats.Conn) Option {
	return func(s *Server) {
		s.natsConn = conn
	}
}

func WithHealthChecker(healthChecker *health.HealthChecker) Option {
	return func(s *Server) {
		s.healthChecker = healthChecker
	}
}

func (s *Server) Start(ctx context.Context) error {
	s.log.Info("Starting application",
		"version", s.healthChecker.GetHealth().Version,
		"environment", s.healthChecker.GetHealth().Environment,
	)

	errChan := make(chan error, 1)

	// Start health checker only if enabled
	if s.config.HealthCheckEnabled {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			if err := s.healthChecker.Start(); err != nil {
				s.log.Error("Health checker error", "error", err)
				errChan <- err
			}
		}()
	}

	// Start service
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := s.service.Serve(s.natsConn); err != nil {
			s.log.Error("Service error", "error", err)
			errChan <- err
		}
	}()

	// Define resources
	s.defineResources()

	return s.handleShutdown(ctx, errChan)
}

func (s *Server) handleShutdown(ctx context.Context, errChan chan error) error {
	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		s.log.Info("Context cancelled")
		return s.shutdown()
	}
}

func (s *Server) shutdown() error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
	defer cancel()

	// Stop health checker
	s.healthChecker.Stop()

	if err := s.natsConn.Drain(); err != nil {
		s.log.Error("Error draining NATS connection", "error", err)
	}

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.log.Info("Service stopped gracefully")
		return nil
	case <-shutdownCtx.Done():
		s.log.Error("Service shutdown timed out")
		return shutdownCtx.Err()
	}
}

func (s *Server) defineResources() {
	s.service.Handle(
		"myresource.>",
		res.Access(res.AccessGranted),
		res.GetResource(func(r res.GetRequest) {
			r.Model(map[string]interface{}{
				"message": "Hello from RES!",
				"time":    time.Now().Format(time.RFC3339),
			})
		}),
	)
}
