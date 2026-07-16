package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	loomhttp "github.com/CaliLuke/loom/http"
	httpm "github.com/CaliLuke/loom/http/middleware"
	"github.com/CaliLuke/loom/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// capturingLogger records every Log call made against it.
type capturingLogger struct {
	entries [][]any
}

func (l *capturingLogger) Log(keyvals ...any) error {
	l.entries = append(l.entries, append([]any(nil), keyvals...))
	return nil
}

func TestLog(t *testing.T) {
	cases := []struct {
		name          string
		requestID     string
		xForwardedFor string
		trustedProxy  bool
		status        int
		body          string
		wantFrom      string
	}{
		{
			name:          "ignores forwarded address without trusted metadata",
			requestID:     "req-1",
			xForwardedFor: "1.2.3.4",
			status:        http.StatusCreated,
			body:          "hello",
			wantFrom:      "192.0.2.1",
		},
		{
			name:          "logs trusted forwarded client",
			requestID:     "req-trusted",
			xForwardedFor: "1.2.3.4, 10.0.0.1",
			trustedProxy:  true,
			status:        http.StatusOK,
			wantFrom:      "1.2.3.4",
		},
		{
			name:      "falls back to remote address",
			requestID: "req-2",
			status:    http.StatusOK,
			body:      "",
			// httptest.NewRequest sets RemoteAddr to 192.0.2.1:1234;
			// from() strips the port.
			wantFrom: "192.0.2.1",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			logger := &capturingLogger{}
			handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(c.status)
				if c.body != "" {
					_, err := w.Write([]byte(c.body))
					require.NoError(t, err)
				}
			})

			req := httptest.NewRequest("GET", "/foo?a=1", nil)
			ctx := context.WithValue(req.Context(), middleware.RequestIDKey, c.requestID) // nolint: staticcheck
			req = req.WithContext(ctx)
			if c.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", c.xForwardedFor)
			}

			logged := httpm.Log(logger)(handler)
			if c.trustedProxy {
				policy, err := loomhttp.NewRequestMetadataPolicy(nil, []netip.Prefix{
					netip.MustParsePrefix("192.0.2.0/24"),
					netip.MustParsePrefix("10.0.0.0/8"),
				})
				require.NoError(t, err)
				logged = loomhttp.RequestMetadataMiddleware(policy)(logged)
			}
			logged.ServeHTTP(httptest.NewRecorder(), req)

			require.Len(t, logger.entries, 2)
			assert.Equal(t, []any{
				"id", c.requestID,
				"req", "GET /foo?a=1",
				"from", c.wantFrom,
			}, logger.entries[0])

			resp := logger.entries[1]
			require.Len(t, resp, 8)
			assert.Equal(t, []any{
				"id", c.requestID,
				"status", c.status,
				"bytes", len(c.body),
			}, resp[:6])
			assert.Equal(t, "time", resp[6])
			assert.IsType(t, "", resp[7])
		})
	}
}

func TestLogGeneratesRequestID(t *testing.T) {
	logger := &capturingLogger{}
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	httpm.Log(logger)(handler).ServeHTTP(httptest.NewRecorder(), req)

	require.Len(t, logger.entries, 2)
	id, ok := logger.entries[0][1].(string)
	require.True(t, ok, "generated request ID is not a string")
	assert.Len(t, id, 8)
	assert.Equal(t, id, logger.entries[1][1], "request and response logged with different IDs")
}

func TestLogContext(t *testing.T) {
	t.Run("uses logger from context", func(t *testing.T) {
		logger := &capturingLogger{}
		var called bool
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})
		fromCtx := func(context.Context) middleware.Logger {
			return logger
		}

		req := httptest.NewRequest("GET", "/", nil)
		httpm.LogContext(fromCtx)(handler).ServeHTTP(httptest.NewRecorder(), req)

		assert.True(t, called, "wrapped handler was not invoked")
		assert.Len(t, logger.entries, 2)
	})

	t.Run("skips logging when logger is nil", func(t *testing.T) {
		var called bool
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})
		fromCtx := func(context.Context) middleware.Logger {
			return nil
		}

		req := httptest.NewRequest("GET", "/", nil)
		httpm.LogContext(fromCtx)(handler).ServeHTTP(httptest.NewRecorder(), req)

		assert.True(t, called, "wrapped handler was not invoked")
	})
}
