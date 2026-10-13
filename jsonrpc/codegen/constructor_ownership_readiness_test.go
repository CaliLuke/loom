package codegen

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	httpcodegen "github.com/CaliLuke/loom/http/codegen"
	"github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/stretchr/testify/require"
)

func TestConstructorOwnershipGenerated(t *testing.T) {
	for _, mode := range []string{"handler", "response", "shared", "distinct", "ordinary", "selector", "same"} {
		for _, reverse := range []bool{false, true} {
			if reverse && mode != "shared" {
				continue
			}
			name := mode
			if reverse {
				name += "-reverse"
			}
			t.Run(name, func(t *testing.T) {
				root := RunJSONRPCDSL(t, func() {
					testdata.ConstructorOwnership(mode, true, reverse)
				})
				dir := t.TempDir()
				if output := os.Getenv("LOOM_CONSTRUCTOR_OWNERSHIP_OUTPUT"); output != "" {
					dir = filepath.Join(output, "jsonrpc", name)
					require.NoError(t, os.MkdirAll(dir, 0o750))
				}
				renderJSONRPCModule(t, dir, "example.com/constructors", root)
				files, err := httpcodegen.OpenAPIFiles(root)
				require.NoError(t, err)
				for _, file := range files {
					_, err := file.Render(dir)
					require.NoError(t, err)
				}
				if mode == "shared" {
					require.NoError(t, os.WriteFile(filepath.Join(dir, "source_test.go"), []byte(fmt.Sprintf(testdata.ConstructorOwnershipSharedHarness, "jsonrpc", "", true, true)), 0o600))
				}
				if mode == "selector" {
					require.NoError(t, os.WriteFile(filepath.Join(dir, "source_test.go"), []byte(fmt.Sprintf(testdata.ConstructorOwnershipWireHarness, "jsonrpc")), 0o600))
				}
				if os.Getenv("LOOM_CONSTRUCTOR_GENERATE_ONLY") != "" {
					return
				}
				runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
				runGoJSONRPCTestCommand(t, dir, "build", "./...")
				if mode == "selector" || mode == "shared" {
					runGoJSONRPCTestCommand(t, dir, "test", "-run", "^Test(SelectedConstructorSource|MountedSharedConstructor)$", ".")
				}
			})
		}
	}
}
