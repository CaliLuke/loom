package testkit

import (
	"context"
	"sync"
	"testing"
	"time"

	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

type (
	// MetricHarness captures metrics using a manual reader.
	MetricHarness struct {
		Provider *metric.MeterProvider
		Reader   *metric.ManualReader
	}

	// HTTPMetricsRecorder records completed requests for tests.
	HTTPMetricsRecorder struct {
		mu      sync.Mutex
		Records []HTTPRequestRecord
	}

	// HTTPRequestRecord stores one completed custom HTTP metric observation.
	HTTPRequestRecord struct {
		Method     string
		Route      string
		StatusCode int
		Duration   time.Duration
	}
)

// NewMetricHarness creates a meter provider backed by a manual reader.
func NewMetricHarness(tb testing.TB) *MetricHarness {
	tb.Helper()
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	tb.Cleanup(func() {
		ctx, cancel := newShutdownContext()
		defer cancel()
		if err := provider.Shutdown(ctx); err != nil {
			tb.Errorf("shutdown meter provider: %v", err)
		}
	})
	return &MetricHarness{Provider: provider, Reader: reader}
}

// Collect reads the current metric set.
func (h *MetricHarness) Collect(ctx context.Context) metricdata.ResourceMetrics {
	var rm metricdata.ResourceMetrics
	if err := h.Reader.Collect(ctx, &rm); err != nil {
		panic(err)
	}
	return rm
}

// NewHTTPMetricsRecorder creates a custom HTTP metrics recorder for tests.
func NewHTTPMetricsRecorder() *HTTPMetricsRecorder {
	return &HTTPMetricsRecorder{}
}

// Record saves one completed HTTP request record.
func (r *HTTPMetricsRecorder) Record(_ context.Context, method string, route string, statusCode int, duration time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Records = append(r.Records, HTTPRequestRecord{
		Method:     method,
		Route:      route,
		StatusCode: statusCode,
		Duration:   duration,
	})
}

// Snapshot returns a copy of the recorded requests.
func (r *HTTPMetricsRecorder) Snapshot() []HTTPRequestRecord {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]HTTPRequestRecord, len(r.Records))
	copy(out, r.Records)
	return out
}
