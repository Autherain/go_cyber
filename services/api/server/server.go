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
	service         *res.Service
	log             *logger.Logger
	healthChecker   *health.HealthChecker
	wg              sync.WaitGroup
	shutdownTimeout time.Duration
}

type Option func(*Server)

// New creates a new server instance with the given options
func New(options ...Option) *Server {
	s := &Server{}

	for _, option := range options {
		option(s)
	}

	if s.service == nil {
		panic("server requires a RES service")
	}

	s.addResourceHandlers()

	if s.log == nil {
		s.log = logger.NewDefault()
	}

	return s
}

// WithService sets the RES service
func WithService(service *res.Service) Option {
	return func(s *Server) {
		s.service = service
	}
}

// WithLogger sets the logger
func WithLogger(logger *logger.Logger) Option {
	return func(s *Server) {
		s.log = logger
	}
}

// WithHealthChecker sets the health checker
func WithHealthChecker(healthChecker *health.HealthChecker) Option {
	return func(s *Server) {
		s.healthChecker = healthChecker
	}
}

// WithShutdownTimeout sets the shutdown timeout
func WithShutdownTimeout(timeout time.Duration) Option {
	return func(s *Server) {
		s.shutdownTimeout = timeout
	}
}

func (s *Server) Start(ctx context.Context, natsConn *nats.Conn) error {
	s.log.Info("Starting application")

	errChan := make(chan error, 1)

	// Start health checker if enabled
	if s.healthChecker != nil {
		s.startHealthChecker(errChan)
	}

	// Start service
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := s.service.Serve(natsConn); err != nil {
			s.log.Error("Service error", "error", err)
			errChan <- err
		}
	}()

	return s.handleShutdown(ctx, errChan)
}

func (s *Server) startHealthChecker(errChan chan error) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := s.healthChecker.Start(); err != nil {
			s.log.Error("Health checker error", "error", err)
			errChan <- err
		}
	}()
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
	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()

	if s.healthChecker != nil {
		s.healthChecker.Stop()
	}

	if err := s.service.Shutdown(); err != nil {
		s.log.Error("Error stopping RES service", "error", err)
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

func (s *Server) addResourceHandlers() {
	// Add your resource handlers here
	s.service.Handle(
		"api.>",
		s.handleAPIRequest(),
	)
}

func (s *Server) handleAPIRequest() res.Option {
	return res.GetModel(func(r res.ModelRequest) {
		r.Model(map[string]interface{}{
			"message": "Hello from API",
			"time":    time.Now().Format(time.RFC3339),
		})
	})
}
