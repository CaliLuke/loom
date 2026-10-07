package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// TestNestedStructMetaNames checks explicit naming at every message position.
// Conflicting declarations fail rather than discarding authored names.
func TestNestedStructMetaNames(t *testing.T) {
	cases := []struct {
		Name  string
		DSL   func(proto func(string) string)
		Error string
	}{
		{"payload fields in metadata next to a nested use", payloadMetadataNestedDSL, "conflicting fields"},
		{"customized payload next to a nested use", customizedPayloadNestedDSL, "conflicting fields"},
		{"explicitly named derived payload", distinctDerivedPayloadDSL, ""},
		{"nested types sharing a name", nestedSharedNameDSL, "conflicting fields"},
		{"nested type named like another type", nestedTypeNameDSL, "conflicting fields"},
		{"nested type named like a generated message", nestedGeneratedNameDSL, "conflicting fields"},
		{"nested type with the Go name of another type", nestedGoNameDSL, "both map to Go type"},
		{"named union field and union branch", nestedUnionDSL, ""},
		{"customized payload and result", sharedCustomizedPayloadResultDSL, ""},
		{"customized payload and streaming payload", sharedCustomizedStreamingDSL, ""},
		{"customized result type payload and result", sharedCustomizedResultTypeDSL, ""},
		{"customized payload and result with an error and another method", sharedCustomizedErrorDSL, ""},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			dsl := func() {
				c.DSL(func(name string) string {
					return name
				})
			}
			root := RunGRPCDSL(t, dsl)
			err := generationError(func() {
				ProtoFiles("gen", CreateGRPCServices(root))
			})
			if c.Error != "" {
				require.ErrorContains(t, err, c.Error)
				return
			}
			require.NoError(t, err)
			dir := t.TempDir()
			renderGRPCModule(t, dir, "example.com/nestedmeta", root, resolveGRPCLoomSource(t))
			runGRPCGoCommand(t, dir, "mod", "tidy")
			runGRPCGoCommand(t, dir, "vet", "./...")
		})
	}
}

// metaType declares the type name whose protocol buffer message name is the
// struct:name:proto value proto, if not empty.
func metaType(name, proto string, fields func()) expr.UserType {
	return Type(name, func() {
		if proto != "" {
			Meta("struct:name:proto", proto)
		}
		fields()
	})
}

// xyFields declares the optional string fields x and y.
func xyFields() {
	Field(1, "x", String)
	Field(2, "y", String)
}

func payloadMetadataNestedDSL(proto func(string) string) {
	a := metaType("A", proto("AProto"), xyFields)
	Service("svc", func() {
		Method("m", func() {
			Payload(a)
			Result(func() {
				Field(1, "a", a)
			})
			GRPC(func() {
				Metadata(func() {
					Attribute("x")
				})
			})
		})
	})
}

func customizedPayloadNestedDSL(proto func(string) string) {
	a := metaType("A", proto("AProto"), xyFields)
	Service("svc", func() {
		Method("m", func() {
			Payload(a, func() {
				Required("y")
			})
			Result(func() {
				Field(1, "a", a)
			})
			GRPC(func() {})
		})
	})
}

func nestedSharedNameDSL(proto func(string) string) {
	a := metaType("A", proto("Shared"), func() {
		Field(1, "x", String)
	})
	b := metaType("B", proto("Shared"), func() {
		Field(1, "y", Int)
	})
	Service("svc", func() {
		Method("m", func() {
			Payload(func() {
				Field(1, "a", a)
				Field(2, "b", b)
			})
			Result(String)
			GRPC(func() {})
		})
	})
}

func nestedTypeNameDSL(proto func(string) string) {
	a := metaType("A", proto("B"), func() {
		Field(1, "x", String)
	})
	b := metaType("B", "", func() {
		Field(1, "y", Int)
	})
	Service("svc", func() {
		Method("m", func() {
			Payload(func() {
				Field(1, "a", a)
				Field(2, "b", b)
			})
			Result(String)
			GRPC(func() {})
		})
	})
}

func nestedGeneratedNameDSL(proto func(string) string) {
	a := metaType("A", proto("MResponse"), func() {
		Field(1, "x", String)
	})
	Service("svc", func() {
		Method("m", func() {
			Payload(func() {
				Field(1, "a", a)
			})
			Result(String)
			GRPC(func() {})
		})
	})
}

func nestedGoNameDSL(proto func(string) string) {
	a := metaType("A", proto("node_tree"), func() {
		Field(1, "x", String)
	})
	b := metaType("NodeTree", "", func() {
		Field(1, "y", Int)
	})
	Service("svc", func() {
		Method("m", func() {
			Payload(func() {
				Field(1, "a", a)
				Field(2, "b", b)
			})
			Result(String)
			GRPC(func() {})
		})
	})
}

func nestedUnionDSL(proto func(string) string) {
	leaf := metaType("Leaf", proto("LeafProto"), func() {
		Field(1, "name", String)
	})
	other := metaType("Other", "", func() {
		Field(1, "count", Int)
	})
	choice := Type("Choice", OneOf(leaf, other), func() {
		if name := proto("ChoiceProto"); name != "" {
			Meta("struct:name:proto", name)
		}
	})
	holder := metaType("Holder", "", func() {
		Field(1, "choice", choice)
		Field(3, "leaves", ArrayOf(leaf))
		Field(4, "index", MapOf(String, leaf))
	})
	Service("svc", func() {
		Method("m", func() {
			Payload(OneOf(choice, holder))
			Result(holder)
			GRPC(func() {})
		})
	})
}

// sharedCustomizedType declares the type A with the struct:name:proto name
// proto("AProto") and the optional fields x and y.
func sharedCustomizedType(proto func(string) string) expr.UserType {
	return metaType("A", proto("AProto"), xyFields)
}

func sharedCustomizedPayloadResultDSL(proto func(string) string) {
	a := sharedCustomizedType(proto)
	Service("svc", func() {
		Method("m1", func() {
			Payload(a, func() {
				Required("y")
			})
			Result(a, func() {
				Required("y")
			})
			GRPC(func() {})
		})
	})
}

func sharedCustomizedStreamingDSL(proto func(string) string) {
	a := sharedCustomizedType(proto)
	Service("svc", func() {
		Method("m1", func() {
			Payload(a, func() {
				Required("y")
			})
			StreamingPayload(a, func() {
				Required("y")
			})
			GRPC(func() {})
		})
	})
}

func sharedCustomizedResultTypeDSL(proto func(string) string) {
	rt := ResultType("application/vnd.a", "A", func() {
		Meta("struct:name:proto", proto("AProto"))
		Attributes(xyFields)
	})
	Service("svc", func() {
		Method("m1", func() {
			Payload(rt, func() {
				Required("y")
			})
			Result(rt, func() {
				Required("y")
			})
			GRPC(func() {})
		})
	})
}

func sharedCustomizedErrorDSL(proto func(string) string) {
	a := sharedCustomizedType(proto)
	Service("svc", func() {
		Method("m1", func() {
			Payload(a, func() {
				Required("y")
			})
			Result(a, func() {
				Required("y")
			})
			Error("bad")
			GRPC(func() {
				Response("bad", CodeInvalidArgument)
			})
		})
		Method("m2", func() {
			Payload(String)
			Result(String)
			GRPC(func() {})
		})
	})
}

// distinctDerivedPayloadDSL gives an intentionally distinct contract its own
// explicit name while retaining the original type's nested declaration.
func distinctDerivedPayloadDSL(proto func(string) string) {
	a := metaType("A", proto("AProto"), xyFields)
	required := Type("RequiredA", a, func() {
		Meta("struct:name:proto", proto("RequiredAProto"))
		Required("y")
	})
	Service("svc", func() {
		Method("m", func() {
			Payload(required)
			Result(func() {
				Field(1, "a", a)
			})
			GRPC(func() {})
		})
	})
}
