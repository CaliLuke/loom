package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

func TestParameterRootCollectionsGenerated(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		rootMap := Type("RootMap", MapOf(String, String))
		rootList := Type("RootList", ArrayOf(String))
		Service("collections", func() {
			for _, name := range []string{"headerMap", "cookieMap", "pathList", "queryList", "pathMap", "queryMap", "headerList", "cookieList"} {
				Method(name, func() {
					var typ any = MapOf(String, String)
					switch name {
					case "cookieMap", "pathMap", "queryMap":
						typ = rootMap
					case "pathList", "headerList", "cookieList":
						typ = rootList
					case "queryList":
						typ = ArrayOf(String)
					}
					Payload(typ)
					Result(typ)
					HTTP(func() {
						switch name {
						case "pathList", "pathMap":
							POST("/" + name + "/{value}")
							Param("value")
						case "headerMap", "headerList":
							POST("/" + name)
							Header("value")
						case "cookieMap", "cookieList":
							POST("/" + name)
							Cookie("value", func() {
								Meta("openapi:explode", "false")
							})
						case "queryMap":
							POST("/" + name)
							Param("value", func() {
								Meta("openapi:style", "deepObject")
							})
						case "queryList":
							POST("/" + name)
							Param("value", func() {
								Meta("openapi:explode", "false")
							})
						}
						Response(StatusOK, func() {
							Header("output")
						})
					})
				})
			}
		})
	})
	directory := t.TempDir()
	renderHTTPModule(t, directory, "example.com/rootparams", root)
	require.NoError(t, os.WriteFile(filepath.Join(directory, "collections_test.go"), []byte(rootCollectionsHarness), 0o600))
	runGoCommand(t, directory, "mod", "tidy")
	runGoCommand(t, directory, "test", "./...")
}

const rootCollectionsHarness = `package rootparams_test

import (
 "context"
 "net/http"
 "net/http/httptest"
 "net/url"
 "testing"

 "github.com/stretchr/testify/require"
 loom "github.com/CaliLuke/loom/pkg"
 loomhttp "github.com/CaliLuke/loom/http"
 service "example.com/rootparams/gen/collections"
 client "example.com/rootparams/gen/http/collections/client"
 server "example.com/rootparams/gen/http/collections/server"
)

func TestRootCollections(t *testing.T) {
 endpoint := func(_ context.Context, payload any) (any, error) { return payload, nil }
 mux := loomhttp.NewMuxer()
 server.Mount(mux, server.New(&service.Endpoints{HeaderMap: endpoint, CookieMap: endpoint, PathList: endpoint, QueryList: endpoint, PathMap: endpoint, QueryMap: endpoint, HeaderList: endpoint, CookieList: endpoint}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
 host := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
  switch r.URL.Path {
  case "/headerMap": require.Equal(t, "a, 1, z, 2", r.Header.Get("value"))
  case "/cookieMap": require.Equal(t, "value=a,1,z,2", r.Header.Get("Cookie"))
  case "/pathList/one,two", "/pathMap/a,1,z,2": require.Empty(t, r.URL.RawQuery)
  case "/queryList": require.Equal(t, "value=one,two", r.URL.RawQuery)
  case "/headerList": require.Equal(t, "one, two", r.Header.Get("value"))
  case "/cookieList": require.Equal(t, "value=one,two", r.Header.Get("Cookie"))
  case "/queryMap": require.Equal(t, url.Values{"value[a]":{"1"}, "value[z]":{"2"}}, r.URL.Query())
  default: t.Errorf("unexpected path %s", r.URL.Path)
  }
  mux.ServeHTTP(w, r)
 }))
 defer host.Close()
 parsed, err := url.Parse(host.URL)
 require.NoError(t, err)
 transport := client.NewClient(parsed.Scheme, parsed.Host, host.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
 for _, tc := range []struct { call loom.Endpoint; value any }{
  {transport.HeaderMap(), map[string]string{"z":"2", "a":"1"}},
  {transport.CookieMap(), service.RootMap{"z":"2", "a":"1"}},
  {transport.PathList(), service.RootList{"one", "two"}},
  {transport.QueryList(), []string{"one", "two"}},
  {transport.PathMap(), service.RootMap{"z":"2", "a":"1"}},
  {transport.QueryMap(), service.RootMap{"z":"2", "a":"1"}},
  {transport.HeaderList(), service.RootList{"one", "two"}},
  {transport.CookieList(), service.RootList{"one", "two"}},
 } {
  got, err := tc.call(context.Background(), tc.value)
  require.NoError(t, err)
  require.Equal(t, tc.value, got)
 }
}
`
