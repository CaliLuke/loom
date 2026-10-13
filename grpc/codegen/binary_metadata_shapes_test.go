package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

func TestBinaryMetadataExplicitShapesCompile(t *testing.T) {
	for _, shape := range []string{"bytes", "named-bytes", "bytes-array", "named-array"} {
		t.Run(shape, func(t *testing.T) {
			root := RunGRPCDSL(t, func() {
				binaryMetadataShapeDSL(shape)
			})
			dir := t.TempDir()
			if output := os.Getenv("LOOM_BINARY_METADATA_SHAPES_OUTPUT"); output != "" {
				dir = filepath.Join(output, shape)
				require.NoError(t, os.MkdirAll(dir, 0o750))
			}
			renderGRPCResponseContractModule(t, dir, "example.com/binaryshapes", root)
			runGRPCGoCommand(t, dir, "mod", "tidy")
			runGRPCGoCommand(t, dir, "build", "./...")
		})
	}
}

func binaryMetadataShapeDSL(shape string) {
	var raw any = dsl.Bytes
	switch shape {
	case "named-bytes":
		raw = dsl.Type("Raw", dsl.Bytes)
	case "bytes-array":
		raw = dsl.ArrayOf(dsl.Bytes)
	case "named-array":
		raw = dsl.Type("RawList", dsl.ArrayOf(dsl.Bytes))
	}
	value := dsl.Type("Value", func() {
		dsl.Field(1, "body", dsl.String)
		dsl.Field(2, "raw", raw)
		dsl.Required("body", "raw")
	})
	dsl.Service("binary", func() {
		dsl.Method("echo", func() {
			dsl.Payload(value)
			dsl.Result(value)
			dsl.GRPC(func() {
				dsl.Metadata(func() {
					dsl.Attribute("raw:x-raw-bin")
				})
				dsl.Response(dsl.CodeOK, func() {
					dsl.Headers(func() {
						dsl.Attribute("raw:x-raw-bin")
					})
				})
			})
		})
	})
}
