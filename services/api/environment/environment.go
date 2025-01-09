package environment

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/Autherain/go_cyber/internal/logger"
	"github.com/caarlos0/env/v8"
	"github.com/joho/godotenv"
	"github.com/nats-io/nats.go"
)

// Variables represents the environment variables used by the application.
type Variables struct {
	// NATS Configuration
	NATSURL string `env:"APP_NATS_URL" envDefault:"nats://localhost:4222"`

	// Service Configuration
	ServiceName          string        `env:"APP_SERVICE_NAME" envDefault:"myapp"`
	ServiceInChannelSize int           `env:"APP_SERVICE_IN_CHANNEL_SIZE" envDefault:"1024"`
	ServiceWorkerCount   int           `env:"APP_SERVICE_WORKER_COUNT" envDefault:"32"`
	ShutdownTimeout      time.Duration `env:"APP_SHUTDOWN_TIMEOUT" envDefault:"5s"`

	// Logger Configuration
	LogFormat logger.Format   `env:"APP_LOG_FORMAT" envDefault:"json"`
	LogLevel  logger.LogLevel `env:"APP_LOG_LEVEL" envDefault:"info"`
	LogSource bool            `env:"APP_LOG_SOURCE" envDefault:"false"`

	// Health Check Configuration
	HealthCheckEnabled     bool          `env:"APP_HEALTH_CHECK_ENABLED" envDefault:"true"`
	HealthCheckInterval    time.Duration `env:"APP_HEALTH_CHECK_INTERVAL" envDefault:"10s"`
	HealthCheckTimeout     time.Duration `env:"APP_HEALTH_CHECK_TIMEOUT" envDefault:"5s"`
	HealthCheckSubject     string        `env:"APP_HEALTH_CHECK_SUBJECT" envDefault:"health"`
	HealthCheckStatusTopic string        `env:"APP_HEALTH_CHECK_STATUS_TOPIC" envDefault:"health.status"`

	// Version Information
	Env string `env:"APP_ENV" envDefault:"development"`
}

// Parse environment variables.
func Parse() *Variables {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		panic(fmt.Errorf("error loading .env file: %w", err))
	}

	cfg := &Variables{}
	if err := env.Parse(cfg); err != nil {
		panic(fmt.Errorf("could not parse environment variables: %w", err))
	}

	return cfg
}

// MustInitNATSConn initializes a NATS connection with retry logic
func MustInitNATSConn(variables *Variables) *nats.Conn {
	opts := []nats.Option{
		nats.Name(variables.ServiceName),
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(5),
		nats.ReconnectWait(time.Second),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				slog.Error("NATS disconnected", "error", err)
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			slog.Info("NATS reconnected", "url", nc.ConnectedUrl())
		}),
	}

	conn, err := nats.Connect(variables.NATSURL, opts...)
	if err != nil {
		panic(fmt.Errorf("could not connect to NATS: %w", err))
	}

	return conn
}
