package generator

import (
	"testing"

	"github.com/stretchr/testify/require"

	dsl "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/internal/testingx"
)

// TestGRPCStreamingGeneratedCodeCompiles generates a gRPC service for every
// streaming kind with and without a regular payload, runs protoc on the
// emitted proto file, then builds and vets the generated packages.
func TestGRPCStreamingGeneratedCodeCompiles(t *testing.T) {
	cases := []struct {
		Name       string
		Kind       string
		HasPayload bool
	}{
		{"client-streaming-with-payload", "client", true},
		{"client-streaming-without-payload", "client", false},
		{"server-streaming-with-payload", "server", true},
		{"server-streaming-without-payload", "server", false},
		{"bidirectional-streaming-with-payload", "bidirectional", true},
		{"bidirectional-streaming-without-payload", "bidirectional", false},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			dir := buildGeneratedModule(t, "example.com/grpcstreaming", grpcStreamingDSL(c.Kind, c.HasPayload))
			output, err := testingx.RunCmd(dir, "go", "vet", "./...")
			require.NoError(t, err, output)
		})
	}
}

// grpcStreamingDSL returns a design with one gRPC method that streams in the
// direction named by kind ("client", "server" or "bidirectional") and that
// declares a regular payload when hasPayload is true.
func grpcStreamingDSL(kind string, hasPayload bool) func() {
	return func() {
		dsl.API("grpcstreaming", func() {})
		var Request = dsl.Type("Request", func() {
			dsl.Field(1, "id", dsl.String)
		})
		var Message = dsl.Type("Message", func() {
			dsl.Field(1, "text", dsl.String)
		})
		var Reply = dsl.Type("Reply", func() {
			dsl.Field(1, "count", dsl.Int)
		})
		dsl.Service("streamer", func() {
			dsl.Method("exchange", func() {
				if hasPayload {
					dsl.Payload(Request)
				}
				switch kind {
				case "client":
					dsl.StreamingPayload(Message)
					dsl.Result(Reply)
				case "server":
					dsl.StreamingResult(Reply)
				case "bidirectional":
					dsl.StreamingPayload(Message)
					dsl.StreamingResult(Reply)
				}
				dsl.GRPC(func() {})
			})
		})
	}
}
