package middleware

import (
	"encoding/json/v2"
	"net/http"
	"strconv"

	"github.com/CaliLuke/loom/admission"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
)

type (
	// RateLimitOption configures RateLimit.
	RateLimitOption func(*rateLimitConfig)

	// RateLimitRejection describes one request that RateLimit refuses.
	RateLimitRejection struct {
		// Status is http.StatusTooManyRequests for quota denial or
		// http.StatusServiceUnavailable for grouping and backend failures.
		Status int
		// Decision is the denial decision for a 429 rejection. It is zero for a
		// 503 rejection.
		Decision admission.RateDecision
		// Err is a safe temporary *loom.ServiceError: "rate_limited" for quota
		// denial or "rate_limit_unavailable" for failures. It never contains
		// grouping keys or backend error text.
		Err error
	}

	// RateLimitResponder renders the body of a rejection. It may set response
	// headers, such as Content-Type, on header and returns the body bytes.
	// RateLimit owns the status code and framing: it writes rejection.Status,
	// replaces any Retry-After set by the responder and removes any
	// Content-Length. A nil body writes no body.
	RateLimitResponder func(header http.Header, r *http.Request, rejection RateLimitRejection) []byte

	rateLimitConfig struct {
		respond RateLimitResponder
	}
)

// RateLimit admits HTTP requests before invoking the next handler. Applications
// choose a trusted grouping function and local or shared quota backend. Quota
// denial returns 429 and a whole-second Retry-After when known. Grouping/backend
// errors and timeouts return 503 with no sensitive details. No retry occurs.
// Rejections use a text/plain status text body unless WithRateLimitResponder
// selects another representation; the status and Retry-After are unchanged.
//
// Place early limits outside the generated handler to protect decoding and
// authentication. Authenticated quotas belong at protected endpoint middleware.
// The request/context is passed unchanged after admission. Do not derive keys
// from forwarded headers without a trusted proxy policy, or expose keys as metric
// labels. Construction panics for nil dependencies, including a nil responder.
func RateLimit(control *admission.RateAdmission, key func(*http.Request) (string, error), opts ...RateLimitOption) func(http.Handler) http.Handler {
	if control == nil || key == nil {
		panic("rate limit: admission and grouping function required")
	}
	cfg := rateLimitConfig{respond: plainRateLimitResponse}
	for _, opt := range opts {
		if opt == nil {
			panic("rate limit: non-nil option required")
		}
		opt(&cfg)
	}
	if cfg.respond == nil {
		panic("rate limit: non-nil responder required")
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
				cfg.reject(w, r, unavailableRejection())
				return
			}
			decision, err := control.Check(r.Context(), group)
			if err != nil {
				cfg.reject(w, r, unavailableRejection())
				return
			}
			if !decision.Allowed {
				cfg.reject(w, r, RateLimitRejection{
					Status:   http.StatusTooManyRequests,
					Decision: decision,
					Err:      admission.RateLimitedError(),
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// WithRateLimitResponder selects how RateLimit renders rejection bodies.
func WithRateLimitResponder(respond RateLimitResponder) RateLimitOption {
	return func(cfg *rateLimitConfig) {
		cfg.respond = respond
	}
}

// ProblemRateLimitResponse is a RateLimitResponder that renders the rejection
// as the RFC 9457 application/problem+json document Loom uses for service
// errors, with code "rate_limited" or "rate_limit_unavailable" and the
// rejection status.
func ProblemRateLimitResponse(header http.Header, r *http.Request, rejection RateLimitRejection) []byte {
	problem := loomhttp.NewProblemResponse(r.Context(), rejection.Err, rejection.Status, "", "")
	body, err := json.Marshal(problem, loom.JSONOptions())
	if err != nil {
		// ProblemResponse holds only strings, ints and booleans, so encoding
		// cannot fail; fall back to the status text to stay well formed.
		return plainRateLimitResponse(header, r, rejection)
	}
	header.Set("Content-Type", loomhttp.ProblemJSONContentType)
	header.Set("X-Content-Type-Options", "nosniff")
	return append(body, '\n')
}

func unavailableRejection() RateLimitRejection {
	return RateLimitRejection{
		Status: http.StatusServiceUnavailable,
		Err:    admission.RateLimitUnavailableError(),
	}
}

// plainRateLimitResponse matches http.Error's status text response.
func plainRateLimitResponse(header http.Header, _ *http.Request, rejection RateLimitRejection) []byte {
	header.Set("Content-Type", "text/plain; charset=utf-8")
	header.Set("X-Content-Type-Options", "nosniff")
	return []byte(http.StatusText(rejection.Status) + "\n")
}

func (cfg rateLimitConfig) reject(w http.ResponseWriter, r *http.Request, rejection RateLimitRejection) {
	header := w.Header()
	body := cfg.respond(header, r, rejection)
	// Like http.Error, drop a Content-Length that outer handlers may have set
	// for a response this rejection replaces.
	header.Del("Content-Length")
	header.Del("Retry-After")
	if seconds := rejection.Decision.RetryAfterSeconds(); seconds > 0 {
		header.Set("Retry-After", strconv.FormatInt(seconds, 10))
	}
	w.WriteHeader(rejection.Status)
	if len(body) == 0 {
		return
	}
	if _, err := w.Write(body); err != nil {
		// The status is already committed, so a failed body write means the
		// client is gone; like http.Error, there is no caller to report to.
		return
	}
}
