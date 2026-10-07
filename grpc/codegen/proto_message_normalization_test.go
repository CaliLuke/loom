package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestProtoMessageNameNormalization(t *testing.T) {
	root := RunGRPCDSL(t, func() {
		leaf := Type("Leaf", func() {
			Meta("struct:name:proto", "leaf_message")
			Field(1, "value", String)
		})
		Type("List", ArrayOf(leaf), func() {
			Meta("struct:name:proto", "list_message")
		})
		index := Type("Index", MapOf(String, leaf), func() {
			Meta("struct:name:proto", "index_message")
		})
		Type("Alias", index, func() {
			Meta("struct:name:proto", "alias_message")
		})
		Type("Choice", OneOf(leaf, String), func() {
			Meta("struct:name:proto", "choice_message")
		})
		Type("Scalar", String, func() {
			Meta("struct:name:proto", "scalar_message")
		})
		Service("svc", func() {
			Method("m", func() {
				Payload(leaf)
				GRPC(func() {})
			})
		})
	})
	for _, tc := range []struct {
		typeName, protoName string
	}{
		{"Leaf", "leaf_message"},
		{"List", "list_message"},
		{"Index", "index_message"},
		{"Alias", "alias_message"},
		{"Choice", "choice_message"},
		{"Scalar", "scalar_message"},
	} {
		t.Run(tc.typeName, func(t *testing.T) {
			source := &expr.AttributeExpr{Type: root.UserType(tc.typeName)}
			sd := &ServiceData{Name: "Svc", Scope: codegen.NewNameScope()}
			message := makeProtoBufMessage(source, "Generated", sd)
			require.Equal(t, tc.protoName, protoBufMessageName(message, sd.Scope))
			require.Equal(t, "*pb."+protoGoName(tc.protoName), protoBufGoFullTypeRef(message, "pb", sd.Scope))
			require.Nil(t, source.Meta, "normalization must not mutate the source occurrence")
			message.Meta["struct:name:proto"][0] = "Changed"
			require.Equal(t, tc.protoName, root.UserType(tc.typeName).Attribute().Meta["struct:name:proto"][0],
				"normalized names must own their metadata")
		})
	}
}

func TestProtoMessageShapeIncludesReferencedNames(t *testing.T) {
	root := RunGRPCDSL(t, func() {
		leaf := Type("Leaf", func() {
			Field(1, "x", String)
		})
		parent := func(name string) expr.UserType {
			return Type(name, func() {
				Meta("struct:name:proto", "Shared")
				Field(1, "first", leaf)
				Field(2, "child", leaf, func() {
					Meta("struct:name:proto", name+"Child")
				})
			})
		}
		a, b := parent("A"), parent("B")
		Service("svc", func() {
			Method("m", func() {
				Payload(a)
				Result(b)
				GRPC(func() {})
			})
		})
	})
	err := generationError(func() {
		ProtoFiles("gen", CreateGRPCServices(root))
	})
	require.ErrorContains(t, err, "different fields")
}
