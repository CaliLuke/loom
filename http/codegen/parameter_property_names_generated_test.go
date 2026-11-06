package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestParameterPropertyNamesGenerated(t *testing.T) {
	root := RunHTTPDSL(t, testdata.ParameterPropertyNamesDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/propertynames", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "properties_test.go"), []byte(parameterPropertyNamesHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}

const parameterPropertyNamesHarness = `package propertynames_test

import (
 "context"
 "io"
 "net/http"
 "net/http/httptest"
 "net/url"
 "strings"
 "testing"

 "github.com/stretchr/testify/require"
 loomhttp "github.com/CaliLuke/loom/http"
 service "example.com/propertynames/gen/propertynames"
 client "example.com/propertynames/gen/http/propertynames/client"
 server "example.com/propertynames/gen/http/propertynames/server"
)

func TestNames(t *testing.T) {
 other := 5
 record := &service.ParameterRecord{Count: 3, Other: &other}
 endpoints := &service.Endpoints{
  Exchange: func(_ context.Context, value any) (any, error) {
   require.Equal(t, &service.ExchangePayload{Filter: record, Data: record}, value)
   return &service.ExchangeResult{Metadata: record, Data: record}, nil
  },
  Hidden: func(_ context.Context, value any) (any, error) {
   require.Equal(t, &service.HiddenPayload{Filter: &service.TextOnlyRecord{Count: 3, Hidden: &other}}, value)
   return nil, nil
  },
  Form: func(_ context.Context, value any) (any, error) {
   require.Equal(t, &service.FormPayload{Styled: record, Ordinary: record}, value)
   return nil, nil
  },
 }
 mux := loomhttp.NewMuxer()
 server.Mount(mux, server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
 host := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
  switch r.URL.Path {
  case "/exchange":
   require.Equal(t, "filter%5Bcount%5D=3&filter%5Bother%5D=5", r.URL.RawQuery)
   body, err := io.ReadAll(r.Body)
   require.NoError(t, err)
   require.NoError(t, r.Body.Close())
   require.JSONEq(t, "{\"wire\":3,\"alternate\":5}", string(body))
   r.Body = io.NopCloser(strings.NewReader(string(body)))
  case "/hidden":
   require.Equal(t, "filter%5Bcount%5D=3&filter%5Bhidden%5D=5", r.URL.RawQuery)
  case "/form":
   body, err := io.ReadAll(r.Body)
   require.NoError(t, err)
   require.NoError(t, r.Body.Close())
   r.Body = io.NopCloser(strings.NewReader(string(body)))
   fields, err := url.ParseQuery(string(body))
   require.NoError(t, err)
   require.Equal(t, "3", fields.Get("styled[count]"))
   require.Equal(t, "5", fields.Get("styled[other]"))
   require.JSONEq(t, "{\"wire\":3,\"alternate\":5}", fields.Get("ordinary"))
  }
  recorder := httptest.NewRecorder()
  mux.ServeHTTP(recorder, r)
  if r.URL.Path == "/exchange" { require.Equal(t, "count, 3, other, 5", recorder.Header().Get("X-Record")) }
  for name, values := range recorder.Header() { w.Header()[name] = values }
  w.WriteHeader(recorder.Code)
  _, err := io.Copy(w, recorder.Body)
  require.NoError(t, err)
 }))
 defer host.Close()
 parsed, err := url.Parse(host.URL)
 require.NoError(t, err)
 transport := client.NewClient(parsed.Scheme, parsed.Host, host.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
 got, err := transport.Exchange()(context.Background(), &service.ExchangePayload{Filter: record, Data: record})
 require.NoError(t, err)
 require.Equal(t, &service.ExchangeResult{Metadata: record, Data: record}, got)
 _, err = transport.Hidden()(context.Background(), &service.HiddenPayload{Filter: &service.TextOnlyRecord{Count: 3, Hidden: &other}})
 require.NoError(t, err)
 _, err = transport.Form()(context.Background(), &service.FormPayload{Styled: record, Ordinary: record})
 require.NoError(t, err)
}
`
