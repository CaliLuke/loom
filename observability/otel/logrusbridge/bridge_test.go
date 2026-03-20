package logrusbridge

import (
	"testing"

	"go.opentelemetry.io/otel/attribute"
	otellog "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/trace"
	"goa.design/goa/v3/observability/otel/internal/testkit"

	"github.com/stretchr/testify/require"
)

func TestNewMirrorsLogrusEntriesIntoOpenTelemetryLogs(t *testing.T) {
	logHarness := testkit.NewLogHarness(t)
	traceHarness := testkit.NewTraceHarness(t)
	logger, err := New(Config{
		ServiceName:    "autok",
		LoggerProvider: logHarness.Provider,
	})
	require.NoError(t, err)

	ctx, span := traceHarness.Provider.Tracer("test").Start(t.Context(), "bridge")
	logger.WithContext(ctx).WithField("project_id", "proj_123").Info("hello")
	span.End()

	records := logHarness.Exporter.Records()
	require.Len(t, records, 1)
	require.Equal(t, "hello", records[0].Body().AsString())
	require.NotEqual(t, trace.TraceID{}, records[0].TraceID())
	requireLogAttribute(t, records[0], attribute.String("project_id", "proj_123"))
}

func requireLogAttribute(t *testing.T, record sdklog.Record, expected attribute.KeyValue) {
	t.Helper()
	found := false
	record.WalkAttributes(func(kv otellog.KeyValue) bool {
		if string(kv.Key) != string(expected.Key) {
			return true
		}
		require.Equal(t, expected.Value.AsString(), kv.Value.AsString())
		found = true
		return false
	})
	require.True(t, found, "missing log attribute %q", expected.Key)
}
