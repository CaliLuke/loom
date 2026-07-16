package middleware

import (
	"context"
	"net/http"
	"time"

	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/CaliLuke/loom/middleware"
)

// Log returns a middleware that logs incoming HTTP requests and outgoing
// responses. The middleware uses the request ID set by the RequestID middleware
// or creates a short unique request ID if missing for each incoming request and
// logs it with the request and corresponding response details.
//
// The middleware logs the incoming requests HTTP method and path as well as the
// originator of the request. The originator comes from the trusted request
// metadata snapshot when present and otherwise uses the direct network peer.
// The middleware also logs the response HTTP status code, body length (in
// bytes), and timing information.
//
// Deprecated: use github.com/CaliLuke/loom/http/middleware/otel instead. This
// function will be removed in a future version of Loom.
func Log(l middleware.Logger) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log(l, r, w, h)
		})
	}
}

// LogContext returns a middleware that logs the incoming requests similarly to
// Log. LogContext calls the given function with the request context to extract
// the logger.
//
// Deprecated: use github.com/CaliLuke/loom/http/middleware/otel instead. This
// function will be removed in a future version of Loom.
func LogContext(logFromCtx func(context.Context) middleware.Logger) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			l := logFromCtx(r.Context())
			if l == nil {
				h.ServeHTTP(w, r)
				return
			}
			log(l, r, w, h)
		})
	}
}

// log does the actual logging given the logger.
func log(l middleware.Logger, r *http.Request, w http.ResponseWriter, next http.Handler) {
	reqID := r.Context().Value(middleware.RequestIDKey)
	if reqID == nil {
		reqID = shortID()
	}
	started := time.Now()

	l.Log("id", reqID, // nolint: errcheck
		"req", r.Method+" "+r.URL.String(),
		"from", from(r))

	rw := CaptureResponse(w)
	next.ServeHTTP(rw, r)

	l.Log("id", reqID, // nolint: errcheck
		"status", rw.StatusCode,
		"bytes", rw.ContentLength,
		"time", time.Since(started).String())
}

// from makes a best effort to compute the request client IP.
func from(req *http.Request) string {
	return loomhttp.EffectiveClientAddress(req)
}
