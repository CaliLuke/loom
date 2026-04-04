package testkit

import (
	"context"
	"sync"
	"testing"

	"go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

type (
	// LogHarness captures exported SDK log records for tests.
	LogHarness struct {
		Provider *sdklog.LoggerProvider
		Exporter *LogExporter
	}

	// LogExporter captures exported log records.
	LogExporter struct {
		mu      sync.Mutex
		records []sdklog.Record
	}
)

// NewLogHarness creates a logger provider backed by a synchronous test
// exporter.
func NewLogHarness(tb testing.TB) *LogHarness {
	tb.Helper()
	exporter := &LogExporter{}
	provider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewSimpleProcessor(exporter)),
	)
	tb.Cleanup(func() {
		if err := provider.Shutdown(tb.Context()); err != nil {
			tb.Errorf("shutdown log provider: %v", err)
		}
	})
	return &LogHarness{Provider: provider, Exporter: exporter}
}

// LoggerProvider returns the configured logger provider.
func (h *LogHarness) LoggerProvider() log.LoggerProvider {
	return h.Provider
}

// Export captures exported records.
func (e *LogExporter) Export(_ context.Context, records []sdklog.Record) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, record := range records {
		e.records = append(e.records, record.Clone())
	}
	return nil
}

// Shutdown satisfies the SDK exporter contract.
func (e *LogExporter) Shutdown(context.Context) error {
	return nil
}

// ForceFlush satisfies the SDK exporter contract.
func (e *LogExporter) ForceFlush(context.Context) error {
	return nil
}

// Records returns a copy of the captured records.
func (e *LogExporter) Records() []sdklog.Record {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]sdklog.Record, len(e.records))
	copy(out, e.records)
	return out
}
