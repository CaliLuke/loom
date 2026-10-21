package generator

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	dsl "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/internal/testingx"
)

func TestExampleCLISelectsTransportByCommand(t *testing.T) {
	codegen.RunDSL(t, cliTransportDispatchDesign)
	dir := t.TempDir()
	goMod := fmt.Sprintf("module example.com/dispatch\n\ngo 1.27.3\n\nrequire github.com/CaliLuke/loom v0.0.0\n\nreplace github.com/CaliLuke/loom => %s\n", loomModuleSource(t))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o600))

	_, err := Generate(dir, "gen", "github.com/CaliLuke/loom/codegen/generator", false)
	require.NoError(t, err)
	_, err = Generate(dir, "example", "github.com/CaliLuke/loom/codegen/generator", false)
	require.NoError(t, err)
	_, err = testingx.RunCmd(dir, "go", "mod", "tidy")
	require.NoError(t, err)
	output, err := testingx.RunCmd(dir, "go", "build", "-o", filepath.Join(dir, "mixed-cli"), "./cmd/mixed-cli")
	require.NoError(t, err, output)

	requests := make(chan *http.Request, 8)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.Clone(r.Context())
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)
	command := filepath.Join(dir, "mixed-cli")
	cases := []struct {
		name   string
		args   []string
		method string
		path   string
	}{
		{name: "JSON-RPC-only service", args: []string{"rpc", "ping"}, method: http.MethodPost, path: "/rpc"},
		{name: "HTTP-only service", args: []string{"web", "ping"}, method: http.MethodGet, path: "/ping"},
		{name: "overlapping service JSON-RPC method", args: []string{"shared", "rpc-only"}, method: http.MethodPost, path: "/shared-rpc"},
		{name: "overlapping service HTTP method", args: []string{"shared", "http-only"}, method: http.MethodGet, path: "/http-only"},
		{name: "explicit JSON-RPC", args: []string{"-j", "rpc", "ping"}, method: http.MethodPost, path: "/rpc"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{"-url", server.URL}, tc.args...)
			_, err := testingx.RunCmd(dir, command, args...)
			require.Error(t, err, "server's 500 response should reach the client")
			select {
			case request := <-requests:
				require.Equal(t, tc.method, request.Method)
				require.Equal(t, tc.path, request.URL.Path)
			case <-time.After(5 * time.Second):
				t.Fatalf("command %v did not reach the server", tc.args)
			}
		})
	}

	output, err = testingx.RunCmd(dir, command, "-url", server.URL, "rpc", "--help")
	require.NoError(t, err)
	require.Contains(t, output, "rpc ping")
	select {
	case request := <-requests:
		t.Fatalf("help command unexpectedly reached %s %s", request.Method, request.URL.Path)
	case <-time.After(100 * time.Millisecond):
	}

	for _, args := range [][]string{{"web", "ping", "unexpected"}, {"missing", "ping"}} {
		output, err := testingx.RunCmd(dir, command, append([]string{"-url", server.URL}, args...)...)
		require.Error(t, err)
		require.NotContains(t, output, "500")
		select {
		case request := <-requests:
			t.Fatalf("invalid command %v unexpectedly reached %s %s", args, request.Method, request.URL.Path)
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func TestHTTPOnlyExampleCLICompiles(t *testing.T) {
	codegen.RunDSL(t, httpOnlyExampleCLIDesign)
	dir := t.TempDir()
	goMod := fmt.Sprintf("module example.com/http-only\n\ngo 1.27.3\n\nrequire github.com/CaliLuke/loom v0.0.0\n\nreplace github.com/CaliLuke/loom => %s\n", loomModuleSource(t))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o600))

	_, err := Generate(dir, "gen", "github.com/CaliLuke/loom/codegen/generator", false)
	require.NoError(t, err)
	_, err = Generate(dir, "example", "github.com/CaliLuke/loom/codegen/generator", false)
	require.NoError(t, err)
	_, err = testingx.RunCmd(dir, "go", "mod", "tidy")
	require.NoError(t, err)
	output, err := testingx.RunCmd(dir, "go", "build", "./cmd/http-cli")
	require.NoError(t, err, output)
}

func cliTransportDispatchDesign() {
	dsl.API("dispatch", func() {
		dsl.JSONRPC(func() {})
		dsl.Server("mixed", func() {
			dsl.Services("web", "rpc", "shared")
			dsl.Host("local", func() {
				dsl.URI("http://localhost:8080")
			})
		})
	})
	dsl.Service("web", func() {
		dsl.Method("ping", func() {
			dsl.Result(dsl.String)
			dsl.HTTP(func() {
				dsl.GET("/ping")
			})
		})
	})
	dsl.Service("rpc", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
		})
		dsl.Method("ping", func() {
			dsl.Result(dsl.String)
			dsl.JSONRPC(func() {})
		})
	})
	dsl.Service("shared", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/shared-rpc")
		})
		dsl.Method("http-only", func() {
			dsl.Result(dsl.String)
			dsl.HTTP(func() {
				dsl.GET("/http-only")
			})
		})
		dsl.Method("rpc-only", func() {
			dsl.Result(dsl.String)
			dsl.JSONRPC(func() {})
		})
	})
}

func httpOnlyExampleCLIDesign() {
	dsl.API("http-only", func() {
		dsl.Server("http", func() {
			dsl.Services("web")
			dsl.Host("local", func() {
				dsl.URI("http://localhost:8080")
			})
		})
	})
	dsl.Service("web", func() {
		dsl.Method("ping", func() {
			dsl.Result(dsl.String)
			dsl.HTTP(func() {
				dsl.GET("/ping")
			})
		})
	})
}
