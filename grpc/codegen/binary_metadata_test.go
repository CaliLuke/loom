package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

func TestBinaryMetadataRoundTrip(t *testing.T) {
	for _, placement := range []string{"request", "header", "trailer"} {
		for _, key := range []string{"x-raw", "x-raw-bin"} {
			t.Run(placement+"/"+key, func(t *testing.T) {
				root := RunGRPCDSL(t, func() {
					binaryMetadataDSL(placement, key)
				})
				dir := t.TempDir()
				if output := os.Getenv("LOOM_BINARY_METADATA_OUTPUT"); output != "" {
					dir = filepath.Join(output, placement, key)
					require.NoError(t, os.MkdirAll(dir, 0o750))
				}
				renderGRPCResponseContractModule(t, dir, "example.com/binarymetadata", root)
				require.NoError(t, os.WriteFile(filepath.Join(dir, "binary_test.go"), []byte(binaryMetadataHarness), 0o600))
				peer := strings.ReplaceAll(binaryMetadataPeerHarness, "PLACEMENT", placement)
				require.NoError(t, os.WriteFile(filepath.Join(dir, "peer_test.go"), []byte(peer), 0o600))
				if os.Getenv("LOOM_BINARY_METADATA_GENERATE_ONLY") == "1" {
					return
				}
				runGRPCGoCommand(t, dir, "mod", "tidy")
				runGRPCGoCommand(t, dir, "build", "./...")
				runGRPCGoCommand(t, dir, "vet", "./...")
				runGRPCGoCommand(t, dir, "test", "-race", "-count=1", "-v", ".")
			})
		}
	}
}

func binaryMetadataDSL(placement, key string) {
	value := dsl.Type("Value", func() {
		dsl.Field(1, "body", dsl.String)
		dsl.Field(2, "raw", dsl.Bytes)
		dsl.Required("body", "raw")
	})
	dsl.Service("binary", func() {
		dsl.Method("echo", func() {
			dsl.Payload(value)
			dsl.Result(value)
			dsl.GRPC(func() {
				if placement == "request" {
					dsl.Message(func() {
						dsl.Attribute("body")
					})
					dsl.Metadata(func() {
						dsl.Attribute("raw:" + key)
					})
				}
				dsl.Response(dsl.CodeOK, func() {
					if placement != "request" {
						dsl.Message(func() {
							dsl.Attribute("body")
						})
					}
					switch placement {
					case "header":
						dsl.Headers(func() {
							dsl.Attribute("raw:" + key)
						})
					case "trailer":
						dsl.Trailers(func() {
							dsl.Attribute("raw:" + key)
						})
					}
				})
			})
		})
	})
}
