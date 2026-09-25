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

// TestExampleStreamMethodsCompile generates the service, transport, and
// example output for methods that return a result next to a server stream
// and for the methods of a JSON-RPC WebSocket service that receive a
// streaming payload and return a result, then builds and vets the module.
// The example stubs must implement the service interface, and the JSON-RPC
// WebSocket server must send the result type of each method, or no result
// for a method without one.
func TestExampleStreamMethodsCompile(t *testing.T) {
	cases := []struct {
		Name string
		DSL  func()
	}{
		{Name: "http-sse-mixed-results", DSL: httpSSEMixedResultsDSL},
		{Name: "jsonrpc-sse-mixed-results", DSL: jsonrpcSSEMixedResultsDSL},
		{Name: "jsonrpc-websocket-streaming-payload", DSL: jsonrpcWebSocketStreamingPayloadDSL},
	}
	repoRoot, err := loomsource.RepositoryRoot(".")
	require.NoError(t, err)
	source, err := loomsource.Resolve(repoRoot, filepath.Join(t.TempDir(), "loom-pinned"))
	require.NoError(t, err)
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			codegen.RunDSL(t, c.DSL)
			dir := t.TempDir()
			goMod := fmt.Sprintf("module example.com/streams\n\ngo 1.27\n\nrequire github.com/CaliLuke/loom v0.0.0\n\nreplace github.com/CaliLuke/loom => %s\n", source)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o600))

			_, err := Generate(dir, "gen", false)
			require.NoError(t, err)
			_, err = Generate(dir, "example", false)
			require.NoError(t, err)

			_, err = testingx.RunCmd(dir, "go", "mod", "tidy")
			require.NoError(t, err)
			output, err := testingx.RunCmd(dir, "go", "build", "./...")
			require.NoError(t, err, output)
			output, err = testingx.RunCmd(dir, "go", "vet", "./...")
			require.NoError(t, err, output)
		})
	}
}

// httpSSEMixedResultsDSL declares HTTP SSE methods whose result differs from
// the streamed type.
func httpSSEMixedResultsDSL() {
	dsl.API("streams", func() {})
	note := dsl.Type("Note", func() {
		dsl.Attribute("text", dsl.String)
		dsl.Required("text")
	})
	dsl.Service("feed", func() {
		dsl.Method("watch", func() {
			dsl.Payload(func() {
				dsl.Attribute("id", dsl.String)
			})
			dsl.Result(note)
			dsl.StreamingResult(dsl.String)
			dsl.HTTP(func() {
				dsl.GET("/watch")
				dsl.Param("id")
				dsl.ServerSentEvents()
			})
		})
		dsl.Method("tail", func() {
			dsl.Result(dsl.Int)
			dsl.StreamingResult(note)
			dsl.HTTP(func() {
				dsl.GET("/tail")
				dsl.ServerSentEvents()
			})
		})
	})
}

// jsonrpcSSEMixedResultsDSL declares a JSON-RPC SSE method whose result
// differs from the streamed type.
func jsonrpcSSEMixedResultsDSL() {
	dsl.API("streams", func() {
		dsl.JSONRPC(func() {})
	})
	note := dsl.Type("Note", func() {
		dsl.Attribute("text", dsl.String)
		dsl.Required("text")
	})
	dsl.Service("feed", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
		})
		dsl.Method("watch", func() {
			dsl.Payload(func() {
				dsl.ID("id", dsl.String)
			})
			dsl.Result(note)
			dsl.StreamingResult(dsl.String)
			dsl.JSONRPC(func() {
				dsl.ServerSentEvents()
			})
		})
	})
}

// jsonrpcWebSocketStreamingPayloadDSL declares a JSON-RPC WebSocket service
// whose methods receive a streaming payload and return results of a user
// type, an inline object and an array, or no result.
func jsonrpcWebSocketStreamingPayloadDSL() {
	dsl.API("streams", func() {
		dsl.JSONRPC(func() {})
	})
	note := dsl.Type("Note", func() {
		dsl.Attribute("text", dsl.String)
		dsl.Required("text")
	})
	dsl.Service("files", func() {
		dsl.JSONRPC(func() {
			dsl.GET("/ws")
		})
		dsl.Method("upload", func() {
			dsl.StreamingPayload(note)
			dsl.Result(note)
			dsl.JSONRPC(func() {})
		})
		dsl.Method("count", func() {
			dsl.StreamingPayload(dsl.String)
			dsl.Result(func() {
				dsl.Attribute("count", dsl.Int)
			})
			dsl.JSONRPC(func() {})
		})
		dsl.Method("names", func() {
			dsl.StreamingPayload(dsl.String)
			dsl.Result(dsl.ArrayOf(dsl.String))
			dsl.JSONRPC(func() {})
		})
		dsl.Method("push", func() {
			dsl.StreamingPayload(dsl.String)
			dsl.JSONRPC(func() {})
		})
	})
}
