package http

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"net/textproto"
	"strings"
)

type (
	requestMetadataContextKey struct{}

	// RequestMetadata is an immutable snapshot of an inbound HTTP request.
	RequestMetadata struct {
		// Method is the request method.
		Method string
		// Path is the decoded request URL path.
		Path string
		// Host is the effective host after trusted-proxy processing.
		Host string
		// Scheme is the effective http or https scheme after trusted-proxy processing.
		Scheme string
		// ClientAddr is the effective client IP after trusted-proxy processing.
		ClientAddr string
		// PeerAddr is the direct network peer IP and is never forwarded.
		PeerAddr string
		// RequestID is the X-Request-Id header value.
		RequestID string
		// UserAgent is the request User-Agent value.
		UserAgent string
		// Origin is the request Origin value.
		Origin string
		// SecFetchSite is the request Sec-Fetch-Site value.
		SecFetchSite string

		headers http.Header
	}

	// RequestMetadataPolicy is an immutable trusted-proxy and retained-header policy.
	RequestMetadataPolicy struct {
		retainedHeaders []string
		trustedProxies  []netip.Prefix
	}
)

var (
	// ErrInvalidRequestMetadataHeader is returned when a retained header name is invalid.
	ErrInvalidRequestMetadataHeader = errors.New("loom http invalid request metadata header")
)

// NewRequestMetadataPolicy validates and returns an immutable metadata policy.
// retainedHeaders is an explicit allowlist; Authorization and Cookie are not
// retained unless named. trustedProxies contains direct peer CIDR ranges that
// may supply X-Forwarded-For, X-Forwarded-Host, and X-Forwarded-Proto.
func NewRequestMetadataPolicy(
	retainedHeaders []string,
	trustedProxies []netip.Prefix,
) (RequestMetadataPolicy, error) {
	headers := make([]string, 0, len(retainedHeaders))
	seen := make(map[string]struct{}, len(retainedHeaders))
	for _, name := range retainedHeaders {
		canonical := textproto.CanonicalMIMEHeaderKey(strings.TrimSpace(name))
		if canonical == "" {
			return RequestMetadataPolicy{}, ErrInvalidRequestMetadataHeader
		}
		if _, ok := seen[canonical]; ok {
			continue
		}
		seen[canonical] = struct{}{}
		headers = append(headers, canonical)
	}
	return RequestMetadataPolicy{
		retainedHeaders: headers,
		trustedProxies:  append([]netip.Prefix(nil), trustedProxies...),
	}, nil
}

// RequestMetadataMiddleware snapshots inbound request metadata before the next
// handler executes. Apply it with a generated server's Use method so endpoint
// decoders and security functions observe the same snapshot.
func RequestMetadataMiddleware(policy RequestMetadataPolicy) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			metadata := snapshotRequestMetadata(r, policy)
			ctx := context.WithValue(r.Context(), requestMetadataContextKey{}, metadata)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequestMetadataFromContext returns the inbound request metadata snapshot.
func RequestMetadataFromContext(ctx context.Context) (RequestMetadata, bool) {
	metadata, ok := ctx.Value(requestMetadataContextKey{}).(RequestMetadata)
	return metadata, ok
}

// HeaderValues returns a copy of the retained values for name.
func (m RequestMetadata) HeaderValues(name string) []string {
	return append([]string(nil), m.headers.Values(name)...)
}

// Headers returns a fresh clone of all explicitly retained headers.
func (m RequestMetadata) Headers() http.Header {
	return m.headers.Clone()
}

func snapshotRequestMetadata(r *http.Request, policy RequestMetadataPolicy) RequestMetadata {
	peerAddr, peerIP := splitRequestPeer(r.RemoteAddr)
	metadata := RequestMetadata{
		Method:       r.Method,
		Path:         r.URL.Path,
		Host:         r.Host,
		Scheme:       requestScheme(r),
		ClientAddr:   peerAddr,
		PeerAddr:     peerAddr,
		RequestID:    r.Header.Get("X-Request-Id"),
		UserAgent:    r.UserAgent(),
		Origin:       r.Header.Get("Origin"),
		SecFetchSite: r.Header.Get("Sec-Fetch-Site"),
		headers:      make(http.Header, len(policy.retainedHeaders)),
	}
	for _, name := range policy.retainedHeaders {
		if values := r.Header.Values(name); len(values) > 0 {
			metadata.headers[name] = append([]string(nil), values...)
		}
	}
	if !isTrustedProxy(peerIP, policy.trustedProxies) {
		return metadata
	}
	if client := forwardedClient(r.Header.Values("X-Forwarded-For"), peerIP, policy.trustedProxies); client.IsValid() {
		metadata.ClientAddr = client.String()
	}
	if host := lastForwardedValue(r.Header.Get("X-Forwarded-Host")); host != "" {
		metadata.Host = host
	}
	if scheme := lastForwardedValue(r.Header.Get("X-Forwarded-Proto")); scheme == "http" || scheme == "https" {
		metadata.Scheme = scheme
	}
	return metadata
}

func requestScheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

func splitRequestPeer(remoteAddr string) (string, netip.Addr) {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	addr, err := netip.ParseAddr(strings.Trim(host, "[]"))
	if err != nil {
		return host, netip.Addr{}
	}
	return host, addr.Unmap()
}

func isTrustedProxy(addr netip.Addr, proxies []netip.Prefix) bool {
	if !addr.IsValid() {
		return false
	}
	for _, proxy := range proxies {
		if proxy.Contains(addr) {
			return true
		}
	}
	return false
}

func forwardedClient(values []string, peer netip.Addr, proxies []netip.Prefix) netip.Addr {
	hops := make([]netip.Addr, 0, len(values)+1)
	for _, value := range values {
		for _, raw := range strings.Split(value, ",") {
			addr, err := netip.ParseAddr(strings.TrimSpace(raw))
			if err == nil {
				hops = append(hops, addr.Unmap())
			}
		}
	}
	hops = append(hops, peer)
	for i := len(hops) - 1; i >= 0; i-- {
		if !isTrustedProxy(hops[i], proxies) {
			return hops[i]
		}
	}
	return netip.Addr{}
}

func lastForwardedValue(value string) string {
	parts := strings.Split(value, ",")
	return strings.TrimSpace(parts[len(parts)-1])
}
