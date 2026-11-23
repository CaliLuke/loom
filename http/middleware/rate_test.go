package middleware_test

import (
	"context"
	"encoding/json/v2"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/admission"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/CaliLuke/loom/http/middleware"
	loom "github.com/CaliLuke/loom/pkg"
)

func TestRateLimitHTTP(t *testing.T) {
	for _, tc := range []struct {
		name                   string
		decision               admission.RateDecision
		backendError, keyError error
		status, calls          int
		retry                  string
	}{
		{"admitted", admission.RateDecision{Allowed: true}, nil, nil, 204, 1, ""},
		{"known delay", admission.RateDecision{RetryAfter: time.Second + time.Nanosecond}, nil, nil, 429, 0, "2"},
		{"subsecond", admission.RateDecision{RetryAfter: time.Nanosecond}, nil, nil, 429, 0, "1"},
		{"unknown delay", admission.RateDecision{}, nil, nil, 429, 0, ""},
		{"backend error", admission.RateDecision{Allowed: true}, errors.New("sensitive backend"), nil, 503, 0, ""},
		{"key error", admission.RateDecision{}, nil, errors.New("sensitive grouping"), 503, 0, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			checks := 0
			control, err := admission.NewRateAdmission(func(_ context.Context, key string) (admission.RateDecision, error) {
				checks++
				require.Equal(t, "bounded-group", key)
				return tc.decision, tc.backendError
			}, time.Second)
			require.NoError(t, err)
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			calls := 0
			handler := middleware.RateLimit(control, func(*http.Request) (string, error) {
				return "bounded-group", tc.keyError
			})(http.HandlerFunc(func(w http.ResponseWriter, got *http.Request) {
				calls++
				require.Same(t, req, got)
				require.NoError(t, got.Context().Err())
				w.WriteHeader(http.StatusNoContent)
			}))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, req)
			require.Equal(t, tc.status, response.Code)
			require.Equal(t, tc.calls, calls)
			require.Equal(t, tc.retry, response.Header().Get("Retry-After"))
			require.NotContains(t, response.Body.String(), "sensitive")
			if tc.keyError != nil {
				require.Zero(t, checks)
			} else {
				require.Equal(t, 1, checks)
			}
		})
	}
}

func TestRateLimitHTTPTimeoutAndCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		checks, calls := 0, 0
		control, err := admission.NewRateAdmission(func(ctx context.Context, _ string) (admission.RateDecision, error) {
			checks++
			<-ctx.Done()
			return admission.RateDecision{}, ctx.Err()
		}, 100*time.Millisecond)
		require.NoError(t, err)
		handler := middleware.RateLimit(control, func(*http.Request) (string, error) {
			return "global", nil
		})(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			calls++
		}))
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		response := httptest.NewRecorder()
		start := time.Now()
		handler.ServeHTTP(response, req)
		require.Equal(t, 100*time.Millisecond, time.Since(start))
		require.Equal(t, 503, response.Code)
		require.Equal(t, 1, checks)
		require.Zero(t, calls)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		handler.ServeHTTP(httptest.NewRecorder(), req.WithContext(ctx))
		require.Equal(t, 1, checks)
		require.Zero(t, calls)
	})
}

func TestRateLimitHTTPConfiguration(t *testing.T) {
	key := func(*http.Request) (string, error) {
		return "", nil
	}
	require.Panics(t, func() {
		middleware.RateLimit(nil, key)
	})
	control, err := admission.NewRateAdmission(func(context.Context, string) (admission.RateDecision, error) {
		return admission.RateDecision{}, nil
	}, time.Second)
	require.NoError(t, err)
	require.Panics(t, func() {
		middleware.RateLimit(control, nil)
	})
	require.Panics(t, func() {
		middleware.RateLimit(control, key)(nil)
	})
}

func TestRateLimitHTTPDefaultResponseBody(t *testing.T) {
	for _, tc := range []struct {
		name     string
		decision admission.RateDecision
		err      error
		status   int
	}{
		{"denied", admission.RateDecision{RetryAfter: time.Second}, nil, http.StatusTooManyRequests},
		{"unavailable", admission.RateDecision{}, errors.New("sensitive backend"), http.StatusServiceUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := serveRateLimited(t, tc.decision, tc.err, nil)
			require.Equal(t, tc.status, response.Code)
			require.Equal(t, "text/plain; charset=utf-8", response.Header().Get("Content-Type"))
			require.Equal(t, "nosniff", response.Header().Get("X-Content-Type-Options"))
			require.Equal(t, http.StatusText(tc.status)+"\n", response.Body.String())
		})
	}
}

func TestRateLimitHTTPResponder(t *testing.T) {
	for _, tc := range []struct {
		name     string
		decision admission.RateDecision
		err      error
		status   int
		code     string
		retry    string
	}{
		{"denied with delay", admission.RateDecision{RetryAfter: 1500 * time.Millisecond}, nil, 429, "rate_limited", "2"},
		{"denied unknown delay", admission.RateDecision{}, nil, 429, "rate_limited", ""},
		{"backend error", admission.RateDecision{}, errors.New("sensitive backend"), 503, "rate_limit_unavailable", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got middleware.RateLimitRejection
			response := serveRateLimited(t, tc.decision, tc.err, func(header http.Header, _ *http.Request, rejection middleware.RateLimitRejection) []byte {
				got = rejection
				header.Set("Content-Type", "application/json")
				// RateLimit owns status and Retry-After; responder values are replaced.
				header.Set("Retry-After", "999")
				return []byte(`{"error":"limited"}`)
			})
			require.Equal(t, tc.status, response.Code)
			require.Equal(t, tc.status, got.Status)
			require.Equal(t, tc.retry, response.Header().Get("Retry-After"))
			require.Equal(t, "application/json", response.Header().Get("Content-Type"))
			require.JSONEq(t, `{"error":"limited"}`, response.Body.String())
			var serviceErr *loom.ServiceError
			require.ErrorAs(t, got.Err, &serviceErr)
			require.Equal(t, tc.code, serviceErr.Name)
			require.True(t, serviceErr.Temporary)
			require.NotContains(t, got.Err.Error(), "sensitive")
			if tc.status == http.StatusTooManyRequests {
				require.Equal(t, tc.decision, got.Decision)
			} else {
				require.Zero(t, got.Decision)
			}
		})
	}
}

func TestRateLimitHTTPKeyErrorUsesResponder(t *testing.T) {
	control, err := admission.NewRateAdmission(func(context.Context, string) (admission.RateDecision, error) {
		return admission.RateDecision{Allowed: true}, nil
	}, time.Second)
	require.NoError(t, err)
	var got middleware.RateLimitRejection
	handler := middleware.RateLimit(control, func(*http.Request) (string, error) {
		return "", errors.New("sensitive grouping")
	}, middleware.WithRateLimitResponder(func(_ http.Header, _ *http.Request, rejection middleware.RateLimitRejection) []byte {
		got = rejection
		return nil
	}))(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("handler must not run")
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	require.Equal(t, http.StatusServiceUnavailable, response.Code)
	require.Empty(t, response.Body.String(), "a nil body writes no body")
	require.Equal(t, http.StatusServiceUnavailable, got.Status)
	require.NotContains(t, got.Err.Error(), "sensitive")
}

func TestProblemRateLimitResponse(t *testing.T) {
	for _, tc := range []struct {
		name     string
		decision admission.RateDecision
		err      error
		status   int
		code     string
	}{
		{"denied", admission.RateDecision{RetryAfter: time.Second}, nil, http.StatusTooManyRequests, "rate_limited"},
		{"unavailable", admission.RateDecision{}, errors.New("sensitive backend"), http.StatusServiceUnavailable, "rate_limit_unavailable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := serveRateLimited(t, tc.decision, tc.err, middleware.ProblemRateLimitResponse)
			require.Equal(t, tc.status, response.Code)
			require.Equal(t, loomhttp.ProblemJSONContentType, response.Header().Get("Content-Type"))
			var problem loomhttp.ProblemResponse
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &problem))
			require.Equal(t, tc.status, problem.Status)
			require.Equal(t, tc.code, problem.Code)
			require.True(t, problem.Temporary)
			require.NotContains(t, response.Body.String(), "sensitive")
			if tc.status == http.StatusTooManyRequests {
				require.Equal(t, "1", response.Header().Get("Retry-After"))
			}
		})
	}
}

func TestRateLimitHTTPOptionConfiguration(t *testing.T) {
	control, err := admission.NewRateAdmission(func(context.Context, string) (admission.RateDecision, error) {
		return admission.RateDecision{}, nil
	}, time.Second)
	require.NoError(t, err)
	key := func(*http.Request) (string, error) {
		return "", nil
	}
	require.Panics(t, func() {
		middleware.RateLimit(control, key, nil)
	})
	require.Panics(t, func() {
		middleware.RateLimit(control, key, middleware.WithRateLimitResponder(nil))
	})
}

func serveRateLimited(t *testing.T, decision admission.RateDecision, backendErr error, respond middleware.RateLimitResponder) *httptest.ResponseRecorder {
	t.Helper()
	control, err := admission.NewRateAdmission(func(context.Context, string) (admission.RateDecision, error) {
		return decision, backendErr
	}, time.Second)
	require.NoError(t, err)
	var opts []middleware.RateLimitOption
	if respond != nil {
		opts = append(opts, middleware.WithRateLimitResponder(respond))
	}
	handler := middleware.RateLimit(control, func(*http.Request) (string, error) {
		return "group", nil
	}, opts...)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("handler must not run")
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	return response
}

func TestRateLimitHTTPDropsStaleContentLength(t *testing.T) {
	control, err := admission.NewRateAdmission(func(context.Context, string) (admission.RateDecision, error) {
		return admission.RateDecision{}, nil
	}, time.Second)
	require.NoError(t, err)
	outer := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Length", "12345")
			next.ServeHTTP(w, r)
		})
	}
	server := httptest.NewTestServer(t, outer(middleware.RateLimit(control, func(*http.Request) (string, error) {
		return "group", nil
	})(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("handler must not run")
	}))))
	client := server.Client()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, resp.Body.Close())
	})
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
	require.Equal(t, http.StatusText(http.StatusTooManyRequests)+"\n", string(body))
}
