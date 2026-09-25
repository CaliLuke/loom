package codegen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// TestNestedStructMetaCompatibility checks designs whose struct:name:proto
// metadata applies to types used in nested positions, next to other uses of
// the type or to other messages with the same name, and designs with
// customized copies of such types. Loom names only the top-level messages of
// a type after its struct:name:proto metadata, so these designs generate,
// protoc accepts them and the generated modules compile and vet. When the
// metadata only appears on types in nested positions, the generated
// protocol buffer file is the one generated without the metadata.
func TestNestedStructMetaCompatibility(t *testing.T) {
	cases := []struct {
		Name       string
		DSL        func(proto func(string) string)
		NestedOnly bool
	}{
		{"payload fields in metadata next to a nested use", payloadMetadataNestedDSL, false},
		{"customized payload next to a nested use", customizedPayloadNestedDSL, false},
		{"nested types sharing a name", nestedSharedNameDSL, true},
		{"nested type named like another type", nestedTypeNameDSL, true},
		{"nested type named like a generated message", nestedGeneratedNameDSL, true},
		{"nested type with the Go name of another type", nestedGoNameDSL, true},
		{"named union field and union branch", nestedUnionDSL, true},
		{"customized payload and result", sharedCustomizedPayloadResultDSL, false},
		{"customized payload and streaming payload", sharedCustomizedStreamingDSL, false},
		{"customized result type payload and result", sharedCustomizedResultTypeDSL, false},
		{"customized payload and result with an error and another method", sharedCustomizedErrorDSL, false},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			dsl := func() {
				c.DSL(func(name string) string { return name })
			}
			code := protoFileCode(t, dsl)
			fpath := codegen.CreateTempFile(t, code)
			require.NoError(t, protoc(defaultProtocCmd, fpath, nil), "compile proto file %q", fpath)
			if c.NestedOnly {
				withoutMeta := protoFileCode(t, func() {
					c.DSL(func(string) string { return "" })
				})
				assert.Equal(t, withoutMeta, code)
			}
			dir := t.TempDir()
			renderGRPCModule(t, dir, "example.com/nestedmeta", RunGRPCDSL(t, dsl), resolveGRPCLoomSource(t))
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
