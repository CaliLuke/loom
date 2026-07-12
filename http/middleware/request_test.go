package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	httpm "github.com/CaliLuke/loom/http/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPopulateRequestContext(t *testing.T) {
	req := httptest.NewRequest("POST", "/foo/bar?baz=1", nil)
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	req.Header.Set("X-Real-Ip", "5.6.7.8")
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("Referer", "https://example.com/origin")
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("X-Request-Id", "req-123")
	req.Header.Set("X-Csrf-Token", "csrf-456")
	req.Header.Set("Accept", "application/json")

	var ctx context.Context
	handler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		ctx = r.Context()
	})

	httpm.PopulateRequestContext()(handler).ServeHTTP(httptest.NewRecorder(), req)

	require.NotNil(t, ctx, "wrapped handler was not invoked")
	cases := []struct {
		name string
		key  any
		want string
	}{
		{name: "method", key: httpm.RequestMethodKey, want: "POST"},
		{name: "request URI", key: httpm.RequestURIKey, want: "/foo/bar?baz=1"},
		{name: "path", key: httpm.RequestPathKey, want: "/foo/bar"},
		{name: "proto", key: httpm.RequestProtoKey, want: "HTTP/1.1"},
		{name: "host", key: httpm.RequestHostKey, want: "example.com"},
		{name: "remote addr", key: httpm.RequestRemoteAddrKey, want: "192.0.2.1:1234"},
		{name: "x-forwarded-for", key: httpm.RequestXForwardedForKey, want: "1.2.3.4"},
		{name: "x-real-ip", key: httpm.RequestXRealIPKey, want: "5.6.7.8"},
		{name: "x-forwarded-proto", key: httpm.RequestXForwardedProtoKey, want: "https"},
		{name: "authorization", key: httpm.RequestAuthorizationKey, want: "Bearer token"},
		{name: "referer", key: httpm.RequestRefererKey, want: "https://example.com/origin"},
		{name: "user agent", key: httpm.RequestUserAgentKey, want: "test-agent"},
		{name: "x-request-id", key: httpm.RequestXRequestIDKey, want: "req-123"},
		{name: "x-csrf-token", key: httpm.RequestXCSRFTokenKey, want: "csrf-456"},
		{name: "accept", key: httpm.RequestAcceptKey, want: "application/json"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, ctx.Value(c.key))
		})
	}
}

func TestPopulateRequestContextMissingHeaders(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)

	var ctx context.Context
	handler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		ctx = r.Context()
	})

	httpm.PopulateRequestContext()(handler).ServeHTTP(httptest.NewRecorder(), req)

	require.NotNil(t, ctx, "wrapped handler was not invoked")
	// Absent headers are stored as empty strings, not left unset.
	assert.Equal(t, "", ctx.Value(httpm.RequestXForwardedForKey))
	assert.Equal(t, "", ctx.Value(httpm.RequestAuthorizationKey))
	assert.Equal(t, "", ctx.Value(httpm.RequestXRequestIDKey))
}
