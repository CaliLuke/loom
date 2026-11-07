package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestUntaggedRootKindsProtobufProjection(t *testing.T) {
	for _, kind := range []string{"scalar", "alias", "any", "array", "map", "nested", "nullable"} {
		t.Run(kind, func(t *testing.T) {
			design := func() {
				var first expr.DataType
				switch kind {
				case "alias":
					first = Type("Code", String)
				case "any":
					first = Any
				case "array":
					first = ArrayOf(Int)
				case "map":
					first = MapOf(String, Int)
				case "nested":
					first = OneOf(func() {
						Untagged()
						Field(1, "Number", Int)
						Field(2, "Text", String)
					})
				default:
					first = Int
				}
				choice := Type("Choice", OneOf(func() {
					Untagged()
					Field(1, "First", first, func() {
						if kind == "nullable" {
							Nullable()
						}
					})
					Field(2, "Second", Boolean)
				}))
				Service("roots", func() {
					Method("read", func() {
						Result(func() {
							Field(1, "choice", choice)
						})
						GRPC(func() {
						})
					})
				})
			}
			if kind == "nullable" {
				err := expr.RunInvalidDSL(t, design)
				require.ErrorContains(t, err, "gRPC response results do not support nullable attributes")
				return
			}
			source := protoFileCode(t, design)
			require.Contains(t, source, "oneof choice")
			require.Contains(t, source, "first = 1")
			require.Contains(t, source, "second = 2")
			require.NoError(t, protoc(defaultProtocCmd, codegen.CreateTempFile(t, source), nil))
		})
	}
}
