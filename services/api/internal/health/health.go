package health

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"time"

	"github.com/nats-io/nats.go"
)

// HealthChecker manages the health check functionality
type HealthChecker struct {
	nc          *nats.Conn
	config      config
	startTime   time.Time
	stopCh      chan struct{}
	versionInfo *VersionInfo
	checks      []Checker
}

// config contains configuration for the health checker
type config struct {
	Interval    time.Duration
	Timeout     time.Duration
	Subject     string
	StatusTopic string
	ServiceName string
}

// Status represents the overall status of the service
type Status struct {
	Status      string
	SystemInfo  map[string]string
	Version     string
	Environment string
	Uptime      string
	Timestamp   time.Time
	Checks      map[string]string
	ServiceName string
}

// New creates a new health checker with the provided options
func New(nc *nats.Conn, versionInfo *VersionInfo, opts ...Option) *HealthChecker {
	if nc == nil {
		panic("NATS connection is required")
	}
	if versionInfo == nil {
		panic("Version info is required")
	}

	hc := &HealthChecker{
		nc: nc,
		config: config{
			Interval:    time.Second * 30,
			Timeout:     time.Second * 5,
			Subject:     "health",
			StatusTopic: "health.status",
			ServiceName: "service",
		},
		startTime:   time.Now(),
		stopCh:      make(chan struct{}),
		versionInfo: versionInfo,
		checks:      make([]Checker, 0),
	}

	// Apply all options
	for _, opt := range opts {
		opt(hc)
	}

	return hc
}

// Start begins the health check service
func (hc *HealthChecker) Start() error {
	// Subscribe to health check requests using service name
	subject := fmt.Sprintf("%s.%s", hc.config.Subject, hc.config.ServiceName)
	_, err := hc.nc.Subscribe(subject, func(msg *nats.Msg) {
		health := hc.GetHealth()
		response, _ := json.Marshal(health)
		msg.Respond(response)
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to health check: %v", err)
	}

	// Start periodic health status publishing
	go hc.publishHealthStatus()
	return nil
}

// publishHealthStatus periodically publishes the health status
func (hc *HealthChecker) publishHealthStatus() {
	ticker := time.NewTicker(hc.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			health := hc.GetHealth()
			healthData, _ := json.Marshal(health)
			topic := fmt.Sprintf("%s.%s", hc.config.StatusTopic, hc.config.ServiceName)
			hc.nc.Publish(topic, healthData)
		case <-hc.stopCh:
			return
		}
	}
}

// GetHealth returns the current health status
func (hc *HealthChecker) GetHealth() Status {
	status := Status{
		Status:      "healthy",
		SystemInfo:  getSystemInfo(),
		Version:     hc.versionInfo.GetVersionString(),
		Environment: hc.versionInfo.Environment,
		Uptime:      time.Since(hc.startTime).String(),
		Timestamp:   time.Now(),
		Checks:      make(map[string]string),
		ServiceName: hc.config.ServiceName, // Include service name in status
	}

	// Perform all health checks
	ctx, cancel := context.WithTimeout(context.Background(), hc.config.Timeout)
	defer cancel()

	for _, check := range hc.checks {
		if err := check.Check(ctx); err != nil {
			status.Status = "unhealthy"
			status.Checks[check.Name()] = fmt.Sprintf("unhealthy: %v", err)
		} else {
			status.Checks[check.Name()] = "healthy"
		}
	}

	return status
}

// getSystemInfo collects system-level information
func getSystemInfo() map[string]string {
	return map[string]string{
		"go_version":    runtime.Version(),
		"os":            runtime.GOOS,
		"arch":          runtime.GOARCH,
		"num_cpu":       fmt.Sprintf("%d", runtime.NumCPU()),
		"num_goroutine": fmt.Sprintf("%d", runtime.NumGoroutine()),
	}
}

func (hc *HealthChecker) Stop() {
	close(hc.stopCh)
}
