package codegen

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/testutil"
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/grpc/codegen/testdata"
)

// TestProtoFilesConstructorOneOfField checks that a constructor OneOf passed
// to Field generates a oneof named after the field whose branches are
// numbered consecutively from the field number, that protoc accepts the
// result, and that it is the proto of the equivalent block form of OneOf.
func TestProtoFilesConstructorOneOfField(t *testing.T) {
	code := protoFileCode(t, testdata.ConstructorOneOfFieldDSL)

	assert.Contains(t, code, "message EchoRequest {\n\toptional string id = 1;\n\toneof pick {\n\t\tLeaf leaf = 2;\n\t\tOther other = 3;\n\t}\n\toneof must {\n\t\tExtra extra = 4;\n\t\tHolder holder = 5;\n\t}\n\trepeated Holder holders = 6;\n}")
	assert.Contains(t, code, "message Holder {\n\toptional string label = 1;\n\toneof inner {\n\t\tLeaf leaf = 2;\n\t\tOther other = 3;\n\t}\n}")
	assert.NotContains(t, code, "= 0;")
	assert.NotContains(t, code, "leaf_or_other")
	testutil.AssertString(t, "testdata/golden/proto_protofiles-constructor-oneof-field.proto.golden", code)
	fpath := codegen.CreateTempFile(t, code)
	assert.NoError(t, protoc(defaultProtocCmd, fpath, nil), "compile proto file %q", fpath)

	assert.Equal(t, protoFileCode(t, testdata.BlockOneOfFieldDSL), code)
}

// TestProtoFilesDuplicateUnionBranchNames checks that generation fails with
// an error naming the message and the fields when union branches, whose
// constructor form derives their names from the branch types, give two
// fields of the same message the same protocol buffer name.
func TestProtoFilesDuplicateUnionBranchNames(t *testing.T) {
	cases := []struct {
		name     string
		fields   func(leaf, other any)
		expected string
	}{
		{"two constructor unions", func(leaf, other any) {
			Field(2, "pick", OneOf(leaf, other))
			Field(4, "alt", OneOf(leaf, other))
		}, `protocol buffer message "EchoRequest" has two fields named "leaf": branch "Leaf" of attribute "pick" and branch "Leaf" of attribute "alt"`},
		{"branch and field", func(leaf, other any) {
			Field(1, "leaf", String)
			Field(2, "pick", OneOf(leaf, other))
		}, `protocol buffer message "EchoRequest" has two fields named "leaf": attribute "leaf" and branch "Leaf" of attribute "pick"`},
		{"block unions", func(_, _ any) {
			OneOf("pick", func() {
				Field(1, "text", String)
			})
			OneOf("alt", func() {
				Field(2, "text", String)
			})
		}, `protocol buffer message "EchoRequest" has two fields named "text": branch "text" of attribute "pick" and branch "text" of attribute "alt"`},
		{"distinct names", func(leaf, other any) {
			Field(2, "pick", OneOf(leaf, other))
			Field(4, "next", String)
		}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := RunGRPCDSL(t, func() {
				leaf := Type("Leaf", func() {
					Field(1, "name", String)
				})
				other := Type("Other", func() {
					Field(1, "count", Int)
				})
				Service("pickunion", func() {
					Method("echo", func() {
						Payload(func() {
							c.fields(leaf, other)
						})
						GRPC(func() {})
					})
				})
			})
			err := generationError(func() { ProtoFiles("", CreateGRPCServices(root)) })
			if c.expected == "" {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), c.expected)
		})
	}
}

// TestProtoFilesNestedFieldNumbers checks that generation fails with an
// error naming the message and the fields when a message reached from a
// request message, a user type or an inline object, has a field or union
// branch without a field number or two fields with the same number, and that
// valid nested messages still generate.
func TestProtoFilesNestedFieldNumbers(t *testing.T) {
	const hint = "; a OneOf passed to Field numbers its branches consecutively from the field number"
	cases := []struct {
		name     string
		holder   func(leaf, other any)
		payload  func(holder any)
		expected string
	}{
		{"constructor union branch and field", func(leaf, other any) {
			Field(2, "inner", OneOf(leaf, other))
			Field(3, "x", String)
		}, nil, `field number 3 in attribute "x" of protocol buffer message "Holder" already exists for attribute "inner.Other"` + hint},
		{"two fields", func(_, _ any) {
			Field(1, "a", String)
			Field(1, "b", String)
		}, nil, `field number 1 in attribute "b" of protocol buffer message "Holder" already exists for attribute "a"`},
		{"untagged block branch", func(leaf, other any) {
			OneOf("inner", func() {
				Attribute("Leaf", leaf)
				Field(3, "Other", other)
			})
		}, nil, `union branch "Leaf" of attribute "inner" of protocol buffer message "Holder" has no field number`},
		{"untagged field", func(_, _ any) {
			Field(1, "a", String)
			Attribute("b", String)
		}, nil, `attribute "b" of protocol buffer message "Holder" has no field number`},
		{"inline object", nil, func(_ any) {
			Field(1, "prefs", func() {
				Field(1, "a", String)
				Field(1, "b", String)
			})
		}, `field number 1 in attribute "b" of protocol buffer message "EchoRequestPrefs" already exists for attribute "a"`},
		{"valid", func(leaf, other any) {
			Field(1, "label", String)
			Field(2, "inner", OneOf(leaf, other))
			Field(4, "x", String)
			OneOf("alt", func() {
				Field(5, "text", String)
				Field(6, "count", Int)
			})
		}, func(holder any) {
			Field(1, "holder", holder)
			Field(2, "holders", ArrayOf(holder))
			Field(3, "prefs", func() {
				Field(1, "a", String)
				Field(2, "b", String)
			})
		}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := RunGRPCDSL(t, func() {
				leaf := Type("Leaf", func() {
					Field(1, "name", String)
				})
				other := Type("Other", func() {
					Field(1, "count", Int)
				})
				holder := Type("Holder", func() {
					if c.holder != nil {
						c.holder(leaf, other)
						return
					}
					Field(1, "label", String)
				})
				Service("nested", func() {
					Method("echo", func() {
						Payload(func() {
							if c.payload != nil {
								c.payload(holder)
								return
							}
							Field(1, "holder", holder)
						})
						GRPC(func() {})
					})
				})
			})
			var code string
			err := generationError(func() {
				fs := ProtoFiles("", CreateGRPCServices(root))
				require.Len(t, fs, 1)
				code = sectionCode(t, fs[0].AllSections()[1:]...)
			})
			if c.expected != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), c.expected)
				return
			}
			require.NoError(t, err)
			assert.NotContains(t, code, "= 0;")
			fpath := codegen.CreateTempFile(t, code)
			assert.NoError(t, protoc(defaultProtocCmd, fpath, nil), "compile proto file %q", fpath)
		})
	}
}

// TestGeneratedConstructorOneOfFieldRoundTrip compiles a generated module
// whose payload and result carry constructor OneOf unions passed to Field,
// optional and required, directly and nested in objects and array elements.
// It round-trips the service values through the generated protobuf
// conversions with every union unset and set to each branch.
func TestGeneratedConstructorOneOfFieldRoundTrip(t *testing.T) {
	const modulePath = "example.com/grpcconstructoroneof"
	root := RunGRPCDSL(t, testdata.ConstructorOneOfFieldDSL)
	dir := t.TempDir()
	renderGRPCModule(t, dir, modulePath, root, resolveGRPCLoomSource(t))
	testDir := filepath.Join(dir, "internal", "roundtrip")
	require.NoError(t, os.MkdirAll(testDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(testDir, "roundtrip_test.go"), []byte(constructorOneOfRoundTripHarness), 0o600))

	runGRPCGoCommand(t, dir, "mod", "tidy")
	runGRPCGoCommand(t, dir, "build", "./...")
	runGRPCGoCommand(t, dir, "vet", "./...")
	runGRPCGoCommand(t, dir, "test", "./internal/roundtrip")
}

// protoFileCode returns the code of the protocol buffer file generated for
// the single service of dsl, without its header.
func protoFileCode(t *testing.T, dsl func()) string {
	t.Helper()
	root := RunGRPCDSL(t, dsl)
	fs := ProtoFiles("", CreateGRPCServices(root))
	require.Len(t, fs, 1)
	sections := fs[0].AllSections()
	require.GreaterOrEqual(t, len(sections), 3)
	return sectionCode(t, sections[1:]...)
}

var constructorOneOfRoundTripHarness = fmt.Sprintf(`package roundtrip

import (
	"testing"

	"github.com/stretchr/testify/require"

	pickunion "%[1]s/gen/pickunion"
	"%[1]s/gen/grpc/pickunion/client"
	pb "%[1]s/gen/grpc/pickunion/pb"
	"%[1]s/gen/grpc/pickunion/server"
)

func leafOrOthers() map[string]*pickunion.LeafOrOther {
	name := "leaf"
	count := 42
	leaf := &pickunion.LeafOrOther{}
	leaf.SetLeaf(&pickunion.Leaf{Name: &name})
	other := &pickunion.LeafOrOther{}
	other.SetOther(&pickunion.Other{Count: &count})
	return map[string]*pickunion.LeafOrOther{"unset": nil, "leaf": leaf, "other": other}
}

func envelopes() map[string]*pickunion.Envelope {
	id := "id"
	flag := true
	label := "label"
	cases := map[string]*pickunion.Envelope{}
	for name, pick := range leafOrOthers() {
		envelope := &pickunion.Envelope{ID: &id, Pick: pick}
		envelope.Must.SetExtra(&pickunion.Extra{Flag: &flag})
		cases["pick/"+name] = envelope
	}
	for name, inner := range leafOrOthers() {
		holder := &pickunion.Holder{Label: &label, Inner: inner}
		envelope := &pickunion.Envelope{Holders: []*pickunion.Holder{holder}}
		envelope.Must.SetHolder(holder)
		cases["holder/"+name] = envelope
	}
	return cases
}

func TestPayloadRoundTrip(t *testing.T) {
	for name, envelope := range envelopes() {
		t.Run(name, func(t *testing.T) {
			var decoded *pickunion.Envelope
			require.NotPanics(t, func() {
				decoded = server.NewEchoPayload(client.NewProtoEchoRequest(envelope))
			})
			require.Equal(t, envelope, decoded)
		})
	}
}

func TestResultRoundTrip(t *testing.T) {
	for name, envelope := range envelopes() {
		t.Run(name, func(t *testing.T) {
			var decoded *pickunion.Envelope
			require.NotPanics(t, func() {
				decoded = client.NewEchoResult(server.NewProtoEchoResponse(envelope))
			})
			require.Equal(t, envelope, decoded)
		})
	}
}

func TestProtoOneofFields(t *testing.T) {
	name := "leaf"
	count := int64(42)
	request := &pb.EchoRequest{
		Pick: &pb.EchoRequest_Other{Other: &pb.Other{Count: &count}},
		Must: &pb.EchoRequest_Holder{Holder: &pb.Holder{Inner: &pb.Holder_Leaf{Leaf: &pb.Leaf{Name: &name}}}},
	}
	payload := server.NewEchoPayload(request)
	other, ok := payload.Pick.AsOther()
	require.True(t, ok)
	require.Equal(t, 42, *other.Count)
	holder, ok := payload.Must.AsHolder()
	require.True(t, ok)
	leaf, ok := holder.Inner.AsLeaf()
	require.True(t, ok)
	require.Equal(t, "leaf", *leaf.Name)
}
`, "example.com/grpcconstructoroneof")
