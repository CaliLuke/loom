package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	dsl "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/internal/testingx"
)

// TestRecursiveLengthValidatedExampleCompile generates the service,
// transport and example output for a design whose recursive types reach
// themselves through arrays and maps with length validations, exposed on
// HTTP and JSON-RPC, then builds and vets the module. The payload and result
// examples of the client CLIs stop at the type that is still being
// generated.
func TestRecursiveLengthValidatedExampleCompile(t *testing.T) {
	source := loomModuleSource(t)
	codegen.RunDSL(t, recursiveLengthValidatedDesign)
	dir := t.TempDir()
	goMod := fmt.Sprintf("module example.com/tree\n\ngo 1.27\n\nrequire github.com/CaliLuke/loom v0.0.0\n\nreplace github.com/CaliLuke/loom => %s\n", source)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o600))

	_, err := Generate(dir, "gen", false)
	require.NoError(t, err)
	_, err = Generate(dir, "example", false)
	require.NoError(t, err)

	cli, err := os.ReadFile(filepath.Join(dir, "gen", "http", "cli", "tree", "cli.go"))
	require.NoError(t, err)
	require.Contains(t, string(cli), `\"children\": [`)
	require.FileExists(t, filepath.Join(dir, "gen", "jsonrpc", "cli", "tree", "cli.go"))
	require.FileExists(t, filepath.Join(dir, "cmd", "tree-cli", "http.go"))
	require.FileExists(t, filepath.Join(dir, "cmd", "tree-cli", "jsonrpc.go"))

	_, err = testingx.RunCmd(dir, "go", "mod", "tidy")
	require.NoError(t, err)
	output, err := testingx.RunCmd(dir, "go", "build", "./...")
	require.NoError(t, err, output)
	output, err = testingx.RunCmd(dir, "go", "vet", "./...")
	require.NoError(t, err, output)
}

// recursiveLengthValidatedDesign declares a type that holds an array of
// itself with MaxLength, the design of issue #380, a type that holds a map of
// itself with MinLength, and two types that reach each other through arrays
// with length validations, used as payloads and results on HTTP and
// JSON-RPC.
func recursiveLengthValidatedDesign() {
	dsl.API("tree", func() {
		dsl.JSONRPC(func() {})
		dsl.Server("tree", func() {
			dsl.Services("nodes", "nodesrpc")
		})
	})
	var Node = dsl.Type("Node", func() {
		dsl.Attribute("children", dsl.ArrayOf("Node"), func() {
			dsl.MaxLength(3)
		})
	})
	var Index = dsl.Type("Index", func() {
		dsl.Attribute("entries", dsl.MapOf(dsl.String, "Index"), func() {
			dsl.MinLength(1)
		})
	})
	var Parent = dsl.Type("Parent", func() {
		dsl.Attribute("kids", dsl.ArrayOf("Child"), func() {
			dsl.MaxLength(2)
		})
	})
	dsl.Type("Child", func() {
		dsl.Attribute("parents", dsl.ArrayOf(Parent), func() {
			dsl.MinLength(1)
			dsl.MaxLength(2)
		})
	})
	dsl.Service("nodes", func() {
		dsl.Method("grow", func() {
			dsl.Payload(Node)
			dsl.Result(Node)
			dsl.HTTP(func() {
				dsl.POST("/grow")
			})
		})
		dsl.Method("lookup", func() {
			dsl.Payload(Index)
			dsl.Result(Index)
			dsl.HTTP(func() {
				dsl.POST("/lookup")
			})
		})
		dsl.Method("adopt", func() {
			dsl.Payload(Parent)
			dsl.Result(Parent)
			dsl.HTTP(func() {
				dsl.POST("/adopt")
			})
		})
	})
	dsl.Service("nodesrpc", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
		})
		dsl.Method("grow", func() {
			dsl.Payload(Node)
			dsl.Result(Node)
			dsl.JSONRPC(func() {})
		})
		dsl.Method("adopt", func() {
			dsl.Payload(Parent)
			dsl.Result(Parent)
			dsl.JSONRPC(func() {})
		})
	})
}
