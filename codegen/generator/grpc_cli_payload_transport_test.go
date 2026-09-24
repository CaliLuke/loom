package generator

import (
	"testing"

	"github.com/stretchr/testify/require"

	dsl "github.com/CaliLuke/loom/dsl"
	grpctestdata "github.com/CaliLuke/loom/grpc/codegen/testdata"
	"github.com/CaliLuke/loom/internal/testingx"
)

// TestGRPCClientCLIPayloadConversionsCompile generates a gRPC service for
// every payload kind whose client CLI payload builder converts the decoded
// protobuf message, runs protoc on the emitted proto file, then builds and
// vets the whole generated tree, including the client CLI packages.
func TestGRPCClientCLIPayloadConversionsCompile(t *testing.T) {
	cases := []struct {
		Name string
		DSL  func()
	}{
		{"any-error-testdata", grpcAnyErrorDSL},
		{"any-field", grpcCLIPayloadDSL(func() any {
			return dsl.Type("Message", func() {
				dsl.Field(1, "value", dsl.Any)
			})
		}, false)},
		{"required-any-field", grpcCLIPayloadDSL(func() any {
			return dsl.Type("Message", func() {
				dsl.Field(1, "value", dsl.Any)
				dsl.Required("value")
			})
		}, false)},
		{"any-nested-in-object", grpcCLIPayloadDSL(func() any {
			inner := dsl.Type("Inner", func() {
				dsl.Field(1, "value", dsl.Any)
			})
			return dsl.Type("Message", func() {
				dsl.Field(1, "inner", inner)
			})
		}, false)},
		{"array-of-any", grpcCLIPayloadDSL(func() any {
			return dsl.Type("Message", func() {
				dsl.Field(1, "values", dsl.ArrayOf(dsl.Any))
			})
		}, false)},
		{"map-of-any", grpcCLIPayloadDSL(func() any {
			return dsl.Type("Message", func() {
				dsl.Field(1, "mapped", dsl.MapOf(dsl.String, dsl.Any))
			})
		}, false)},
		{"array-of-objects-with-any", grpcCLIPayloadDSL(func() any {
			inner := dsl.Type("Inner", func() {
				dsl.Field(1, "value", dsl.Any)
			})
			return dsl.Type("Message", func() {
				dsl.Field(1, "items", dsl.ArrayOf(inner))
			})
		}, false)},
		{"any-alias-field", grpcCLIPayloadDSL(func() any {
			blob := dsl.Type("Blob", dsl.Any)
			return dsl.Type("Message", func() {
				dsl.Field(1, "value", blob)
			})
		}, false)},
		{"bytes-field", grpcCLIPayloadDSL(func() any {
			return dsl.Type("Message", func() {
				dsl.Field(1, "data", dsl.Bytes)
			})
		}, false)},
		{"union-field", grpcCLIPayloadDSL(func() any {
			return dsl.Type("Message", func() {
				dsl.OneOf("pick", func() {
					dsl.Field(1, "text", dsl.String)
					dsl.Field(2, "count", dsl.Int)
				})
			})
		}, false)},
		{"union-field-with-any", grpcCLIPayloadDSL(func() any {
			return dsl.Type("Message", func() {
				dsl.OneOf("pick", func() {
					dsl.Field(1, "text", dsl.String)
					dsl.Field(2, "value", dsl.Any)
				})
			})
		}, false)},
		{"union-field-with-any-object", grpcCLIPayloadDSL(func() any {
			inner := dsl.Type("Inner", func() {
				dsl.Field(1, "value", dsl.Any)
			})
			other := dsl.Type("Other", func() {
				dsl.Field(1, "count", dsl.Int)
			})
			return dsl.Type("Message", func() {
				dsl.Field(1, "pick", dsl.OneOf(inner, other))
			})
		}, false)},
		{"top-level-union-of-any-objects", grpcCLIPayloadDSL(func() any {
			inner := dsl.Type("Inner", func() {
				dsl.Field(1, "value", dsl.Any)
			})
			other := dsl.Type("Other", func() {
				dsl.Field(1, "count", dsl.Int)
			})
			return dsl.OneOf(inner, other)
		}, false)},
		{"top-level-union-of-string-and-any-aliases", grpcCLIPayloadDSL(func() any {
			label := dsl.Type("Label", dsl.String)
			blob := dsl.Type("Blob", dsl.Any)
			return dsl.OneOf(label, blob)
		}, false)},
		{"top-level-union-of-string-and-any", grpcCLIPayloadDSL(func() any {
			return dsl.OneOf(dsl.String, dsl.Any)
		}, false)},
		{"union-field-of-string-and-any", grpcCLIPayloadDSL(func() any {
			return dsl.Type("Message", func() {
				dsl.Field(1, "pick", dsl.OneOf(dsl.String, dsl.Any))
			})
		}, false)},
		{"streaming-any-field", grpcCLIPayloadDSL(func() any {
			return dsl.Type("Message", func() {
				dsl.Field(1, "value", dsl.Any)
			})
		}, true)},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			dir := buildGeneratedModule(t, "example.com/grpcclicli", c.DSL)
			output, err := testingx.RunCmd(dir, "go", "vet", "./...")
			require.NoError(t, err, output)
		})
	}
}

// grpcCLIPayloadDSL returns a design with one gRPC method whose payload is the
// type built by payload. When stream is true the method also declares a
// streaming payload of the same type.
func grpcCLIPayloadDSL(payload func() any, stream bool) func() {
	return func() {
		dsl.API("grpcclicli", func() {})
		p := payload()
		dsl.Service("converter", func() {
			dsl.Method("convert", func() {
				dsl.Payload(p)
				if stream {
					dsl.StreamingPayload(p)
				}
				dsl.Result(dsl.String)
				dsl.GRPC(func() {})
			})
		})
	}
}

// grpcAnyErrorDSL exposes the gRPC AnyErrorDSL fixture, which converts Any
// values in unary and streaming payloads, results and errors, under an API.
func grpcAnyErrorDSL() {
	dsl.API("grpcclicli", func() {})
	grpctestdata.AnyErrorDSL()
}
