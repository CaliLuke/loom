package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

// TestCustomStringUnionGRPCGenerated covers metadata reachable only through a
// union branch and the selected custom-to-native protobuf conversion boundary.
func TestCustomStringUnionGRPCGenerated(t *testing.T) {
	for _, custom := range []bool{false, true} {
		name := map[bool]string{false: "native", true: "custom"}[custom]
		t.Run(name, func(t *testing.T) {
			root := RunGRPCDSL(t, func() {
				record := dsl.Type("TextRecord", func() {
					dsl.Field(1, "text", dsl.String, func() {
						if custom {
							dsl.Meta("struct:field:type", "scalar.Text", "example.com/customgrpc/scalar")
						}
					})
				})
				node := dsl.Type("Choice", func() {
					dsl.OneOf("value", func() {
						dsl.Field(1, "record", record)
						dsl.Field(2, "number", dsl.Int)
					})
				})
				dsl.Service("choices", func() {
					dsl.Method("echo", func() {
						dsl.Payload(node)
						dsl.Result(node)
						dsl.GRPC(func() {})
					})
				})
			})
			dir := t.TempDir()
			renderGRPCResponseContractModule(t, dir, "example.com/customgrpc", root)
			require.NoError(t, os.MkdirAll(filepath.Join(dir, "scalar"), 0o750))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "scalar", "text.go"), []byte(customGRPCStringSource), 0o600))
			var source string
			err := filepath.WalkDir(filepath.Join(dir, "gen", "grpc"), func(path string, entry os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if entry.IsDir() || !strings.HasSuffix(path, ".go") {
					return nil
				}
				data, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				source += string(data)
				return nil
			})
			require.NoError(t, err)
			for _, operation := range []string{"loom.StringValue(", "loom.ParseStringValue["} {
				if custom {
					require.Contains(t, source, operation)
				} else {
					require.NotContains(t, source, operation)
				}
			}

			runGRPCGoCommand(t, dir, "mod", "tidy")
			runGRPCGoCommand(t, dir, "test", "-race", "./gen/...", "./scalar")
		})
	}
}
