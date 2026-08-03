package otel

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CaliLuke/loom/observability/otel/internal/testkit"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"

	"github.com/stretchr/testify/require"
)

type (
	httpAttributeSource struct{}
	customRecorder      struct {
		recorder *testkit.HTTPMetricsRecorder
	}
)

func TestHTTPMiddlewareUsesRoutePatternAndEnrichment(t *testing.T) {
	traceHarness := testkit.NewTraceHarness(t)
	metricHarness := testkit.NewMetricHarness(t)
	recorder := testkit.NewHTTPMetricsRecorder()
	fixture := testkit.NewHTTPFixture(t)
	fixture.Mux.Use(HTTPMiddleware(HTTPConfig{
		ServiceName:     "autok",
		MetricMode:      HTTPMetricModeBoth,
		TracerProvider:  traceHarness.Provider,
		MeterProvider:   metricHarness.Provider,
		AttributeSource: httpAttributeSource{},
		MetricsRecorder: customRecorder{recorder: recorder},
	}))
	fixture.Mux.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.True(t, trace.SpanFromContext(r.Context()).SpanContext().IsValid())
			AddHTTPAttributes(r.Context(),
				attribute.String("project_id", "proj_123"),
				attribute.Bool("autok.authenticated", true),
			)
			next.ServeHTTP(w, r)
		})
	})
	fixture.Mux.Handle("GET", "/projects/{project_id}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, err := w.Write([]byte("ok"))
		require.NoError(t, err)
	})

	resp, err := fixture.Request(t.Context(), http.MethodGet, "/projects/proj_123", nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusAccepted, resp.StatusCode)
	require.NoError(t, resp.Body.Close())

	ended := traceHarness.Recorder.Ended()
	require.Len(t, ended, 1)
	require.Equal(t, "GET /projects/{project_id}", ended[0].Name())
	requireContainsAttribute(t, ended[0].Attributes(), "project_id", "proj_123")
	requireContainsAttribute(t, ended[0].Attributes(), "autok.authenticated", true)

	records := recorder.Snapshot()
	require.Len(t, records, 1)
	require.Equal(t, "GET", records[0].Method)
	require.Equal(t, "GET /projects/{project_id}", records[0].Route)
	require.Equal(t, http.StatusAccepted, records[0].StatusCode)
}

func TestHTTPMiddlewareMetricModes(t *testing.T) {
	cases := []struct {
		name             string
		mode             HTTPMetricMode
		wantCustomRecord bool
		wantOTelMetrics  bool
	}{
		{name: "otel-only", mode: HTTPMetricModeOTelOnly, wantOTelMetrics: true},
		{name: "custom-only", mode: HTTPMetricModeCustomOnly, wantCustomRecord: true},
		{name: "both", mode: HTTPMetricModeBoth, wantCustomRecord: true, wantOTelMetrics: true},
		{name: "none", mode: HTTPMetricModeNone},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			traceHarness := testkit.NewTraceHarness(t)
			metricHarness := testkit.NewMetricHarness(t)
			recorder := testkit.NewHTTPMetricsRecorder()
			fixture := testkit.NewHTTPFixture(t)
			fixture.Mux.Use(HTTPMiddleware(HTTPConfig{
				ServiceName:     "autok",
				MetricMode:      tc.mode,
				TracerProvider:  traceHarness.Provider,
				MeterProvider:   metricHarness.Provider,
				MetricsRecorder: customRecorder{recorder: recorder},
			}))
			fixture.Mux.Handle("GET", "/health", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			})

			resp, err := fixture.Request(t.Context(), http.MethodGet, "/health", nil)
			require.NoError(t, err)
			require.Equal(t, http.StatusNoContent, resp.StatusCode)
			require.NoError(t, resp.Body.Close())

			if tc.wantCustomRecord {
				require.Len(t, recorder.Snapshot(), 1)
			} else {
				require.Empty(t, recorder.Snapshot())
			}

			rm := metricHarness.Collect(t.Context())
			if tc.wantOTelMetrics {
				require.True(t, resourceMetricsContainData(rm))
			} else {
				require.False(t, resourceMetricsContainData(rm))
			}
		})
	}
}

func TestWrapHTTPClientEmitsClientSpanAndPropagatesContext(t *testing.T) {
	traceHarness := testkit.NewTraceHarness(t)
	metricHarness := testkit.NewMetricHarness(t)

	server := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NotEmpty(t, r.Header.Get("Traceparent"))
		w.WriteHeader(http.StatusOK)
	})
	ts := httptest.NewTestServer(t, server)
	client := WrapHTTPClient(ts.Client(), HTTPClientConfig{
		ServiceName:    "autok-client",
		MetricMode:     HTTPMetricModeNone,
		TracerProvider: traceHarness.Provider,
		MeterProvider:  metricHarness.Provider,
	})

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, ts.URL, nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.NoError(t, resp.Body.Close())

	ended := traceHarness.Recorder.Ended()
	require.NotEmpty(t, ended)
}

func TestMakeHTTPInstrumentationOptions(t *testing.T) {
	t.Parallel()

	tp := tracenoop.NewTracerProvider()
	mp := noop.NewMeterProvider()

	cases := []struct {
		name string
		cfg  HTTPConfig
		want int
	}{
		{
			name: "defaults include propagators only",
			cfg:  HTTPConfig{},
			want: 1,
		},
		{
			name: "otel metrics add tracer and meter providers",
			cfg: HTTPConfig{
				TracerProvider: tp,
				MeterProvider:  mp,
				MetricMode:     HTTPMetricModeOTelOnly,
			},
			want: 3,
		},
		{
			name: "custom mode swaps in noop meter provider",
			cfg: HTTPConfig{
				MetricMode: HTTPMetricModeCustomOnly,
			},
			want: 2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			serverOpts := makeHTTPInstrumentationOptions(
				tc.cfg.TracerProvider,
				tc.cfg.MeterProvider,
				tc.cfg.Propagators,
				tc.cfg.MetricMode,
			)
			clientOpts := makeHTTPInstrumentationOptions(
				tc.cfg.TracerProvider,
				tc.cfg.MeterProvider,
				tc.cfg.Propagators,
				tc.cfg.MetricMode,
			)
			require.Len(t, serverOpts, tc.want)
			require.Len(t, clientOpts, tc.want)
		})
	}
}

func (httpAttributeSource) Attributes(_ *http.Request, _ HTTPRequestInfo) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("observer", "framework"),
	}
}

func (r customRecorder) Record(_ context.Context, _ *http.Request, info HTTPRequestInfo) {
	r.recorder.Record(context.Background(), info.Method, info.Route, info.StatusCode, info.Duration)
}

func requireContainsAttribute(t *testing.T, attrs []attribute.KeyValue, key string, value any) {
	t.Helper()
	for _, attr := range attrs {
		if string(attr.Key) != key {
			continue
		}
		switch expected := value.(type) {
		case string:
			require.Equal(t, expected, attr.Value.AsString())
		case bool:
			require.Equal(t, expected, attr.Value.AsBool())
		default:
			t.Fatalf("unsupported expected attribute type %T", value)
		}
		return
	}
	t.Fatalf("missing attribute %q", key)
}

func resourceMetricsContainData(rm metricdata.ResourceMetrics) bool {
	for _, scope := range rm.ScopeMetrics {
		if len(scope.Metrics) > 0 {
			return true
		}
	}
	return false
}
