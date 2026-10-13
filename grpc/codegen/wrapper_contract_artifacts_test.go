package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/grpc/codegen/testdata"
)

// TestWrapperContractArtifacts retains a bounded specimen set only when an
// external comparison directory is requested. Each invocation regenerates in
// its own process; callers compare gen/ bytes and build/vet each distinct set.
func TestWrapperContractArtifacts(t *testing.T) {
	output := os.Getenv("LOOM_WRAPPER_CONTRACT_OUTPUT")
	if output == "" {
		t.Skip("set LOOM_WRAPPER_CONTRACT_OUTPUT for isolated artifact comparison")
	}
	for _, tc := range []struct {
		name string
		dsl  func()
	}{
		{"permissive-first", wrapperContractAllShapesDSL(false)},
		{"strict-first", wrapperContractAllShapesDSL(true)},
		{"declarations", testdata.MessageDeclarationsDSL},
		{"recursive", wrapperContractRecursiveDSL},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(output, tc.name)
			require.NoError(t, os.MkdirAll(dir, 0o750))
			root := RunGRPCDSL(t, tc.dsl)
			renderGRPCModule(t, dir, "example.com/wrapperartifacts", root, resolveGRPCLoomSource(t))
			path := filepath.Join(dir, "go.mod")
			mod, err := os.ReadFile(path)
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(path, []byte(strings.ReplaceAll(string(mod), "require github.com/CaliLuke/loom v1.0.0", "require github.com/CaliLuke/loom v1.10.0-alpha.9.0.20261013020843-063198a2f427")), 0o600))
		})
	}
}

func wrapperContractAllShapesDSL(reverse bool) func() {
	return func() {
		for _, shape := range []string{"array-array", "array-map", "map-array", "map-map"} {
			free, strict := wrapperContractFields(shape, false)
			Service(shape, func() {
				Method("echo", func() {
					Payload(func() {
						if reverse {
							Field(2, "strict", strict)
							Field(1, "free", free)
						} else {
							Field(1, "free", free)
							Field(2, "strict", strict)
						}
					})
					GRPC(func() {
					})
				})
			})
		}
	}
}

func wrapperContractRecursiveDSL() {
	node := Type("Node", func() {
		Field(1, "children", ArrayOf(ArrayOf("Node")))
		Field(2, "labels", ArrayOf(ArrayOf(String)))
	})
	Service("recursive", func() {
		Method("echo", func() {
			Payload(node)
			Result(node)
			GRPC(func() {
			})
		})
	})
}
