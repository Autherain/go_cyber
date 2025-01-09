package health

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

// Option defines a function type for configuring the HealthChecker
type Option func(*HealthChecker)

// Checker interface defines the contract for health checks
type Checker interface {
	Check(ctx context.Context) error
	Name() string
}

// WithNATSCheck adds NATS connection health check
func WithNATSCheck(conn *nats.Conn) Option {
	return func(hc *HealthChecker) {
		checker := &natsChecker{conn: conn}
		hc.checks = append(hc.checks, checker)
	}
}

// WithSQLCheck adds SQL database health check
func WithSQLCheck(db *sql.DB) Option {
	return func(hc *HealthChecker) {
		checker := &sqlChecker{db: db}
		hc.checks = append(hc.checks, checker)
	}
}

// WithInterval sets the health check interval
func WithInterval(interval time.Duration) Option {
	return func(hc *HealthChecker) {
		hc.config.Interval = interval
	}
}

// WithTimeout sets the health check timeout
func WithTimeout(timeout time.Duration) Option {
	return func(hc *HealthChecker) {
		hc.config.Timeout = timeout
	}
}

// WithSubject sets the NATS subject for health check requests
func WithSubject(subject string) Option {
	return func(hc *HealthChecker) {
		hc.config.Subject = subject
	}
}

// WithServiceName sets the service name
func WithServiceName(name string) Option {
	return func(hc *HealthChecker) {
		hc.config.ServiceName = name
	}
}

// Internal checker implementations
type natsChecker struct {
	conn *nats.Conn
}

func (n *natsChecker) Check(ctx context.Context) error {
	if !n.conn.IsConnected() {
		return fmt.Errorf("NATS connection is not alive")
	}
	return nil
}

func (n *natsChecker) Name() string {
	return "nats"
}

var _ Checker = (*natsChecker)(nil)

type sqlChecker struct {
	db *sql.DB
}

func (s *sqlChecker) Check(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *sqlChecker) Name() string {
	return "database"
}

var _ Checker = (*sqlChecker)(nil)
