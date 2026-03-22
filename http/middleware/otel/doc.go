/*
Package otel provides thin OpenTelemetry middleware helpers for Loom HTTP
servers and clients.

The package intentionally wraps the official OpenTelemetry contrib HTTP
instrumentation instead of implementing tracing itself. Applications remain
responsible for configuring the tracer provider, exporter, and resource
attributes. Loom owns the transport seam so generated and hand-written handlers
can share one instrumentation path.

Use Middleware with goahttp.NewMuxer so spans can inherit the matched
METHOD-plus-route pattern from r.Pattern:

	mux := goahttp.NewMuxer()
	mux.Use(otel.Middleware("service"))

For generated HTTP clients, wrap an *http.Client before passing it anywhere a
goa HTTP Doer is expected:

	client := otel.WrapClient(&http.Client{})
*/
package otel
