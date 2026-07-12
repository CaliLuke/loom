package otel

import (
	"net/http"
	"testing"

	"github.com/CaliLuke/loom/observability/otel/internal/testkit"
	"github.com/CaliLuke/loom/observability/otel/logrusbridge"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	otelglobal "go.opentelemetry.io/otel/log/global"

	"github.com/stretchr/testify/require"
)

func TestAutoKStyleCompatibilityContract(t *testing.T) {
	traceHarness := testkit.NewTraceHarness(t)
	metricHarness := testkit.NewMetricHarness(t)
	logHarness := testkit.NewLogHarness(t)
	lpBefore := otelglobal.GetLoggerProvider()
	t.Cleanup(func() {
		otelglobal.SetLoggerProvider(lpBefore)
	})
	otelglobal.SetLoggerProvider(logHarness.Provider)

	recorder := testkit.NewHTTPMetricsRecorder()
	logger, err := logrusbridge.New(logrusbridge.Config{
		ServiceName:    "autok",
		LoggerProvider: logHarness.Provider,
	})
	require.NoError(t, err)

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
			logger.WithField("route", r.Pattern).Info("request started")
			next.ServeHTTP(w, r)
		})
	})
	fixture.Mux.Handle("GET", "/projects/{project_id}", func(w http.ResponseWriter, r *http.Request) {
		AddHTTPAttributes(r.Context(), attribute.String("operation.kind", "request"))
		record := log.Record{}
		record.SetBody(log.StringValue("handler log"))
		otelglobal.Logger("autok.handler").Emit(r.Context(), record)
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
	require.Len(t, logHarness.Exporter.Records(), 2)
}
