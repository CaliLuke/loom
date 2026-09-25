package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	dsl "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/internal/loomsource"
	"github.com/CaliLuke/loom/internal/testingx"
)

// TestExampleTransportlessServiceCompile generates the service, transport,
// and example output for designs whose server hosts a service that declares
// no transport, then builds and vets the resulting module. The example main
// initializes only the services that a transport serves, so a service without
// a transport leaves no unused variable behind.
func TestExampleTransportlessServiceCompile(t *testing.T) {
	cases := []struct {
		Name string
		// Server lists the services hosted by the server "a". A nil list
		// declares no server, so the default server hosts every service.
		Server []string
		// Served lists the services that the example main initializes.
		Served []string
	}{
		{Name: "no-server", Server: nil, Served: nil},
		{Name: "plain-only", Server: []string{"plain"}, Served: nil},
		{Name: "http-and-plain", Server: []string{"web", "plain"}, Served: []string{"web"}},
		{Name: "plain-and-http", Server: []string{"plain", "web"}, Served: []string{"web"}},
		{Name: "grpc-and-plain", Server: []string{"store", "plain"}, Served: []string{"store"}},
		{Name: "jsonrpc-and-plain", Server: []string{"rpc", "plain"}, Served: []string{"rpc"}},
		{Name: "all-and-plain", Server: []string{"web", "plain", "rpc", "store"}, Served: []string{"web", "rpc", "store"}},
	}
	repoRoot, err := loomsource.RepositoryRoot(".")
	require.NoError(t, err)
	source, err := loomsource.Resolve(repoRoot, filepath.Join(t.TempDir(), "loom-pinned"))
	require.NoError(t, err)
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			codegen.RunDSL(t, transportlessServiceDesign(c.Server))
			dir := t.TempDir()
			goMod := fmt.Sprintf("module example.com/demo\n\ngo 1.27\n\nrequire github.com/CaliLuke/loom v0.0.0\n\nreplace github.com/CaliLuke/loom => %s\n", source)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o600))

			_, err := Generate(dir, "gen", false)
			require.NoError(t, err)
			_, err = Generate(dir, "example", false)
			require.NoError(t, err)

			svr := "a"
			if c.Server == nil {
				svr = "demo"
			}
			main, err := os.ReadFile(filepath.Join(dir, "cmd", svr, "main.go"))
			require.NoError(t, err)
			for _, name := range []string{"web", "store", "rpc", "plain"} {
				if slices.Contains(c.Served, name) {
					assert.Regexp(t, `(?m)^\s+`+name+`Endpoints\s+\*`+name+`\.Endpoints$`, string(main))
					continue
				}
				assert.NotRegexp(t, `\b`+name+`(Svc|Interceptors|Endpoints)\b`, string(main))
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

// transportlessServiceDesign returns a design whose server "a" hosts the
// services named in hosted, or a design without a server when hosted is nil.
// The services are "plain" (no transport, with a server interceptor), "web"
// (HTTP), "store" (gRPC) and "rpc" (JSON-RPC). Only "plain" and the hosted
// services are declared.
func transportlessServiceDesign(hosted []string) func() {
	return func() {
		dsl.API("demo", func() {
			if slices.Contains(hosted, "rpc") {
				dsl.JSONRPC(func() {})
			}
			if hosted != nil {
				dsl.Server("a", func() {
					dsl.Services(hosted...)
				})
			}
		})
		audit := dsl.Interceptor("audit", func() {})
		dsl.Service("plain", func() {
			dsl.ServerInterceptor(audit)
			dsl.Method("ping", func() {
				dsl.Payload(func() {
					dsl.Attribute("text", dsl.String)
				})
				dsl.Result(dsl.String)
			})
		})
		if slices.Contains(hosted, "web") {
			dsl.Service("web", func() {
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
			})
		}
		if slices.Contains(hosted, "store") {
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
		if slices.Contains(hosted, "rpc") {
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
	}
}
