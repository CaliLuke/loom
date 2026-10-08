package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

func TestFormRootMapEmptyKeyIntegration(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		Service("formmap", func() {
			Method("submit", func() {
				Payload(MapOf(String, String))
				HTTP(func() {
					POST("/form")
					FormRequest()
				})
			})
		})
	})
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/formmap", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "form_map_test.go"), []byte(formRootMapHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-race", "-count=1", "./...")
}

const formRootMapHarness = `package formmap

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	svc "example.com/formmap/gen/formmap"
	client "example.com/formmap/gen/http/formmap/client"
	server "example.com/formmap/gen/http/formmap/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

type service struct {
	seen chan map[string]string
}

func (s *service) Submit(ctx context.Context, p map[string]string) error {
	s.seen <- p
	return nil
}

type wireClient struct {
	t        *testing.T
	client   *http.Client
	captured string
}

func (c *wireClient) Do(req *http.Request) (*http.Response, error) {
	c.t.Helper()
	require.NotNil(c.t, req.GetBody)
	body, err := req.GetBody()
	require.NoError(c.t, err)
	wire, err := io.ReadAll(body)
	require.NoError(c.t, err)
	require.NoError(c.t, body.Close())
	c.captured = string(wire)
	return c.client.Do(req)
}

func TestEmptyMapKey(t *testing.T) {
	s := &service{seen: make(chan map[string]string, 1)}
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(svc.NewEndpoints(s), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	host := httptest.NewServer(mux)
	defer host.Close()
	u, err := url.Parse(host.URL)
	require.NoError(t, err)
	transport := &wireClient{t: t, client: host.Client()}
	c := client.NewClient(u.Scheme, u.Host, transport, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	for _, value := range []string{"empty", ""} {
		input := map[string]string{"": value, "normal": "value"}
		_, err = c.Submit()(context.Background(), input)
		require.NoError(t, err)
		wire := transport.captured
		require.Equal(t, url.Values{"": {value}, "normal": {"value"}}.Encode(), wire)
		require.Equal(t, input, <-s.seen)
		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, host.URL+"/form", strings.NewReader(wire))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		response, err := host.Client().Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, response.StatusCode)
		require.NoError(t, response.Body.Close())
		require.Equal(t, input, <-s.seen)
	}
}
`
