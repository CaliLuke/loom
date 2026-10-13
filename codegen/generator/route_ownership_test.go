package generator

import (
	"testing"

	"github.com/CaliLuke/loom/dsl"
)

const routeOwnershipCombinedHarness = `package routes_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	webserver "example.com/routes/gen/http/web/server"
	rpcserver "example.com/routes/gen/jsonrpc/rpc/server"
	rpc "example.com/routes/gen/rpc"
	web "example.com/routes/gen/web"
	loomhttp "github.com/CaliLuke/loom/http"
)

type recordingMux struct {
	loomhttp.Muxer
	registered map[string]int
}

// TestPhysicalMounts verifies registrations, dispatch and preflight ownership.
func TestPhysicalMounts(t *testing.T) {

	mux := &recordingMux{Muxer: loomhttp.NewMuxer(), registered: map[string]int{}}
	endpoint := func(context.Context, any) (any, error) {
		return "web", nil
	}
	filesystem := http.FS(fstest.MapFS{"assets/index.html": &fstest.MapFile{Data: []byte("index")}, "assets/a.txt": &fstest.MapFile{Data: []byte("asset")}})
	webserver.Mount(mux, webserver.New(&web.Endpoints{Ping: endpoint}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, filesystem))
	rpcserver.Mount(mux, rpcserver.New(&rpc.Endpoints{Call: func(context.Context, any) (any, error) {
		return "rpc", nil
	}}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil))
	want := []string{"GET /one/ping", "GET /two/ping", "OPTIONS /one/ping", "OPTIONS /two/ping", "GET /one/assets/", "GET /one/assets/{*path}", "GET /two/assets/", "GET /two/assets/{*path}", "POST /rpc", "GET /rpc", "OPTIONS /rpc"}
	if len(mux.registered) != len(want) {
		t.Fatalf("registrations=%v", mux.registered)
	}
	for _, key := range want {
		if mux.registered[key] != 1 {
			t.Errorf("%s registrations=%d", key, mux.registered[key])
		}
	}
	for _, prefix := range []string{"/one", "/two"} {

		for _, c := range []struct{ path, body string }{{"/ping", "web"}, {"/assets/", "index"}, {"/assets/a.txt", "asset"}} {

			w := httptest.NewRecorder()
			mux.ServeHTTP(w, httptest.NewRequest("GET", prefix+c.path, nil))
			if w.Code != 200 || !strings.Contains(w.Body.String(), c.body) {
				t.Errorf("%s: %d %s", prefix+c.path, w.Code, w.Body.String())
			}

		}

	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/rpc", strings.NewReader("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"call\"}"))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(w, req)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "rpc") {
		t.Errorf("RPC: %d %s", w.Code, w.Body.String())
	}
	for _, path := range []string{"/one/ping", "/two/ping", "/rpc"} {

		verb := "GET"
		if path == "/rpc" {
			verb = "POST"
		}
		w := httptest.NewRecorder()
		req := httptest.NewRequest("OPTIONS", path, nil)
		req.Header.Set("Origin", "https://example.com")
		req.Header.Set("Access-Control-Request-Method", verb)
		mux.ServeHTTP(w, req)
		if w.Code != 204 || w.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Errorf("preflight %s: %d %v", path, w.Code, w.Header())
		}

	}

}
func (m *recordingMux) Handle(method, path string, h http.HandlerFunc) {

	m.registered[method+" "+path]++
	m.Muxer.Handle(method, path, h)

}


`
const routeOwnershipDisjointHarness = `package routes_test

import (
	"context"
	first "example.com/routes/gen/first"
	firstserver "example.com/routes/gen/http/first/server"
	secondserver "example.com/routes/gen/http/second/server"
	second "example.com/routes/gen/second"
	loomhttp "github.com/CaliLuke/loom/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestIndependentMounts preserves separate server ownership of one path.
func TestIndependentMounts(t *testing.T) {

	a, b := loomhttp.NewMuxer(), loomhttp.NewMuxer()
	firstserver.Mount(a, firstserver.New(&first.Endpoints{Call: func(context.Context, any) (any, error) {
		return "first", nil
	}}, a, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	secondserver.Mount(b, secondserver.New(&second.Endpoints{Call: func(context.Context, any) (any, error) {
		return "second", nil
	}}, b, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	for _, c := range []struct {
		mux  loomhttp.Muxer
		want string
	}{{a, "first"}, {b, "second"}} {

		w := httptest.NewRecorder()
		c.mux.ServeHTTP(w, httptest.NewRequest("GET", "/same", nil))
		if w.Code != 200 || !strings.Contains(w.Body.String(), c.want) {
			t.Errorf("%s: %d %s", c.want, w.Code, w.Body.String())
		}

	}

}
`

// TestPhysicalRouteModules checks actual generated route ownership across mixed
// transports and independent server scopes using the existing module harness.
func TestPhysicalRouteModules(t *testing.T) {
	t.Run("combined", func(t *testing.T) {
		runDesignHarness(t, "example.com/routes", routeOwnershipCombinedDSL, routeOwnershipCombinedHarness)
	})
	t.Run("disjoint", func(t *testing.T) {
		runDesignHarness(t, "example.com/routes", routeOwnershipDisjointDSL, routeOwnershipDisjointHarness)
	})
}

func routeOwnershipCombinedDSL() {
	dsl.API("routes", func() {
	})
	dsl.Service("web", func() {
		dsl.HTTP(func() {
			dsl.Path("/one")
			dsl.Path("/two")
			dsl.CORS(func() {
				dsl.Origin("*")
			})
		})
		dsl.Method("ping", func() {
			dsl.Result(dsl.String)
			dsl.HTTP(func() {
				dsl.GET("/ping")
			})
		})
		dsl.Files("/assets/{*path}", "assets")
	})
	dsl.Service("rpc", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
			dsl.CORS(func() {
				dsl.Origin("*")
			})
		})
		dsl.Method("call", func() {
			dsl.Result(dsl.String)
			dsl.JSONRPC(func() {
			})
		})
		dsl.Method("events/stream", func() {
			dsl.StreamingResult(dsl.String)
			dsl.JSONRPC(func() {
				dsl.ServerSentEvents()
			})
		})
	})
}

func routeOwnershipDisjointDSL() {
	dsl.API("routes", func() {
		dsl.Server("first", func() {
			dsl.Services("first")
		})
		dsl.Server("second", func() {
			dsl.Services("second")
		})
	})
	for _, name := range []string{"first", "second"} {
		dsl.Service(name, func() {
			dsl.Meta("openapi:generate", "false")
			dsl.Method("call", func() {
				dsl.Result(dsl.String)
				dsl.HTTP(func() {
					dsl.GET("/same")
				})
			})
		})
	}
}
