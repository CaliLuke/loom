package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

const pathFieldNamesHarness = `package client

import (
 "context"
 "net/http"
 "net/http/httptest"
 "net/url"
 "testing"

 svc "example.com/pathfields/gen/service_path_custom_name"
 server "example.com/pathfields/gen/http/service_path_custom_name/server"
 loomhttp "github.com/CaliLuke/loom/http"
)

func TestPathFieldOverride(t *testing.T) {
 c := &Client{scheme: "https", host: "example.com"}
 for _, value := range []string{"first", "second value", "third/value"} {
  t.Run(value, func(t *testing.T) {
   p := &svc.MethodPathCustomNamePayload{Path: value}
   req, err := c.BuildMethodPathCustomNameRequest(context.Background(), p)
   if err != nil {
    t.Fatal(err)
   }
   if got, want := req.URL.String(), "https://example.com/" + url.PathEscape(value); got != want {
    t.Errorf("URL = %q, want %q", got, want)
   }
   mux := loomhttp.NewMuxer()
   called := false
   mux.Handle("GET", "/{p}", func(w http.ResponseWriter, r *http.Request) {
    called = true
    got, err := server.DecodeMethodPathCustomNameRequest(mux, nil)(r)
    if err != nil {
     t.Fatal(err)
    }
    if got.Path != value {
     t.Errorf("decoded path = %q, want %q", got.Path, value)
    }
   })
   mux.ServeHTTP(httptest.NewRecorder(), req)
   if !called {
    t.Error("decoder was not called")
   }
   got, err := BuildMethodPathCustomNamePayload(value)
   if err != nil {
    t.Fatal(err)
   }
   if got.Path != value {
    t.Errorf("CLI path = %q, want %q", got.Path, value)
   }
  })
 }
}
`

func TestPathFieldNames(t *testing.T) {
	data := CreateHTTPServices(RunHTTPDSL(t, testdata.PayloadPathCustomNameDSL)).Get("ServicePathCustomName")
	endpoint := data.Endpoints[0]
	require.Equal(t, "Path", endpoint.Routes[0].PathInit.ClientArgs[0].FieldName)
	require.Contains(t, endpoint.RequestInit.ClientCode, "p.Path")
}

func TestPathFieldNamesGeneratedModule(t *testing.T) {
	root := RunHTTPDSL(t, testdata.PayloadPathCustomNameDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/pathfields", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "gen", "http", "service_path_custom_name", "client", "fields_test.go"), []byte(pathFieldNamesHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-count=1", "./...")
}
