package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/jsonrpc/codegen/testdata"
)

const middlewareDispatchHarness = `package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	loomhttp "github.com/CaliLuke/loom/http"
)

type route struct{ method, path string }
type recordingMux struct {
	loomhttp.Muxer
	routes []route
}
type dispatchKey struct{}

func (m *recordingMux) Handle(method, path string, h http.HandlerFunc) {
	m.routes = append(m.routes, route{method, path})
	m.Muxer.Handle(method, path, h)
}
func TestMiddlewareDispatch(t *testing.T) {
	for _, mountAt := range []int{0, 1, 2} {
		for _, reject := range []bool{false, true} {
			calls := 0
			var order []int
			// The terminal handler isolates dispatch ownership from protocol framing.
			s := &Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Context().Value(dispatchKey{}) != 1 {
					t.Error("lost middleware context")
				}
				w.WriteHeader(http.StatusAccepted)
			})}
			mux := &recordingMux{Muxer: loomhttp.NewMuxer()}
			for step := 0; step <= 2; step++ {
				if step == mountAt {
					Mount(mux, s)
				}
				if step == 2 {
					break
				}
				id := step + 1
				s.Use(func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						order = append(order, id)
						if id == 2 {
							r = r.WithContext(context.WithValue(r.Context(), dispatchKey{}, 2))
						}
						if id == 1 {
							if r.Context().Value(dispatchKey{}) != 2 {
								t.Error("lost outer context")
							}
							if reject {
								w.WriteHeader(http.StatusForbidden)
								return
							}
							r = r.WithContext(context.WithValue(r.Context(), dispatchKey{}, 1))
						}
						next.ServeHTTP(w, r)
					})
				})
			}
			run := func(h http.Handler, method, path string) {
				before := calls
				order = nil
				w := httptest.NewRecorder()
				h.ServeHTTP(w, httptest.NewRequest(method, path, nil))
				wantStatus, wantCalls := http.StatusAccepted, before+1
				if reject {
					wantStatus, wantCalls = http.StatusForbidden, before
				}
				if w.Code != wantStatus || calls != wantCalls || !reflect.DeepEqual(order, []int{2, 1}) {
					t.Errorf("mountAt=%d reject=%v %s %s: status=%d calls=%d order=%v", mountAt, reject, method, path, w.Code, calls, order)
				}
			}
			for _, route := range mux.routes {
				if route.method != http.MethodOptions {
					run(mux, route.method, route.path)
				}
			}
			run(s, http.MethodPost, "/direct")
		}
	}
}

`

const middlewareEndpointHarness = `package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	svc "example.com/dispatch/gen/plain"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
)

func TestMiddlewareProtectsEndpoint(t *testing.T) {
	calls := 0
	endpoints := &svc.Endpoints{Call: loom.Endpoint(func(ctx context.Context, _ any) (any, error) {
		calls++
		if ctx.Value(dispatchKey{}) != 42 {
			t.Error("lost context")
		}
		return "ok", nil
	})}
	mux := loomhttp.NewMuxer()
	s := New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, func(_ context.Context, _ http.ResponseWriter, err error) { t.Error(err) })
	Mount(mux, s)
	s.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Reject") == "yes" {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), dispatchKey{}, 42)))
		})
	})
	for _, reject := range []bool{false, true} {
		before := calls
		r := httptest.NewRequest("POST", "/plain", strings.NewReader(` + "`" + `{"jsonrpc":"2.0","id":1,"method":"call"}` + "`" + `))
		r.Header.Set("Content-Type", "application/json")
		if reject {
			r.Header.Set("Reject", "yes")
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		wantStatus, wantCalls := http.StatusOK, before+1
		if reject {
			wantStatus, wantCalls = http.StatusForbidden, before
		}
		if w.Code != wantStatus || calls != wantCalls {
			t.Errorf("reject=%v status=%d calls=%d", reject, w.Code, calls)
		}
	}
}
`

func TestMiddlewareDispatchGenerated(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.MiddlewareDispatchDSL)
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/dispatch", root)
	for _, service := range root.Services {
		path := filepath.Join(dir, "gen", "jsonrpc", service.Name, "server", "dispatch_test.go")
		require.NoError(t, os.WriteFile(path, []byte(middlewareDispatchHarness), 0o600))
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "gen", "jsonrpc", "plain", "server", "endpoint_dispatch_test.go"), []byte(middlewareEndpointHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "test", "./...")
}
