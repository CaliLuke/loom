package middleware

import (
	"net/http"
	"strconv"

	"github.com/CaliLuke/loom/admission"
)

// RateLimit admits HTTP requests before invoking the next handler. Applications
// choose a trusted grouping function and local or shared quota backend. Quota
// denial returns 429 and a whole-second Retry-After when known. Grouping/backend
// errors and timeouts return 503 with no sensitive details. No retry occurs.
//
// Place early limits outside the generated handler to protect decoding and
// authentication. Authenticated quotas belong at protected endpoint middleware.
// The request/context is passed unchanged after admission. Do not derive keys
// from forwarded headers without a trusted proxy policy, or expose keys as metric
// labels. Construction panics for nil dependencies.
func RateLimit(control *admission.RateAdmission, key func(*http.Request) (string, error)) func(http.Handler) http.Handler {
	if control == nil || key == nil {
		panic("rate limit: admission and grouping function required")
	}
	return func(next http.Handler) http.Handler {
		if next == nil {
			panic("rate limit: non-nil handler required")
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Context().Err() != nil {
				return
			}
			group, err := key(r)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
				return
			}
			decision, err := control.Check(r.Context(), group)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
				return
			}
			if !decision.Allowed {
				if seconds := decision.RetryAfterSeconds(); seconds > 0 {
					w.Header().Set("Retry-After", strconv.FormatInt(seconds, 10))
				}
				http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
