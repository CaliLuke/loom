package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

const unionCollectionsHarness = `package choices_test

import (
	"context"
	"encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	svc "example.com/choices/gen/choices"
	client "example.com/choices/gen/http/choices/client"
	server "example.com/choices/gen/http/choices/server"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/require"
)

func TestUnionCollectionsRoundTrip(t *testing.T) {
	var calls atomic.Int32
	echo := func(_ context.Context, value any) (any, error) {
		calls.Add(1)
		return value, nil
	}
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(&svc.Endpoints{Anonymous: echo, Named: echo, Nested: echo, Mapped: echo}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	host := httptest.NewServer(mux)
	defer host.Close()
	address, err := url.Parse(host.URL)
	require.NoError(t, err)
	c := client.NewClient(address.Scheme, address.Host, http.DefaultClient, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	for _, tc := range []struct {
		name, body string
		payload    any
		endpoint   loom.Endpoint
	}{
		{"anonymous", "[{\"type\":\"Float32\",\"value\":1.25},{\"type\":\"String\",\"value\":\"text\"}]", new([]svc.Float32OrString), c.Anonymous()},
		{"named", "[{\"type\":\"String\",\"value\":\"named\"}]", new(svc.Choices), c.Named()},
		{"nested", "[[{\"type\":\"Float32\",\"value\":1.25}]]", new([][]svc.Float32OrString), c.Nested()},
		{"mapped", "{\"key\":{\"type\":\"String\",\"value\":\"mapped\"}}", new(map[string]svc.Float32OrString), c.Mapped()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, json.Unmarshal([]byte(tc.body), tc.payload))
			var payload any
			switch value := tc.payload.(type) {
			case *[]svc.Float32OrString:
				payload = *value
			case *svc.Choices:
				payload = *value
			case *[][]svc.Float32OrString:
				payload = *value
			case *map[string]svc.Float32OrString:
				payload = *value
			}
			before := calls.Load()
			result, err := tc.endpoint(context.Background(), payload)
			require.NoError(t, err)
			require.Equal(t, before+1, calls.Load())
			encoded, err := json.Marshal(result)
			require.NoError(t, err)
			require.JSONEq(t, tc.body, string(encoded))
		})
	}
	// Null elements and unknown alternatives must not reach the service.
	for _, body := range []string{"[null]", "[{\"type\":\"Unknown\",\"value\":1}]"} {
		before := calls.Load()
		response, err := http.Post(host.URL+"/anonymous", "application/json", strings.NewReader(body))
		require.NoError(t, err)
		data, err := io.ReadAll(response.Body)
		require.NoError(t, err)
		require.NoError(t, response.Body.Close())
		require.Equal(t, http.StatusBadRequest, response.StatusCode, string(data))
		require.Equal(t, before, calls.Load())
	}
}
`

func TestUnionCollectionsGeneratedHTTP(t *testing.T) {
	root := RunHTTPDSL(t, testdata.UnionCollectionsDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/choices", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "union_test.go"), []byte(unionCollectionsHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-race", "./...")
}
