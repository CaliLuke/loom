package otel

import (
	goagrpcotel "github.com/CaliLuke/loom/v3/grpc/middleware/otel"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

type (
	// GRPCConfig configures gRPC OpenTelemetry instrumentation.
	GRPCConfig struct {
		// TracerProvider overrides the trace provider used by otelgrpc.
		TracerProvider trace.TracerProvider
		// MeterProvider overrides the meter provider used by otelgrpc.
		MeterProvider metric.MeterProvider
		// Propagators overrides the propagators used by otelgrpc.
		Propagators propagation.TextMapPropagator
	}
)

// GRPCServerOption returns a gRPC server option that installs OpenTelemetry
// stats handling using the official otelgrpc implementation.
func GRPCServerOption(cfg GRPCConfig) grpc.ServerOption {
	return goagrpcotel.ServerOption(makeGRPCOptions(cfg)...)
}

// GRPCClientOption returns a gRPC dial option that installs OpenTelemetry stats
// handling using the official otelgrpc implementation.
func GRPCClientOption(cfg GRPCConfig) grpc.DialOption {
	return goagrpcotel.ClientOption(makeGRPCOptions(cfg)...)
}

func makeGRPCOptions(cfg GRPCConfig) []goagrpcotel.Option {
	opts := make([]goagrpcotel.Option, 0, 3)
	if cfg.TracerProvider != nil {
		opts = append(opts, otelgrpc.WithTracerProvider(cfg.TracerProvider))
	}
	if cfg.MeterProvider != nil {
		opts = append(opts, otelgrpc.WithMeterProvider(cfg.MeterProvider))
	}
	opts = append(opts, otelgrpc.WithPropagators(defaultPropagators(cfg.Propagators)))
	return opts
}
