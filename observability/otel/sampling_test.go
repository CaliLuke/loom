package otel

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func TestTraceProviderRetainsParentSampling(t *testing.T) {
	for _, ratio := range []float64{0, 0.5, 1} {
		for _, remote := range []bool{false, true} {
			for _, sampled := range []bool{false, true} {
				t.Run(fmt.Sprintf("ratio=%g/remote=%t/sampled=%t", ratio, remote, sampled), func(t *testing.T) {
					provider, err := newTraceProvider(context.Background(), TraceConfig{SampleRatio: ratio}, resource.Empty())
					require.NoError(t, err)
					t.Cleanup(func() {
						require.NoError(t, provider.Shutdown(context.Background()))
					})
					parent := samplingParent(t, remote, sampled)
					ctx := trace.ContextWithSpanContext(context.Background(), parent)
					_, span := provider.Tracer("sampling").Start(ctx, "child")
					defer span.End()
					require.Equal(t, sampled, span.SpanContext().IsSampled())
					require.Equal(t, sampled, span.IsRecording())
					require.Equal(t, parent.TraceID(), span.SpanContext().TraceID())
					require.Equal(t, parent.TraceState(), span.SpanContext().TraceState())
				})
			}
		}
	}
}

func TestTraceProviderRootSampling(t *testing.T) {
	for _, tc := range []struct {
		ratio     float64
		rootRatio float64
	}{
		{0, 1}, {-1, 1}, {2, 1}, {1, 1}, {0.5, 0.5},
	} {
		for _, mode := range []string{"no parent", "invalid parent", "new root"} {
			t.Run(fmt.Sprintf("ratio=%g/%s", tc.ratio, mode), func(t *testing.T) {
				provider, err := newTraceProvider(context.Background(), TraceConfig{SampleRatio: tc.ratio}, resource.Empty())
				require.NoError(t, err)
				t.Cleanup(func() {
					require.NoError(t, provider.Shutdown(context.Background()))
				})
				ctx := context.Background()
				var opts []trace.SpanStartOption
				switch mode {
				case "invalid parent":
					ctx = trace.ContextWithSpanContext(ctx, trace.NewSpanContext(trace.SpanContextConfig{TraceFlags: trace.FlagsSampled, Remote: true}))
				case "new root":
					ctx = trace.ContextWithSpanContext(ctx, samplingParent(t, true, false))
					opts = append(opts, trace.WithNewRoot())
				}
				_, span := provider.Tracer("sampling").Start(ctx, "root", opts...)
				defer span.End()
				// The SDK owns the ratio algorithm. Compare the actual root ID,
				// so this checks routing to that policy without statistical sampling.
				expected := sdktrace.TraceIDRatioBased(tc.rootRatio).ShouldSample(sdktrace.SamplingParameters{ParentContext: context.Background(), TraceID: span.SpanContext().TraceID()})
				require.Equal(t, expected.Decision == sdktrace.RecordAndSample, span.SpanContext().IsSampled())
				require.True(t, span.SpanContext().IsValid())
			})
		}
	}
}

func TestHTTPPropagationRetainsSampling(t *testing.T) {
	for _, sampled := range []bool{false, true} {
		t.Run(fmt.Sprintf("sampled=%t", sampled), func(t *testing.T) {
			provider, err := newTraceProvider(context.Background(), TraceConfig{SampleRatio: 0.5}, resource.Empty())
			require.NoError(t, err)
			t.Cleanup(func() {
				require.NoError(t, provider.Shutdown(context.Background()))
			})
			parent := samplingParent(t, true, sampled)
			req := httptest.NewRequest(http.MethodGet, "/sampling", nil)
			propagation.TraceContext{}.Inject(trace.ContextWithSpanContext(req.Context(), parent), propagation.HeaderCarrier(req.Header))
			called := false
			handler := HTTPMiddleware(HTTPConfig{ServiceName: "sampling", TracerProvider: provider, Propagators: propagation.TraceContext{}, MetricMode: HTTPMetricModeNone})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				span := trace.SpanFromContext(r.Context())
				require.Equal(t, sampled, span.SpanContext().IsSampled())
				require.Equal(t, sampled, span.IsRecording())
				require.Equal(t, parent.TraceID(), span.SpanContext().TraceID())
				require.Equal(t, parent.TraceState(), span.SpanContext().TraceState())
				w.WriteHeader(http.StatusNoContent)
			}))
			handler.ServeHTTP(httptest.NewRecorder(), req)
			require.True(t, called)
		})
	}
}

func samplingParent(t *testing.T, remote, sampled bool) trace.SpanContext {
	t.Helper()
	id := "00000000000000000000000000000001"
	var flags trace.TraceFlags
	if sampled {
		id = "ffffffffffffffffffffffffffffffff"
		flags = trace.FlagsSampled
	}
	traceID, err := trace.TraceIDFromHex(id)
	require.NoError(t, err)
	spanID, err := trace.SpanIDFromHex("0000000000000001")
	require.NoError(t, err)
	state, err := trace.ParseTraceState("vendor=value")
	require.NoError(t, err)
	return trace.NewSpanContext(trace.SpanContextConfig{TraceID: traceID, SpanID: spanID, TraceFlags: flags, TraceState: state, Remote: remote})
}
