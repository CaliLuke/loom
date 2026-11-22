package otel

import (
	"log/slog"
	"net/http"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/trace"

	"github.com/CaliLuke/loom/observability/otel/internal/testkit"
	"github.com/CaliLuke/loom/observability/otel/slogbridge"

	"github.com/stretchr/testify/require"
)

func TestAutoKStyleCompatibilityContract(t *testing.T) {
	traceHarness := testkit.NewTraceHarness(t)
	metricHarness := testkit.NewMetricHarness(t)
	logHarness := testkit.NewLogHarness(t)
	lpBefore := otel.GetLoggerProvider()
	t.Cleanup(func() {
		otel.SetLoggerProvider(lpBefore)
	})
	otel.SetLoggerProvider(logHarness.Provider)

	recorder := testkit.NewHTTPMetricsRecorder()
	logger := slog.New(slogbridge.NewHandler(slogbridge.Config{
		ScopeName:      "autok",
		LoggerProvider: logHarness.Provider,
	}))

	fixture := testkit.NewHTTPFixture(t)
	fixture.Mux.Use(HTTPMiddleware(HTTPConfig{
		ServiceName:     "autok",
		MetricMode:      HTTPMetricModeCustomOnly,
		TracerProvider:  traceHarness.Provider,
		MeterProvider:   metricHarness.Provider,
		AttributeSource: httpAttributeSource{},
		MetricsRecorder: customRecorder{recorder: recorder},
	}))
	fixture.Mux.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			AddHTTPAttributes(r.Context(),
				attribute.String("project_id", "proj_789"),
				attribute.Bool("autok.authenticated", true),
			)
			logger.InfoContext(r.Context(), "request started", slog.String("route", r.Pattern))
			next.ServeHTTP(w, r)
		})
	})
	fixture.Mux.Handle("GET", "/projects/{project_id}", func(w http.ResponseWriter, r *http.Request) {
		AddHTTPAttributes(r.Context(), attribute.String("operation.kind", "request"))
		record := log.Record{}
		record.SetBody(attribute.StringValue("handler log"))
		otel.Logger("autok.handler").Emit(r.Context(), record)
		w.WriteHeader(http.StatusOK)
	})

	resp, err := fixture.Request(t.Context(), http.MethodGet, "/projects/proj_789", nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.NoError(t, resp.Body.Close())

	ended := traceHarness.Recorder.Ended()
	require.Len(t, ended, 1)
	requireContainsAttribute(t, ended[0].Attributes(), "project_id", "proj_789")
	requireContainsAttribute(t, ended[0].Attributes(), "autok.authenticated", true)
	requireContainsAttribute(t, ended[0].Attributes(), "operation.kind", "request")
	require.Len(t, recorder.Snapshot(), 1)

	logRecords := logHarness.Exporter.Records()
	require.Len(t, logRecords, 2)
	spanCtx := ended[0].SpanContext()
	require.Equal(t, "request started", logRecords[0].Body().AsString())
	require.Equal(t, spanCtx.TraceID(), logRecords[0].TraceID())
	require.Equal(t, spanCtx.SpanID(), logRecords[0].SpanID())
	require.NotEqual(t, trace.TraceID{}, logRecords[0].TraceID())
	var gotRoute string
	logRecords[0].WalkAttributes(func(kv attribute.KeyValue) bool {
		if kv.Key == "route" {
			gotRoute = kv.Value.AsString()
			return false
		}
		return true
	})
	require.Equal(t, "GET /projects/{project_id}", gotRoute)
}
