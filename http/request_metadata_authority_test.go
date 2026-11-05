package http

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRequestMetadataForwardedAuthority(t *testing.T) {
	policy, err := NewRequestMetadataPolicy(nil, []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")})
	require.NoError(t, err)
	for _, tc := range []struct {
		name, peer           string
		hosts, protocols     []string
		wantHost, wantScheme string
	}{
		{"combined", "10.0.0.2:1234", []string{"client.example, api.example.test"}, []string{"http, https"}, "api.example.test", "https"},
		{"repeated", "10.0.0.2:1234", []string{"client.example", "api.example.test"}, []string{"http", "https"}, "api.example.test", "https"},
		{"mixed", "10.0.0.2:1234", []string{"client.example", "old.example, api.example.test"}, []string{"https", "http, https"}, "api.example.test", "https"},
		{"empty final line", "10.0.0.2:1234", []string{"client.example", ""}, []string{"https", ""}, "internal", "http"},
		{"empty final entry", "10.0.0.2:1234", []string{"client.example", "api.example.test, "}, []string{"https", "http, "}, "internal", "http"},
		{"invalid final scheme", "10.0.0.2:1234", []string{"client.example", "api.example.test"}, []string{"https", "unknown"}, "api.example.test", "http"},
		{"empty earlier line", "10.0.0.2:1234", []string{"client.example", "", "api.example.test"}, []string{"http", "", "https"}, "api.example.test", "https"},
		{"untrusted peer", "192.0.2.1:1234", []string{"client.example", "api.example.test"}, []string{"http", "https"}, "internal", "http"},
		{"absent", "10.0.0.2:1234", nil, nil, "internal", "http"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "http://internal/resource", nil)
			request.RemoteAddr = tc.peer
			for _, value := range tc.hosts {
				request.Header.Add("X-Forwarded-Host", value)
			}
			for _, value := range tc.protocols {
				request.Header.Add("X-Forwarded-Proto", value)
			}
			headers := request.Header.Clone()
			metadata := captureRequestMetadata(t, policy, request)
			require.Equal(t, tc.wantHost, metadata.Host)
			require.Equal(t, tc.wantScheme, metadata.Scheme)
			require.Equal(t, headers, request.Header)
		})
	}
}

func TestRequestMetadataForwardedAuthorityWire(t *testing.T) {
	policy, err := NewRequestMetadataPolicy(nil, []netip.Prefix{
		netip.MustParsePrefix("127.0.0.0/8"), netip.MustParsePrefix("::1/128"),
	})
	require.NoError(t, err)
	received := make(chan RequestMetadata, 1)
	server := httptest.NewServer(RequestMetadataMiddleware(policy)(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		metadata, _ := RequestMetadataFromContext(request.Context())
		received <- metadata
		w.WriteHeader(http.StatusNoContent)
	})))
	t.Cleanup(server.Close)
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
	require.NoError(t, err)
	// A proxy may append its values after an existing client-supplied field line.
	request.Header.Add("X-Forwarded-Host", "client.example")
	request.Header.Add("X-Forwarded-Host", "api.example.test")
	request.Header.Add("X-Forwarded-Proto", "http")
	request.Header.Add("X-Forwarded-Proto", "https")
	response, err := server.Client().Do(request)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	require.Equal(t, http.StatusNoContent, response.StatusCode)
	metadata := <-received
	require.Equal(t, "api.example.test", metadata.Host)
	require.Equal(t, "https", metadata.Scheme)
}
