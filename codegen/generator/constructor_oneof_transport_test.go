package generator

import (
	"testing"

	"github.com/stretchr/testify/require"

	dsl "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/internal/testingx"
)

// TestConstructorOneOfFieldGeneratedCodeCompiles builds and vets the service
// and transport packages generated for constructor OneOf unions passed to
// Field, optional, required and nested in a type, in a design exposed on
// HTTP, gRPC and JSON-RPC.
func TestConstructorOneOfFieldGeneratedCodeCompiles(t *testing.T) {
	dir := buildGeneratedModule(t, "example.com/constructoroneof", constructorOneOfAllTransportsDSL)
	output, err := testingx.RunCmd(dir, "go", "vet", "./...")
	require.NoError(t, err, output)
}

func constructorOneOfAllTransportsDSL() {
	dsl.API("constructoroneof", func() {
		dsl.JSONRPC(func() {})
	})
	var Leaf = dsl.Type("Leaf", func() {
		dsl.Field(1, "name", dsl.String)
	})
	var Other = dsl.Type("Other", func() {
		dsl.Field(1, "count", dsl.Int)
	})
	var Extra = dsl.Type("Extra", func() {
		dsl.Field(1, "flag", dsl.Boolean)
	})
	var Holder = dsl.Type("Holder", func() {
		dsl.Field(1, "label", dsl.String)
		dsl.Field(2, "inner", dsl.OneOf(Leaf, Other))
	})
	var Envelope = dsl.Type("Envelope", func() {
		dsl.Field(1, "id", dsl.String)
		dsl.Field(2, "pick", dsl.OneOf(Leaf, Other))
		dsl.Field(4, "must", dsl.OneOf(Extra, Holder))
		dsl.Field(6, "holders", dsl.ArrayOf(Holder))
		dsl.Required("must")
	})
	dsl.Service("pickunion", func() {
		dsl.Method("echo", func() {
			dsl.Payload(Envelope)
			dsl.Result(Envelope)
			dsl.HTTP(func() {
				dsl.POST("/echo")
			})
			dsl.GRPC(func() {})
		})
	})
	dsl.Service("pickunionrpc", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
		})
		dsl.Method("echo", func() {
			dsl.Payload(Envelope)
			dsl.Result(Envelope)
			dsl.JSONRPC(func() {})
		})
	})
}
