package otel

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type (
	// Option configures HTTP OpenTelemetry instrumentation.
	Option = otelhttp.Option
)

// Middleware instruments an HTTP handler chain using the official
// OpenTelemetry HTTP middleware.
//
// The provided service name is used as the fallback operation name. When the
// request carries a matched Goa route pattern in r.Pattern, the middleware uses
// that pattern as the span name so spans remain stable across path parameters.
func Middleware(service string, opts ...Option) func(http.Handler) http.Handler {
	return otelhttp.NewMiddleware(service, append([]Option{
		otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
			if r.Pattern != "" {
				return r.Pattern
			}
			return operation
		}),
	}, opts...)...)
}

// Handler wraps an HTTP handler with OpenTelemetry instrumentation.
func Handler(handler http.Handler, service string, opts ...Option) http.Handler {
	return Middleware(service, opts...)(handler)
}

// WrapTransport instruments an HTTP RoundTripper using the official
// OpenTelemetry HTTP transport wrapper.
func WrapTransport(rt http.RoundTripper, opts ...Option) http.RoundTripper {
	return otelhttp.NewTransport(rt, opts...)
}

// WrapClient returns a shallow copy of client whose Transport is instrumented
// with OpenTelemetry.
//
// If client is nil, WrapClient returns a new client using the default
// instrumented transport.
func WrapClient(client *http.Client, opts ...Option) *http.Client {
	if client == nil {
		return &http.Client{Transport: WrapTransport(nil, opts...)}
	}
	clone := *client
	clone.Transport = WrapTransport(clone.Transport, opts...)
	return &clone
}
