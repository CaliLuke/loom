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

// TestExampleMultiServerTransportsCompile generates the service, transport,
// and example output for designs with two servers that host services of
// different transports, then builds and vets the resulting module. Each server
// gets the example server files of the transports it hosts only, so that no
// example handler is declared without a service to serve.
func TestExampleMultiServerTransportsCompile(t *testing.T) {
	http := multiServerWant{Server: []string{"http.go"}, CLI: []string{"http"}}
	grpc := multiServerWant{Server: []string{"grpc.go"}, CLI: []string{"grpc"}}
	jsonrpc := multiServerWant{Server: []string{"jsonrpc.go"}, CLI: []string{"jsonrpc"}}
	httpGRPC := multiServerWant{Server: []string{"http.go", "grpc.go"}, CLI: []string{"http", "grpc"}}
	cases := []struct {
		Name  string
		A     []string
		B     []string
		WantA multiServerWant
		WantB multiServerWant
	}{
		{Name: "grpc-and-http", A: []string{"store"}, B: []string{"web"}, WantA: grpc, WantB: http},
		{Name: "grpc-and-jsonrpc", A: []string{"store"}, B: []string{"rpc"}, WantA: grpc, WantB: jsonrpc},
		{Name: "http-and-jsonrpc", A: []string{"web"}, B: []string{"rpc"}, WantA: http, WantB: jsonrpc},
		{Name: "http-grpc-and-grpc", A: []string{"web", "store"}, B: []string{"store"}, WantA: httpGRPC, WantB: grpc},
		{Name: "dual-and-http", A: []string{"dual"}, B: []string{"web"}, WantA: httpGRPC, WantB: http},
		{Name: "dual-and-grpc", A: []string{"dual"}, B: []string{"store"}, WantA: httpGRPC, WantB: grpc},
		{
			Name:  "jsonrpc-grpc-and-http",
			A:     []string{"rpc", "store"},
			B:     []string{"web"},
			WantA: multiServerWant{Server: []string{"jsonrpc.go", "grpc.go"}, CLI: []string{"jsonrpc", "grpc"}},
			WantB: http,
		},
		{
			Name:  "all-and-grpc",
			A:     []string{"web", "rpc", "store"},
			B:     []string{"store"},
			WantA: multiServerWant{Server: []string{"jsonrpc.go", "grpc.go"}, CLI: []string{"http", "jsonrpc", "grpc"}},
			WantB: grpc,
		},
		{
			Name:  "all-and-jsonrpc",
			A:     []string{"web", "rpc", "dual"},
			B:     []string{"rpc"},
			WantA: multiServerWant{Server: []string{"jsonrpc.go", "grpc.go"}, CLI: []string{"http", "jsonrpc", "grpc"}},
			WantB: jsonrpc,
		},
	}
	repoRoot, err := loomsource.RepositoryRoot(".")
	require.NoError(t, err)
	source, err := loomsource.Resolve(repoRoot, filepath.Join(t.TempDir(), "loom-pinned"))
	require.NoError(t, err)
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			codegen.RunDSL(t, multiServerTransportsDesign(c.A, c.B))
			dir := t.TempDir()
			goMod := fmt.Sprintf("module example.com/multi\n\ngo 1.27\n\nrequire github.com/CaliLuke/loom v0.0.0\n\nreplace github.com/CaliLuke/loom => %s\n", source)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o600))

			_, err := Generate(dir, "gen", false)
			require.NoError(t, err)
			_, err = Generate(dir, "example", false)
			require.NoError(t, err)

			c.WantA.assertFiles(t, dir, "a")
			c.WantB.assertFiles(t, dir, "b")

			_, err = testingx.RunCmd(dir, "go", "mod", "tidy")
			require.NoError(t, err)
			output, err := testingx.RunCmd(dir, "go", "build", "./...")
			require.NoError(t, err, output)
			output, err = testingx.RunCmd(dir, "go", "vet", "./...")
			require.NoError(t, err, output)
		})
	}
}

// multiServerWant lists the example files expected for one server.
type multiServerWant struct {
	// Server lists the example server files under cmd/<server>.
	Server []string
	// CLI lists the transports of the example client files under
	// cmd/<server>-cli and of the CLI support packages under
	// gen/<transport>/cli/<server>.
	CLI []string
}

// assertFiles checks that the module in dir contains exactly the example
// server and client files listed in w for the server svr.
func (w multiServerWant) assertFiles(t *testing.T, dir, svr string) {
	t.Helper()
	for _, name := range []string{"http.go", "grpc.go", "jsonrpc.go"} {
		file := filepath.Join(dir, "cmd", svr, name)
		if slices.Contains(w.Server, name) {
			assert.FileExists(t, file)
			continue
		}
		assert.NoFileExists(t, file)
	}
	for _, transport := range []string{"http", "grpc", "jsonrpc"} {
		cli := filepath.Join(dir, "cmd", svr+"-cli", transport+".go")
		pkg := filepath.Join(dir, "gen", transport, "cli", svr, "cli.go")
		if slices.Contains(w.CLI, transport) {
			assert.FileExists(t, cli)
			assert.FileExists(t, pkg)
			continue
		}
		assert.NoFileExists(t, cli)
		assert.NoFileExists(t, pkg)
	}
}

// multiServerTransportsDesign returns a design whose server "a" hosts the
// services named in a and whose server "b" hosts the services named in b. The
// services are "web" (HTTP), "store" (gRPC), "rpc" (JSON-RPC) and "dual"
// (HTTP and gRPC). Only the hosted services are declared.
func multiServerTransportsDesign(a, b []string) func() {
	return func() {
		hosted := slices.Concat(a, b)
		dsl.API("multi", func() {
			if slices.Contains(hosted, "rpc") {
				dsl.JSONRPC(func() {})
			}
			dsl.Server("a", func() {
				dsl.Services(a...)
			})
			dsl.Server("b", func() {
				dsl.Services(b...)
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
		if slices.Contains(hosted, "dual") {
			dsl.Service("dual", func() {
				dsl.Method("fetch", func() {
					dsl.Payload(func() {
						dsl.Field(1, "id", dsl.String)
						dsl.Required("id")
					})
					dsl.Result(func() {
						dsl.Field(1, "name", dsl.String)
					})
					dsl.HTTP(func() {
						dsl.GET("/dual/{id}")
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
