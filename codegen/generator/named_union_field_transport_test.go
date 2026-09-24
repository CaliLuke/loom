package generator

import (
	"testing"

	"github.com/stretchr/testify/require"

	dsl "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/internal/testingx"
)

// TestNamedUnionFieldGeneratedCodeCompiles builds and vets the service and
// transport packages generated for a named union passed to Field. On HTTP,
// gRPC and JSON-RPC the union is an optional field of the body and directly
// the payload and result. On gRPC it is also a required field, a field of
// nested types and the single required field "field" of a type, whose
// message must not be mistaken for the wrapper of a union payload.
func TestNamedUnionFieldGeneratedCodeCompiles(t *testing.T) {
	dir := buildGeneratedModule(t, "example.com/namedunionfield", namedUnionFieldAllTransportsDSL)
	output, err := testingx.RunCmd(dir, "go", "vet", "./...")
	require.NoError(t, err, output)
}

func namedUnionFieldAllTransportsDSL() {
	dsl.API("namedunionfield", func() {
		dsl.JSONRPC(func() {})
	})
	var Leaf = dsl.Type("Leaf", func() {
		dsl.Field(1, "name", dsl.String)
	})
	var Other = dsl.Type("Other", func() {
		dsl.Field(1, "count", dsl.Int)
	})
	var Choice = dsl.Type("Choice", dsl.OneOf(Leaf, Other))
	var Holder = dsl.Type("Holder", func() {
		dsl.Field(1, "label", dsl.String)
		dsl.Field(2, "choice", Choice)
	})
	var Box = dsl.Type("Box", func() {
		dsl.Field(1, "field", Choice)
		dsl.Required("field")
	})
	var Envelope = dsl.Type("Envelope", func() {
		dsl.Field(1, "id", dsl.String)
		dsl.Field(2, "choice", Choice)
		dsl.Field(4, "holder", Holder)
		dsl.Field(5, "holders", dsl.ArrayOf(Holder))
		dsl.Field(6, "box", Box)
		dsl.Required("choice")
	})
	dsl.Service("namedunion", func() {
		dsl.Method("echo", func() {
			dsl.Payload(Holder)
			dsl.Result(Holder)
			dsl.HTTP(func() {
				dsl.POST("/echo")
			})
			dsl.GRPC(func() {})
		})
		dsl.Method("named", func() {
			dsl.Payload(Choice)
			dsl.Result(Choice)
			dsl.HTTP(func() {
				dsl.POST("/named")
			})
			dsl.GRPC(func() {})
		})
	})
	dsl.Service("namedunionnested", func() {
		dsl.Method("echo", func() {
			dsl.Payload(Envelope)
			dsl.Result(Envelope)
			dsl.GRPC(func() {})
		})
		dsl.Method("box", func() {
			dsl.Payload(Box)
			dsl.Result(Box)
			dsl.GRPC(func() {})
		})
	})
	dsl.Service("namedunionrpc", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
		})
		dsl.Method("echo", func() {
			dsl.Payload(Holder)
			dsl.Result(Holder)
			dsl.JSONRPC(func() {})
		})
	})
}
