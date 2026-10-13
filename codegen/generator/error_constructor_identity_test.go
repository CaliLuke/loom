package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/testdata"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
	"github.com/CaliLuke/loom/internal/testingx"
)

func TestErrorConstructorIdentityCompile(t *testing.T) {
	for _, tc := range []struct {
		name   string
		design func()
	}{
		{"collision", func() {
			testdata.ErrorConstructorIdentityDSL(false)
		}},
		{"ordinary", func() {
			testdata.ErrorConstructorIdentityDSL(true)
		}},
		{"namespace", testdata.ErrorConstructorNamespaceDSL},
		{"external", testdata.ErrorConstructorExternalDSL},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if output := os.Getenv("LOOM_ERROR_CONSTRUCTOR_OUTPUT"); output != "" {
				dir = filepath.Join(output, tc.name)
				require.NoError(t, os.MkdirAll(dir, 0o750))
			}
			renderDesignModule(t, dir, "example.com/erroridentity", tc.design)
			specs, err := httpcodegen.OpenAPIFiles(codegen.RunDSL(t, tc.design))
			require.NoError(t, err)
			for _, spec := range specs {
				_, err := spec.Render(dir)
				require.NoError(t, err)
			}
			if tc.name == "collision" {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "error_identity_test.go"), []byte(errorConstructorWireHarness), 0o600))
			}
			if os.Getenv("LOOM_ERROR_CONSTRUCTOR_GENERATE_ONLY") != "" {
				return
			}
			for _, args := range [][]string{{"mod", "tidy"}, {"build", "./..."}, {"vet", "./..."}} {
				output, err := testingx.RunCmd(dir, "go", args...)
				require.NoError(t, err, output)
			}
			if tc.name == "collision" {
				output, err := testingx.RunCmd(dir, "go", "test", "-race", "-count=1", "-run", "^TestConstructorWireIdentity$", ".")
				require.NoError(t, err, output)
			}
		})
	}
}

const errorConstructorWireHarness = `package erroridentity_test

import (
    "context"
    "errors"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"

    "github.com/stretchr/testify/require"
    loomhttp "github.com/CaliLuke/loom/http"
    svc "example.com/erroridentity/gen/errors_"
    server "example.com/erroridentity/gen/http/errors_/server"
)

type implementation struct {
    err error
    calls int
}

func (s *implementation) Show(context.Context, *svc.ShowPayload) (string, error) {
    s.calls++
    return "", s.err
}

func TestConstructorWireIdentity(t *testing.T) {
    for _, tc := range []struct {
        wire string
        err error
        status int
    }{
        {"foo_bar", svc.MakeFooBar(errors.New("first")), http.StatusConflict},
        {"FooBar", svc.MakeFooBar2(errors.New("second")), http.StatusBadRequest},
    } {
        t.Run(tc.wire, func(t *testing.T) {
            service := &implementation{err: tc.err}
            mux := loomhttp.NewMuxer()
            handler := server.New(svc.NewEndpoints(service), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil)
            server.Mount(mux, handler)
            request := httptest.NewRequest(http.MethodPost, "/show", strings.NewReader("{\"value\":\"x\"}"))
            request.Header.Set("Content-Type", "application/json")
            response := httptest.NewRecorder()
            mux.ServeHTTP(response, request)
            require.Equal(t, 1, service.calls)
            require.Equal(t, tc.status, response.Code, response.Body.String())
            require.Equal(t, tc.wire, response.Header().Get("loom-error"))
        })
    }
}
`
