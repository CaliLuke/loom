package logrusbridge

import (
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/bridges/otellogrus"
	"go.opentelemetry.io/otel/log"
)

type (
	// Config configures a logrus logger bridged into OpenTelemetry logs.
	Config struct {
		// ServiceName is the instrumentation name attached to bridged logs.
		ServiceName string
		// LoggerProvider overrides the OpenTelemetry logger provider used by the
		// otellogrus bridge.
		LoggerProvider log.LoggerProvider
	}
)

// New returns a logrus logger that mirrors entries into OpenTelemetry logs
// using the official otellogrus bridge.
func New(cfg Config) (*logrus.Logger, error) {
	logger := logrus.New()
	opts := make([]otellogrus.Option, 0, 1)
	if cfg.LoggerProvider != nil {
		opts = append(opts, otellogrus.WithLoggerProvider(cfg.LoggerProvider))
	}
	logger.AddHook(otellogrus.NewHook(cfg.ServiceName, opts...))
	return logger, nil
}
