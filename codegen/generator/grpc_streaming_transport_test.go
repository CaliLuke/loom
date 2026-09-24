package generator

import (
	"testing"

	"github.com/stretchr/testify/require"

	dsl "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/internal/testingx"
)

// TestGRPCStreamingGeneratedCodeCompiles generates a gRPC service for every
// streaming kind with and without a regular payload, runs protoc on the
// emitted proto file, then builds and vets the generated packages. Every case
// runs for a plain service name and for names whose package import aliases
// differ from the alias Jennifer derives from an import path: an underscore
// in the protobuf package (event_streamerpb) and an escaped astral rune
// (U0001d49cstreamer) in the service and views packages.
func TestGRPCStreamingGeneratedCodeCompiles(t *testing.T) {
	services := []struct {
		Name    string
		Service string
	}{
		{"plain", "streamer"},
		{"underscored", "event_streamer"},
		{"astral", "𝒜streamer"},
	}
	cases := []struct {
		Name       string
		Kind       string
		HasPayload bool
	}{
		{"client-streaming-with-payload", "client", true},
		{"client-streaming-without-payload", "client", false},
		{"server-streaming-with-payload", "server", true},
		{"server-streaming-without-payload", "server", false},
		{"server-streaming-viewed-result", "server-viewed", true},
		{"bidirectional-streaming-with-payload", "bidirectional", true},
		{"bidirectional-streaming-without-payload", "bidirectional", false},
	}
	for _, s := range services {
		for _, c := range cases {
			t.Run(s.Name+"/"+c.Name, func(t *testing.T) {
				dir := buildGeneratedModule(t, "example.com/grpcstreaming", grpcStreamingDSL(s.Service, c.Kind, c.HasPayload))
				output, err := testingx.RunCmd(dir, "go", "vet", "./...")
				require.NoError(t, err, output)
			})
		}
	}
}

// grpcStreamingDSL returns a design with one gRPC method on the service named
// svc that streams in the direction named by kind ("client", "server",
// "server-viewed" or "bidirectional") and that declares a regular payload when
// hasPayload is true. The "server-viewed" kind streams a result type with
// several views and no view selected by the design.
func grpcStreamingDSL(svc, kind string, hasPayload bool) func() {
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
		var ViewedReply = dsl.ResultType("application/vnd.viewed-reply", func() {
			dsl.TypeName("ViewedReply")
			dsl.Attributes(func() {
				dsl.Field(1, "count", dsl.Int)
				dsl.Field(2, "note", dsl.String)
			})
			dsl.View("default", func() {
				dsl.Attribute("count")
				dsl.Attribute("note")
			})
			dsl.View("tiny", func() {
				dsl.Attribute("count")
			})
		})
		dsl.Service(svc, func() {
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
				case "server-viewed":
					dsl.StreamingResult(ViewedReply)
				case "bidirectional":
					dsl.StreamingPayload(Message)
					dsl.StreamingResult(Reply)
				}
				dsl.GRPC(func() {})
			})
		})
	}
}
