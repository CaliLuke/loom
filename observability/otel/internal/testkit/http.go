package testkit

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	loomhttp "github.com/CaliLuke/loom/http"
)

type (
	// HTTPFixture wraps a Loom mux in an httptest server.
	HTTPFixture struct {
		Mux    loomhttp.ResolverMuxer
		Server *httptest.Server
	}
)

// NewHTTPFixture creates a Loom mux plus httptest server.
func NewHTTPFixture(tb testing.TB) *HTTPFixture {
	tb.Helper()
	mux := loomhttp.NewMuxer()
	server := httptest.NewServer(mux)
	tb.Cleanup(server.Close)
	return &HTTPFixture{Mux: mux, Server: server}
}

// Request performs one HTTP request against the test server.
func (f *HTTPFixture) Request(ctx context.Context, method string, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, f.Server.URL+path, body)
	if err != nil {
		return nil, err
	}
	return f.Server.Client().Do(req)
}
