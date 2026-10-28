package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/admission"
	"github.com/CaliLuke/loom/http/middleware"
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
