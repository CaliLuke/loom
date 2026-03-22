package otel

import (
	"context"
	"net/http"
	"regexp"
	"sync"
	"time"

	goahttpotel "github.com/CaliLuke/loom/v3/http/middleware/otel"
	"github.com/felixge/httpsnoop"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type (
	// HTTPMetricMode controls whether transport-level HTTP metrics are emitted by
	// otelhttp, by a custom recorder, by both, or by neither.
	HTTPMetricMode string

	// HTTPConfig configures HTTP server-side OpenTelemetry instrumentation.
	HTTPConfig struct {
		// ServiceName is the fallback operation name when no Loom route pattern is
		// available.
		ServiceName string
		// MetricMode selects otelhttp metrics, custom metrics, both, or neither.
		MetricMode HTTPMetricMode
		// TracerProvider overrides the trace provider used by otelhttp.
		TracerProvider trace.TracerProvider
		// MeterProvider overrides the meter provider used by otelhttp.
		MeterProvider metric.MeterProvider
		// Propagators overrides the propagators used by otelhttp.
		Propagators propagation.TextMapPropagator
		// AttributeSource returns post-response span attributes.
		AttributeSource HTTPAttributeSource
		// MetricsRecorder records custom post-response HTTP metrics.
		MetricsRecorder HTTPMetricsRecorder
	}

	// HTTPClientConfig configures HTTP client-side OpenTelemetry instrumentation.
	HTTPClientConfig struct {
		// ServiceName is the fallback operation name for client spans.
		ServiceName string
		// MetricMode selects otelhttp metrics or suppression for the client
		// transport.
		MetricMode HTTPMetricMode
		// TracerProvider overrides the trace provider used by otelhttp.
		TracerProvider trace.TracerProvider
		// MeterProvider overrides the meter provider used by otelhttp.
		MeterProvider metric.MeterProvider
		// Propagators overrides the propagators used by otelhttp.
		Propagators propagation.TextMapPropagator
	}

	// HTTPRequestInfo describes a completed HTTP request as seen by the
	// observability middleware.
	HTTPRequestInfo struct {
		// Method is the HTTP method.
		Method string
		// Route is the Loom route pattern or fallback method-path name.
		Route string
		// StatusCode is the final HTTP response status.
		StatusCode int
		// Duration is the total request duration observed by the middleware.
		Duration time.Duration
		// PathValues contains resolved path parameter values keyed by parameter
		// name when the route pattern contains named segments.
		PathValues map[string]string
		// Authenticated is reserved for consumers that want to project auth state
		// into the request info externally; the framework does not infer it.
		Authenticated bool
	}

	// HTTPAttributeSource returns span attributes for a completed request.
	HTTPAttributeSource interface {
		Attributes(*http.Request, HTTPRequestInfo) []attribute.KeyValue
	}

	// HTTPMetricsRecorder records custom transport metrics for a completed
	// request.
	HTTPMetricsRecorder interface {
		Record(context.Context, *http.Request, HTTPRequestInfo)
	}

	httpAttributeCollector struct {
		mu    sync.Mutex
		attrs []attribute.KeyValue
	}
)

const (
	// HTTPMetricModeOTelOnly emits otelhttp metrics only.
	HTTPMetricModeOTelOnly HTTPMetricMode = "otel_only"
	// HTTPMetricModeCustomOnly suppresses otelhttp metrics and uses only the
	// custom recorder.
	HTTPMetricModeCustomOnly HTTPMetricMode = "custom_only"
	// HTTPMetricModeBoth emits both otelhttp metrics and the custom recorder.
	HTTPMetricModeBoth HTTPMetricMode = "both"
	// HTTPMetricModeNone emits neither otelhttp metrics nor custom metrics.
	HTTPMetricModeNone HTTPMetricMode = "none"
)

var pathVarPattern = regexp.MustCompile(`\{[*]?([^}:]+)(?::[^}]+)?\}`)

type httpAttributeCollectorKey struct{}

// HTTPMiddleware instruments an HTTP handler chain using the official
// OpenTelemetry HTTP middleware plus Loom-specific post-response enrichment
// hooks.
func HTTPMiddleware(cfg HTTPConfig) func(http.Handler) http.Handler {
	opts := makeHTTPInstrumentationOptions(
		cfg.TracerProvider,
		cfg.MeterProvider,
		cfg.Propagators,
		cfg.MetricMode,
	)
	inner := goahttpotel.Middleware(cfg.ServiceName, opts...)
	return func(next http.Handler) http.Handler {
		return inner(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			collector := &httpAttributeCollector{}
			r = r.WithContext(context.WithValue(r.Context(), httpAttributeCollectorKey{}, collector))
			metrics := httpsnoop.CaptureMetrics(next, w, r)
			info := HTTPRequestInfo{
				Method:     r.Method,
				Route:      routeName(r),
				StatusCode: metrics.Code,
				Duration:   metrics.Duration,
				PathValues: pathValues(r),
			}
			var attrs []attribute.KeyValue
			if cfg.AttributeSource != nil {
				attrs = append(attrs, cfg.AttributeSource.Attributes(r, info)...)
			}
			attrs = append(attrs, collector.snapshot()...)
			if len(attrs) > 0 {
				trace.SpanFromContext(r.Context()).SetAttributes(attrs...)
			}
			if shouldRecordCustomMetrics(cfg.MetricMode) && cfg.MetricsRecorder != nil {
				cfg.MetricsRecorder.Record(r.Context(), r, info)
			}
		}))
	}
}

// WrapHTTPClient returns a shallow copy of client with OpenTelemetry transport
// instrumentation configured according to cfg.
func WrapHTTPClient(client *http.Client, cfg HTTPClientConfig) *http.Client {
	return goahttpotel.WrapClient(client, makeHTTPInstrumentationOptions(
		cfg.TracerProvider,
		cfg.MeterProvider,
		cfg.Propagators,
		cfg.MetricMode,
	)...)
}

func makeHTTPInstrumentationOptions(
	tracerProvider trace.TracerProvider,
	meterProvider metric.MeterProvider,
	propagators propagation.TextMapPropagator,
	mode HTTPMetricMode,
) []goahttpotel.Option {
	opts := make([]goahttpotel.Option, 0, 3)
	if tracerProvider != nil {
		opts = append(opts, otelhttp.WithTracerProvider(tracerProvider))
	}
	opts = append(opts, otelhttp.WithPropagators(defaultPropagators(propagators)))
	if provider := selectedMeterProvider(meterProvider, mode); provider != nil {
		opts = append(opts, otelhttp.WithMeterProvider(provider))
	}
	return opts
}

func selectedMeterProvider(provider metric.MeterProvider, mode HTTPMetricMode) metric.MeterProvider {
	switch mode {
	case HTTPMetricModeCustomOnly, HTTPMetricModeNone:
		return noop.NewMeterProvider()
	default:
		return provider
	}
}

func shouldRecordCustomMetrics(mode HTTPMetricMode) bool {
	switch mode {
	case HTTPMetricModeCustomOnly, HTTPMetricModeBoth:
		return true
	default:
		return false
	}
}

func defaultPropagators(propagators propagation.TextMapPropagator) propagation.TextMapPropagator {
	if propagators != nil {
		return propagators
	}
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

func routeName(r *http.Request) string {
	if r.Pattern != "" {
		return r.Pattern
	}
	return r.Method + " " + r.URL.Path
}

func pathValues(r *http.Request) map[string]string {
	matches := pathVarPattern.FindAllStringSubmatch(r.Pattern, -1)
	if len(matches) == 0 {
		return nil
	}
	values := make(map[string]string, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		values[match[1]] = r.PathValue(match[1])
	}
	return values
}

// AddHTTPAttributes appends request-scoped transport attributes to the active
// HTTP observability collector when the request is running inside
// HTTPMiddleware.
func AddHTTPAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	collector, _ := ctx.Value(httpAttributeCollectorKey{}).(*httpAttributeCollector)
	if collector == nil || len(attrs) == 0 {
		return
	}
	collector.add(attrs...)
}

func (c *httpAttributeCollector) add(attrs ...attribute.KeyValue) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.attrs = append(c.attrs, attrs...)
}

func (c *httpAttributeCollector) snapshot() []attribute.KeyValue {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]attribute.KeyValue, len(c.attrs))
	copy(out, c.attrs)
	return out
}
