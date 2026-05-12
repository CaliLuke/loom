package transport

import "net/http"

// HTTPMiddleware returns an HTTP middleware that injects observer into the
// request context so generated handlers can resolve it via
// ObserverFromContext. A nil observer disables event delivery without
// affecting the wrapped handler chain.
//
// HTTPMiddleware is intentionally narrow: it only performs context
// injection. Span/trace setup, propagation, and metric recording remain the
// responsibility of `loom/observability/otel`, which can be composed on the
// outside of this middleware.
func HTTPMiddleware(observer Observer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if observer == nil {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(WithObserver(r.Context(), observer)))
		})
	}
}
