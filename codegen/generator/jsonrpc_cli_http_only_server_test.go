package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	dsl "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/internal/loomsource"
	"github.com/CaliLuke/loom/internal/testingx"
)

// TestJSONRPCCLIHTTPOnlyServerCompile generates the service, transport, and
// example output for a design whose server "b" hosts only an HTTP service
// while server "a" hosts a JSON-RPC service, then builds and vets the module.
// Neither server may get a client CLI for a transport it does not host.
func TestJSONRPCCLIHTTPOnlyServerCompile(t *testing.T) {
	repoRoot, err := loomsource.RepositoryRoot(".")
	require.NoError(t, err)
	source, err := loomsource.Resolve(repoRoot, filepath.Join(t.TempDir(), "loom-pinned"))
	require.NoError(t, err)
	codegen.RunDSL(t, jsonrpcCLIHTTPOnlyServerDesign)
	dir := t.TempDir()
	goMod := fmt.Sprintf("module example.com/split\n\ngo 1.27\n\nrequire github.com/CaliLuke/loom v0.0.0\n\nreplace github.com/CaliLuke/loom => %s\n", source)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o600))

	_, err = Generate(dir, "gen", false)
	require.NoError(t, err)
	_, err = Generate(dir, "example", false)
	require.NoError(t, err)

	// Each server gets the client CLI of the transports it hosts only.
	require.FileExists(t, filepath.Join(dir, "gen", "jsonrpc", "cli", "a", "cli.go"))
	require.FileExists(t, filepath.Join(dir, "gen", "http", "cli", "b", "cli.go"))
	require.NoDirExists(t, filepath.Join(dir, "gen", "http", "cli", "a"))
	require.NoDirExists(t, filepath.Join(dir, "gen", "jsonrpc", "cli", "b"))
	require.FileExists(t, filepath.Join(dir, "cmd", "a-cli", "jsonrpc.go"))
	require.FileExists(t, filepath.Join(dir, "cmd", "b-cli", "http.go"))
	require.NoFileExists(t, filepath.Join(dir, "cmd", "a-cli", "http.go"))
	require.NoFileExists(t, filepath.Join(dir, "cmd", "b-cli", "jsonrpc.go"))

	_, err = testingx.RunCmd(dir, "go", "mod", "tidy")
	require.NoError(t, err)
	output, err := testingx.RunCmd(dir, "go", "build", "./...")
	require.NoError(t, err, output)
	output, err = testingx.RunCmd(dir, "go", "vet", "./...")
	require.NoError(t, err, output)
}

// jsonrpcCLIHTTPOnlyServerDesign declares server "a" hosting the JSON-RPC
// service "rpc" and server "b" hosting only the HTTP service "web".
func jsonrpcCLIHTTPOnlyServerDesign() {
	dsl.API("split", func() {
		dsl.JSONRPC(func() {})
		dsl.Server("a", func() {
			dsl.Services("rpc")
		})
		dsl.Server("b", func() {
			dsl.Services("web")
		})
	})
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
