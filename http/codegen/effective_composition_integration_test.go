package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestEffectiveCompositionGeneratedHTTP(t *testing.T) {
	root := RunHTTPDSL(t, testdata.EffectiveCompositionDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/composition", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "composition_test.go"), []byte(effectiveCompositionHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}

func TestRecursiveEffectiveCompositionGeneratedHTTP(t *testing.T) {
	root := RunHTTPDSL(t, testdata.RecursiveEffectiveCompositionDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/recursivecomposition", root)
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "build", "./...")
}

const effectiveCompositionHarness = `package composition_test

import (
 "context"
 "encoding/json/v2"
 "io"
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"

 "github.com/stretchr/testify/require"
 "example.com/composition/gen/composition"
 server "example.com/composition/gen/http/composition/server"
 loomhttp "github.com/CaliLuke/loom/http"
)

func TestEffectiveComposition(t *testing.T) {
 calls := 0
 endpoints := &composition.Endpoints{Read: func(_ context.Context, value any) (any, error) {
  calls++
  return value, nil
 }}
 mux := loomhttp.NewMuxer()
 server.Mount(mux, server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
 for _, test := range []struct {
  wire string
  status int
  code string
 }{
  {"{\"count\":-1,\"label\":\"ok\"}", 400, "invalid_range"},
  {"{\"count\":101,\"label\":\"ok\"}", 400, "invalid_range"},
  {"{\"label\":\"ok\"}", 400, "missing_field"},
  {"{\"count\":50}", 400, "missing_field"},
  {"{\"count\":50,\"label\":\"ok\"}", 200, ""},
 } {
  t.Run(test.wire, func(t *testing.T) {
   before := calls
   request := httptest.NewRequest(http.MethodPost, "/read", strings.NewReader(test.wire))
   request.Header.Set("Content-Type", "application/json")
   response := httptest.NewRecorder()
   mux.ServeHTTP(response, request)
   require.Equal(t, test.status, response.Code, response.Body.String())
   if test.code != "" {
    require.Contains(t, response.Body.String(), "\"code\":\""+test.code+"\"")
    require.Equal(t, before, calls)
    return
   }
   require.Equal(t, before+1, calls)
   data, err := io.ReadAll(response.Body)
   require.NoError(t, err)
   var decoded map[string]any
   require.NoError(t, json.Unmarshal(data, &decoded))
   require.Equal(t, float64(5), decoded["retries"])
   require.Equal(t, float64(50), decoded["count"])
  })
 }
}
`
