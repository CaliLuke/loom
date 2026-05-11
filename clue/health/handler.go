package health

import (
	"context"
	"net/http"

	loomhttp "github.com/CaliLuke/loom/http"
)

// Handler returns a HTTP handler that serves health check requests. The
// response body is the health status returned by chk.Check(). By default
// it's encoded as JSON, but you can specify a different encoding in the
// HTTP Accept header. The response status is 200 if chk.Check() returns
// a nil error, 503 otherwise.
func Handler(chk Checker) http.HandlerFunc {
	encoder := loomhttp.ResponseEncoder
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), loomhttp.AcceptTypeKey, r.Header.Get("Accept"))
		enc := encoder(ctx, w)
		h, healthy := chk.Check(ctx)
		if healthy {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		enc.Encode(h) // nolint: errcheck
	})
}
