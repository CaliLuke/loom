package middleware_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	loomhttp "github.com/CaliLuke/loom/http"
	httpm "github.com/CaliLuke/loom/http/middleware"
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

	rec := httptest.NewRecorder()
	httpm.Debug(loomhttp.NewMuxer(), &buf)(handler).ServeHTTP(rec, req)

	// The middleware restores the request body for the wrapped handler.
	assert.Equal(t, "hi", gotBody, "request body not restored for handler")
	// The response still reaches the client.
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "hello", rec.Body.String())

	id := debugRequestID(t, buf.String())
	want := "> [" + id + "] POST /foo?a=1\n" +
		"> [" + id + "] X-Test: value\n" +
		"[" + id + "] hi\n" +
		"\n< [" + id + "] OK\n" +
		"< [" + id + "] Content-Type: application/json\n" +
		"[" + id + "] hello\n\n"
	assert.Equal(t, want, buf.String())
}

func TestDebugEmptyBodies(t *testing.T) {
	var buf bytes.Buffer
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest("GET", "/health", nil)

	rec := httptest.NewRecorder()
	httpm.Debug(loomhttp.NewMuxer(), &buf)(handler).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	id := debugRequestID(t, buf.String())
	want := "> [" + id + "] GET /health\n" +
		"< [" + id + "] No Content\n"
	assert.Equal(t, want, buf.String())
}

func TestDebugGeneratesRequestID(t *testing.T) {
	var buf bytes.Buffer
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	httpm.Debug(loomhttp.NewMuxer(), &buf)(handler).ServeHTTP(httptest.NewRecorder(), req)

	id := debugRequestID(t, buf.String())
	assert.Len(t, id, 8, "generated request ID has unexpected length")
}

func debugRequestID(t *testing.T, output string) string {
	t.Helper()
	require.True(t, strings.HasPrefix(output, "> ["), "unexpected debug output: %q", output)
	end := strings.Index(output, "]")
	require.Positive(t, end, "unexpected debug output: %q", output)
	return output[3:end]
}
