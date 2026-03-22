package otel

import (
	"context"
	"io"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	goahttp "github.com/CaliLuke/loom/http"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"github.com/stretchr/testify/require"
)

func TestMiddlewareUsesMatchedRoutePatternAsSpanName(t *testing.T) {
	t.Parallel()

	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider()
	tp.RegisterSpanProcessor(recorder)
	t.Cleanup(func() {
		require.NoError(t, tp.Shutdown(context.Background()))
	})

	mux := goahttp.NewMuxer()
	mux.Use(Middleware("users-service", otelhttp.WithTracerProvider(tp)))
	mux.Handle(stdhttp.MethodGet, "/users/{id}", func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		w.WriteHeader(stdhttp.StatusNoContent)
	})

	req := httptest.NewRequest(stdhttp.MethodGet, "/users/42", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	spans := recorder.Ended()
	require.Len(t, spans, 1)
	require.Equal(t, "GET /users/{id}", spans[0].Name())
}

func TestWrapClientCreatesClientSpan(t *testing.T) {
	t.Parallel()

	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider()
	tp.RegisterSpanProcessor(recorder)
	t.Cleanup(func() {
		require.NoError(t, tp.Shutdown(context.Background()))
	})

	srv := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		w.WriteHeader(stdhttp.StatusAccepted)
		_, err := w.Write([]byte("ok"))
		require.NoError(t, err)
	}))
	t.Cleanup(srv.Close)

	client := WrapClient(&stdhttp.Client{}, otelhttp.WithTracerProvider(tp))
	req, err := stdhttp.NewRequestWithContext(context.Background(), stdhttp.MethodGet, srv.URL, nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	_, err = io.Copy(io.Discard, resp.Body)
	require.NoError(t, err)

	spans := recorder.Ended()
	require.Len(t, spans, 1)
	require.Equal(t, trace.SpanKindClient, spans[0].SpanKind())
	require.Contains(t, spanAttributes(spans[0].Attributes()), "http.request.method")
}

func spanAttributes(attrs []attribute.KeyValue) map[string]attribute.Value {
	m := make(map[string]attribute.Value, len(attrs))
	for _, attr := range attrs {
		m[string(attr.Key)] = attr.Value
	}
	return m
}
