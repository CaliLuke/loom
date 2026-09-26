package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	dsl "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/internal/testingx"
)

// TestExampleMixedTransportServicesCompile generates the service, transport,
// and example output for designs that pair a JSON-RPC-only service with
// services on other transports, then builds and vets the resulting module.
// The JSON-RPC example server hosts every HTTP-side service, so it must
// declare the arguments the example main passes and mount each server.
func TestExampleMixedTransportServicesCompile(t *testing.T) {
	cases := []struct {
		Name        string
		Design      mixedTransportsDesign
		WantFiles   []string
		NoFiles     []string
		WantSources []string
	}{
		{
			Name:      "http-and-jsonrpc",
			Design:    mixedTransportsDesign{HTTP: true},
			WantFiles: []string{"main.go", "jsonrpc.go"},
			NoFiles:   []string{"http.go", "grpc.go"},
			WantSources: []string{
				"func handleHTTPServer(ctx context.Context, u *url.URL, webEndpoints *web.Endpoints, rpcSvc rpc.Service, rpcEndpoints *rpc.Endpoints,",
				"websvr.Mount(mux, webServer)",
				"rpcjssvr.Mount(mux, rpcJSONRPCServer)",
			},
		},
		{
			Name:      "http-jsonrpc-service-and-jsonrpc",
			Design:    mixedTransportsDesign{HTTP: true, WebJSONRPC: true},
			WantFiles: []string{"main.go", "jsonrpc.go"},
			NoFiles:   []string{"http.go", "grpc.go"},
			WantSources: []string{
				"func handleHTTPServer(ctx context.Context, u *url.URL, webEndpoints *web.Endpoints, rpcSvc rpc.Service, rpcEndpoints *rpc.Endpoints, webSvc web.Service,",
				"websvr.Mount(mux, webServer)",
				"webjssvr.Mount(mux, webJSONRPCServer)",
				"rpcjssvr.Mount(mux, rpcJSONRPCServer)",
			},
		},
		{
			Name:      "grpc-and-jsonrpc",
			Design:    mixedTransportsDesign{GRPC: true},
			WantFiles: []string{"main.go", "jsonrpc.go", "grpc.go"},
			NoFiles:   []string{"http.go"},
			WantSources: []string{
				"func handleHTTPServer(ctx context.Context, u *url.URL, rpcEndpoints *rpc.Endpoints, rpcSvc rpc.Service,",
				"rpcjssvr.Mount(mux, rpcJSONRPCServer)",
			},
		},
		{
			Name:      "http-grpc-and-jsonrpc",
			Design:    mixedTransportsDesign{HTTP: true, GRPC: true},
			WantFiles: []string{"main.go", "jsonrpc.go", "grpc.go"},
			NoFiles:   []string{"http.go"},
			WantSources: []string{
				"func handleHTTPServer(ctx context.Context, u *url.URL, webEndpoints *web.Endpoints, rpcSvc rpc.Service, rpcEndpoints *rpc.Endpoints,",
				"websvr.Mount(mux, webServer)",
				"rpcjssvr.Mount(mux, rpcJSONRPCServer)",
			},
		},
	}
	source := loomModuleSource(t)
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			codegen.RunDSL(t, c.Design.DSL)
			dir := t.TempDir()
			goMod := fmt.Sprintf("module example.com/mixed\n\ngo 1.27\n\nrequire github.com/CaliLuke/loom v0.0.0\n\nreplace github.com/CaliLuke/loom => %s\n", source)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o600))

			_, err := Generate(dir, "gen", false)
			require.NoError(t, err)
			_, err = Generate(dir, "example", false)
			require.NoError(t, err)

			cmdDir := filepath.Join(dir, "cmd", "mixed")
			for _, name := range c.WantFiles {
				assert.FileExists(t, filepath.Join(cmdDir, name))
			}
			for _, name := range c.NoFiles {
				assert.NoFileExists(t, filepath.Join(cmdDir, name))
			}
			server, err := os.ReadFile(filepath.Join(cmdDir, "jsonrpc.go"))
			require.NoError(t, err)
			for _, want := range c.WantSources {
				assert.Contains(t, string(server), want)
			}

			_, err = testingx.RunCmd(dir, "go", "mod", "tidy")
			require.NoError(t, err)
			output, err := testingx.RunCmd(dir, "go", "build", "./...")
			require.NoError(t, err, output)
			output, err = testingx.RunCmd(dir, "go", "vet", "./...")
			require.NoError(t, err, output)
		})
	}
}

// mixedTransportsDesign selects the services declared next to the
// JSON-RPC-only "rpc" service.
type mixedTransportsDesign struct {
	// HTTP declares the HTTP service "web".
	HTTP bool
	// WebJSONRPC also exposes "web" over JSON-RPC.
	WebJSONRPC bool
	// GRPC declares the gRPC-only service "store".
	GRPC bool
}

// DSL declares the design selected by d.
func (d mixedTransportsDesign) DSL() {
	dsl.API("mixed", func() {
		dsl.JSONRPC(func() {})
	})
	if d.HTTP {
		dsl.Service("web", func() {
			if d.WebJSONRPC {
				dsl.JSONRPC(func() {
					dsl.POST("/web/rpc")
				})
			}
			dsl.Method("show", func() {
				dsl.Payload(func() {
					dsl.Attribute("id", dsl.String)
					dsl.Required("id")
				})
				dsl.Result(dsl.String)
				dsl.HTTP(func() {
					dsl.GET("/widgets/{id}")
				})
			})
			if d.WebJSONRPC {
				dsl.Method("ping", func() {
					dsl.Payload(func() {
						dsl.Attribute("text", dsl.String)
						dsl.Required("text")
					})
					dsl.Result(dsl.String)
					dsl.JSONRPC(func() {})
				})
			}
		})
	}
	if d.GRPC {
		dsl.Service("store", func() {
			dsl.Method("get", func() {
				dsl.Payload(func() {
					dsl.Field(1, "id", dsl.String)
					dsl.Required("id")
				})
				dsl.Result(func() {
					dsl.Field(1, "name", dsl.String)
				})
				dsl.GRPC(func() {})
			})
		})
	}
	dsl.Service("rpc", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
		})
		dsl.Method("echo", func() {
			dsl.Payload(func() {
				dsl.Attribute("text", dsl.String)
				dsl.Required("text")
			})
			dsl.Result(dsl.String)
			dsl.JSONRPC(func() {})
		})
	})
}
