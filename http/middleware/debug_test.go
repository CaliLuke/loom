package middleware_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	loomhttp "github.com/CaliLuke/loom/http"
	httpm "github.com/CaliLuke/loom/http/middleware"
	"github.com/CaliLuke/loom/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDebug(t *testing.T) {
	var (
		buf     bytes.Buffer
		gotBody string
	)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte("hello"))
		require.NoError(t, err)
	})

	req := httptest.NewRequest("POST", "/foo?a=1", strings.NewReader("hi"))
	req.Header.Set("X-Test", "value")
	ctx := context.WithValue(req.Context(), middleware.RequestIDKey, "req-1") // nolint: staticcheck
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	httpm.Debug(loomhttp.NewMuxer(), &buf)(handler).ServeHTTP(rec, req)

	// The middleware restores the request body for the wrapped handler.
	assert.Equal(t, "hi", gotBody, "request body not restored for handler")
	// The response still reaches the client.
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "hello", rec.Body.String())

	want := "> [req-1] POST /foo?a=1\n" +
		"> [req-1] X-Test: value\n" +
		"[req-1] hi\n" +
		"\n< [req-1] OK\n" +
		"< [req-1] Content-Type: application/json\n" +
		"[req-1] hello\n\n"
	assert.Equal(t, want, buf.String())
}

func TestDebugEmptyBodies(t *testing.T) {
	var buf bytes.Buffer
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest("GET", "/health", nil)
	ctx := context.WithValue(req.Context(), middleware.RequestIDKey, "req-2") // nolint: staticcheck
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	httpm.Debug(loomhttp.NewMuxer(), &buf)(handler).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	want := "> [req-2] GET /health\n" +
		"< [req-2] No Content\n"
	assert.Equal(t, want, buf.String())
}

func TestDebugGeneratesRequestID(t *testing.T) {
	var buf bytes.Buffer
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	httpm.Debug(loomhttp.NewMuxer(), &buf)(handler).ServeHTTP(httptest.NewRecorder(), req)

	out := buf.String()
	require.True(t, strings.HasPrefix(out, "> ["), "unexpected debug output: %q", out)
	id := out[3:strings.Index(out, "]")]
	assert.Len(t, id, 8, "generated request ID has unexpected length")
}
