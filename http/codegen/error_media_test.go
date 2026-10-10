package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestErrorMediaGenerated(t *testing.T) {
	root := RunHTTPDSL(t, testdata.ErrorMediaDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/projection", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "error_media_test.go"), []byte(errorMediaHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "-count=1", "./...")
}

const errorMediaHarness = `package projection_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	client "example.com/projection/gen/http/projection/client"
	server "example.com/projection/gen/http/projection/server"
	service "example.com/projection/gen/projection"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/require"
)

func TestErrorMedia(t *testing.T) {
	for _, method := range []string{"mixed", "reversed", "same_media"} {
		for _, tc := range []struct {
			name, media, wire string
			value             error
		}{
			{"html", "text/html; charset=utf-8", "<html>failed</html>", &service.HTMLError{Kind: "html", Body: "<html>failed</html>"}},
			{"object", "application/json", "{\"message\":\"failed\"}\n", &service.ObjectError{Kind: "object", Body: &service.ErrorDetails{Message: "failed"}}},
		} {
			if method == "same_media" && tc.name == "html" {
 tc.media = "application/json"
 tc.wire = "\"<html>failed</html>\"\n"
 }
			t.Run(method+"/"+tc.name, func(t *testing.T) {
				endpoint := func(context.Context, any) (any, error) {
 return nil, tc.value
 }
				mux := loomhttp.NewMuxer()
				generated := server.New(&service.Endpoints{Mixed: endpoint, Reversed: endpoint, SameMedia: endpoint}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, func(_ context.Context, _ http.ResponseWriter, err error) {
 t.Errorf("encoding: %v", err)
 }, nil)
				server.Mount(mux, generated)
				host := httptest.NewServer(mux)
				t.Cleanup(host.Close)
				response, err := host.Client().Get(host.URL + "/" + method)
				require.NoError(t, err)
				body, err := io.ReadAll(response.Body)
				require.NoError(t, err)
				require.NoError(t, response.Body.Close())
				require.Equal(t, 500, response.StatusCode)
				require.Equal(t, tc.media, response.Header.Get("Content-Type"))
				require.Equal(t, tc.name, response.Header.Get("Loom-Error"))
				require.Equal(t, tc.name, response.Header.Get("X-Kind"))
				require.Equal(t, tc.wire, string(body))
				checked := 0
				for _, contract := range server.ResponseContractCases() {
					if contract.ID != "projection."+method+".error."+tc.name+".500" {
						continue
					}
					checked++
					require.Equal(t, []string{tc.media}, contract.ContentTypes)
					response.Body = io.NopCloser(bytes.NewReader(body))
					require.NoError(t, loomhttp.ValidateResponseContract(response, contract))
					require.NoError(t, response.Body.Close())
				}
				require.Equal(t, 1, checked, "each method retains its individual error contracts")
				address, err := url.Parse(host.URL)
				require.NoError(t, err)
				cli := client.NewClient(address.Scheme, address.Host, host.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
				call := map[string]loom.Endpoint{"mixed": cli.Mixed(), "reversed": cli.Reversed(), "same_media": cli.SameMedia()}[method]
				result, err := call(t.Context(), nil)
				require.Nil(t, result)
				require.Equal(t, tc.value, err)
			})
		}
	}
}
`
