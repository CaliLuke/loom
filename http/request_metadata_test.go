package http

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRequestMetadataMiddleware(t *testing.T) {
	trusted := netip.MustParsePrefix("10.0.0.0/8")
	policy, err := NewRequestMetadataPolicy([]string{"X-Tenant", "X-Multi"}, []netip.Prefix{trusted})
	require.NoError(t, err)

	t.Run("direct TLS request", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "https://api.example.test/items", nil)
		r.TLS = &tls.ConnectionState{}
		r.RemoteAddr = "192.0.2.1:1234"
		r.Header.Set("Origin", "https://app.example.test")
		r.Header.Set("Sec-Fetch-Site", "same-origin")
		metadata := captureRequestMetadata(t, policy, r)
		require.Equal(t, "https", metadata.Scheme)
		require.Equal(t, "api.example.test", metadata.Host)
		require.Equal(t, "192.0.2.1", metadata.ClientAddr)
		require.Equal(t, metadata.PeerAddr, metadata.ClientAddr)
		require.Equal(t, "https://app.example.test", metadata.Origin)
		require.Equal(t, "same-origin", metadata.SecFetchSite)
	})

	t.Run("trusted proxy selects first untrusted hop from the right", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "http://internal/items", nil)
		r.RemoteAddr = "10.0.0.2:4321"
		r.Header.Set("X-Forwarded-For", "198.51.100.7, 10.0.0.1")
		r.Header.Set("X-Forwarded-Host", "old.example, api.example.test")
		r.Header.Set("X-Forwarded-Proto", "http, https")
		metadata := captureRequestMetadata(t, policy, r)
		require.Equal(t, "198.51.100.7", metadata.ClientAddr)
		require.Equal(t, "10.0.0.2", metadata.PeerAddr)
		require.Equal(t, "api.example.test", metadata.Host)
		require.Equal(t, "https", metadata.Scheme)
	})

	t.Run("untrusted peer cannot spoof forwarding", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "http://internal/items", nil)
		r.RemoteAddr = "192.0.2.2:4321"
		r.Header.Set("X-Forwarded-For", "198.51.100.7")
		r.Header.Set("X-Forwarded-Host", "api.example.test")
		r.Header.Set("X-Forwarded-Proto", "https")
		metadata := captureRequestMetadata(t, policy, r)
		require.Equal(t, "192.0.2.2", metadata.ClientAddr)
		require.Equal(t, "internal", metadata.Host)
		require.Equal(t, "http", metadata.Scheme)
	})

	t.Run("retained headers are cloned and sensitive headers default absent", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "http://internal/items", nil)
		r.Header.Add("X-Multi", "one")
		r.Header.Add("X-Multi", "two")
		r.Header.Set("Authorization", "Bearer secret")
		r.Header.Set("Cookie", "session=secret")
		metadata := captureRequestMetadata(t, policy, r)

		values := metadata.HeaderValues("X-Multi")
		require.Equal(t, []string{"one", "two"}, values)
		values[0] = "changed"
		headers := metadata.Headers()
		headers.Set("X-Multi", "changed again")
		require.Equal(t, []string{"one", "two"}, metadata.HeaderValues("X-Multi"))
		require.Empty(t, metadata.HeaderValues("Authorization"))
		require.Empty(t, metadata.HeaderValues("Cookie"))
		require.Equal(t, []string{"one", "two"}, r.Header.Values("X-Multi"))
	})
}

func TestEffectiveClientAddress(t *testing.T) {
	t.Run("ignores forwarded headers without a metadata policy", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "https://service.example.test/tasks", nil)
		r.RemoteAddr = "192.0.2.10:4321"
		r.Header.Set("X-Forwarded-For", "198.51.100.7")

		require.Equal(t, "192.0.2.10", EffectiveClientAddress(r))
	})

	t.Run("uses trusted proxy metadata", func(t *testing.T) {
		policy, err := NewRequestMetadataPolicy(nil, []netip.Prefix{
			netip.MustParsePrefix("10.0.0.0/8"),
		})
		require.NoError(t, err)

		var got string
		handler := RequestMetadataMiddleware(policy)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			got = EffectiveClientAddress(r)
		}))
		r := httptest.NewRequest(http.MethodGet, "https://service.example.test/tasks", nil)
		r.RemoteAddr = "10.0.0.2:4321"
		r.Header.Set("X-Forwarded-For", "198.51.100.7, 10.0.0.1")
		handler.ServeHTTP(httptest.NewRecorder(), r)

		require.Equal(t, "198.51.100.7", got)
	})
}

func TestRequestMetadataSensitiveHeaderOptIn(t *testing.T) {
	policy, err := NewRequestMetadataPolicy([]string{"Authorization", "Cookie"}, nil)
	require.NoError(t, err)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer secret")
	r.Header.Set("Cookie", "session=secret")
	metadata := captureRequestMetadata(t, policy, r)
	require.Equal(t, []string{"Bearer secret"}, metadata.HeaderValues("Authorization"))
	require.Equal(t, []string{"session=secret"}, metadata.HeaderValues("Cookie"))
}

func TestRequestMetadataAbsent(t *testing.T) {
	_, ok := RequestMetadataFromContext(httptest.NewRequest(http.MethodGet, "/", nil).Context())
	require.False(t, ok)
}

func captureRequestMetadata(t *testing.T, policy RequestMetadataPolicy, r *http.Request) RequestMetadata {
	t.Helper()
	var metadata RequestMetadata
	handler := RequestMetadataMiddleware(policy)(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		var ok bool
		metadata, ok = RequestMetadataFromContext(request.Context())
		require.True(t, ok)
	}))
	handler.ServeHTTP(httptest.NewRecorder(), r)
	return metadata
}
