package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

const errorResponseOwnershipHarness = `package ownership_test

import (
 "context"
 "errors"
 "fmt"
 "net/http"
 "net/http/httptest"
 "testing"

 "github.com/stretchr/testify/require"
 service "example.com/ownership/gen/ownership"
 server "example.com/ownership/gen/http/ownership/server"
 loomhttp "github.com/CaliLuke/loom/http"
 loom "github.com/CaliLuke/loom/pkg"
)

type (
 namedOuter struct {
  cause error
 }
 safeOuter struct {
  cause error
 }
)

func TestMountedResponseOwner(t *testing.T) {
 typed := &service.TypedError{Name: "denied", Detail: "branch-private-sentinel"}
 other := errors.New("independent cleanup")
 for _, tc := range []struct {
  name string
  err error
  owned bool
 }{
  {"direct", typed, true},
  {"wrapped", fmt.Errorf("diagnostic: %w", typed), true},
  {"single join", errors.Join(nil, typed), true},
  {"safe wrapper", &safeOuter{cause: typed}, true},
  {"same name outer", &namedOuter{cause: typed}, false},
  {"same name outer join", &namedOuter{cause: errors.Join(typed, other)}, false},
  {"join", errors.Join(typed, other), false},
  {"reverse", errors.Join(other, typed), false},
 } {
  t.Run(tc.name, func(t *testing.T) {
   calls := 0
   mux := loomhttp.NewMuxer()
   endpoints := &service.Endpoints{Show: func(context.Context, any) (any, error) {
    calls++
    return nil, tc.err
   }}
   generated := server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, func(_ context.Context, _ http.ResponseWriter, err error) {
    t.Errorf("encode: %v", err)
   }, nil)
   server.Mount(mux, generated)
   rec := httptest.NewRecorder()
   mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/show", nil))
   require.Equal(t, 1, calls)
   if tc.owned {
    require.Equal(t, http.StatusForbidden, rec.Code)
    require.Contains(t, rec.Body.String(), "branch-private-sentinel")
    return
   }
   require.Equal(t, http.StatusInternalServerError, rec.Code)
   require.NotContains(t, rec.Body.String(), "branch-private-sentinel")
   require.NotEqual(t, "denied", rec.Header().Get("loom-error"))
  })
 }
}

func TestFormatterReceivesOriginalWrapper(t *testing.T) {
 typed := &service.TypedError{Name: "denied", Detail: "branch-private-sentinel"}
 original := fmt.Errorf("wrapper: %w", typed)
 calls := 0
 mux := loomhttp.NewMuxer()
 endpoints := &service.Endpoints{Show: func(context.Context, any) (any, error) {
  return nil, original
 }}
 generated := server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, func(_ context.Context, _ http.ResponseWriter, err error) {
  t.Errorf("encode: %v", err)
 }, func(ctx context.Context, err error) loomhttp.StatusCoder {
  calls++
  require.Same(t, original, err)
  return loomhttp.NewProblemResponse(ctx, err, http.StatusForbidden, "", "")
 })
 server.Mount(mux, generated)
 rec := httptest.NewRecorder()
 mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/show", nil))
 require.Equal(t, 1, calls)
 require.Equal(t, http.StatusForbidden, rec.Code)
}

func (e *namedOuter) Error() string {
 return "outer detail"
}

func (e *namedOuter) LoomErrorName() string {
 return "denied"
}

func (e *namedOuter) Unwrap() error {
 return e.cause
}

func (e *safeOuter) Error() string {
 return "diagnostic wrapper"
}

func (e *safeOuter) LoomErrorRemedy() *loom.ErrorRemedy {
 return &loom.ErrorRemedy{SafeMessage: "outer safe"}
}

func (e *safeOuter) Unwrap() error {
 return e.cause
}
`

func TestGeneratedErrorResponseOwnership(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		typed := dsl.Type("TypedError", func() {
			dsl.ErrorName("name", dsl.String)
			dsl.Attribute("detail", dsl.String)
			dsl.Required("name", "detail")
		})
		dsl.Service("ownership", func() {
			dsl.Method("show", func() {
				dsl.Result(dsl.String)
				dsl.Error("denied", typed)
				dsl.HTTP(func() {
					dsl.GET("/show")
					dsl.Response("denied", dsl.StatusForbidden)
				})
			})
		})
	})
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/ownership", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ownership_test.go"), []byte(errorResponseOwnershipHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "-count=1", "./...")
}
