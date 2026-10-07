package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/testutil"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

const responseSelectionHarness = `package selection_test

import (
 "context"
 "net/http"
 "net/http/httptest"
 "testing"

 client "example.com/selection/gen/http/selection/client"
 server "example.com/selection/gen/http/selection/server"
 service "example.com/selection/gen/selection"
 loomhttp "github.com/CaliLuke/loom/http"
)

func TestSelection(t *testing.T) {
 for _, method := range []struct {
  name string
  encode func(context.Context, http.ResponseWriter, any) error
  decode func(*http.Response) (any, error)
 }{
  {"first", server.EncodeDefaultFirstResponse(loomhttp.ResponseEncoder), client.DecodeDefaultFirstResponse(loomhttp.ResponseDecoder, false)},
  {"middle", server.EncodeDefaultMiddleResponse(loomhttp.ResponseEncoder), client.DecodeDefaultMiddleResponse(loomhttp.ResponseDecoder, false)},
  {"last", server.EncodeDefaultLastResponse(loomhttp.ResponseEncoder), client.DecodeDefaultLastResponse(loomhttp.ResponseDecoder, false)},
 } {
  for _, tc := range []struct {
   name, first, second string
   status int
  }{
   {"both", "yes", "yes", 201},
   {"first only", "yes", "no", 201},
   {"second only", "no", "yes", 202},
   {"neither", "no", "no", 200},
  } {
   t.Run(method.name+"/"+tc.name, func(t *testing.T) {
    rec := httptest.NewRecorder()
    value := &service.SelectionResult{First:tc.first, Second:tc.second, Message:"message", Trace:"trace"}
    if err := method.encode(context.Background(), rec, value); err != nil {
     t.Fatal(err)
    }
    if rec.Code != tc.status {
     t.Errorf("status = %d, want %d", rec.Code, tc.status)
    }
    if got := rec.Header().Get("X-Trace"); got != "trace" {
     t.Errorf("trace = %q", got)
    }
    if got := rec.Body.String(); got != "\"message\"\n" {
     t.Errorf("body = %q", got)
    }
    decoded, err := method.decode(rec.Result())
    if err != nil {
     t.Fatal(err)
    }
    actual := decoded.(*service.SelectionResult)
    if actual.Message != "message" || actual.Trace != "trace" {
     t.Errorf("decoded result = %#v", actual)
    }
   })
  }
 }
}
`

func TestResponseSelectionEncoders(t *testing.T) {
	root := RunHTTPDSL(t, testdata.ResponseSelectionOrderDSL)
	files := ServerFiles("example.com/selection/gen", CreateHTTPServices(root))
	sections := files[1].Section("response-encoder")
	require.Len(t, sections, 3)
	testutil.AssertGo(t, "testdata/golden/response_selection.go.golden", codegen.SectionsCode(t, sections))
}

func TestResponseSelectionGenerated(t *testing.T) {
	root := RunHTTPDSL(t, testdata.ResponseSelectionOrderDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/selection", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "selection_test.go"), []byte(responseSelectionHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "-count=1", "./...")
}
