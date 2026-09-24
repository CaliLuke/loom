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

// TestProtoFilesNamedUnionField checks that a named union passed to Field
// generates a oneof named after the field in the message that holds it, with
// branches numbered from the field number, that it generates no message for
// the union itself, and that the result is the proto of the same design with
// constructor OneOf unions.
func TestProtoFilesNamedUnionField(t *testing.T) {
	code := protoFileCode(t, testdata.NamedOneOfFieldDSL)

	assert.Contains(t, code, "message EchoRequest {\n\toptional string id = 1;\n\toneof pick {\n\t\tLeaf leaf = 2;\n\t\tOther other = 3;\n\t}\n\toneof must {\n\t\tExtra extra = 4;\n\t\tHolder holder = 5;\n\t}\n\trepeated Holder holders = 6;\n}")
	assert.Contains(t, code, "message Holder {\n\toptional string label = 1;\n\toneof inner {\n\t\tLeaf leaf = 2;\n\t\tOther other = 3;\n\t}\n}")
	assert.NotContains(t, code, "message Pick")
	assert.NotContains(t, code, "message Must")
	assert.NotContains(t, code, "= 0;")
	fpath := codegen.CreateTempFile(t, code)
	assert.NoError(t, protoc(defaultProtocCmd, fpath, nil), "compile proto file %q", fpath)

	assert.Equal(t, protoFileCode(t, testdata.ConstructorOneOfFieldDSL), code)
}

// TestProtoFilesNamedUnionFieldReuse checks that a named union used as fields
// of several messages is a oneof in each of them, numbered from the number of
// each field, and that the same union used directly as a payload and result
// is the message that wraps its oneof.
func TestProtoFilesNamedUnionFieldReuse(t *testing.T) {
	code := protoFileCode(t, testdata.NamedUnionFieldReuseDSL)

	assert.Contains(t, code, "rpc Named (Choice) returns (Choice);")
	assert.Contains(t, code, "message Choice {\n\toneof field {\n\t\tLeaf leaf = 1;\n\t\tOther other = 2;\n\t}\n}")
	assert.Contains(t, code, "message Holder {\n\toptional string label = 1;\n\toneof choice {\n\t\tLeaf leaf = 2;\n\t\tOther other = 3;\n\t}\n}")
	assert.Contains(t, code, "message EchoRequest {\n\toptional string id = 1;\n\toneof choice {\n\t\tLeaf leaf = 3;\n\t\tOther other = 4;\n\t}\n\tHolder holder = 5;\n\trepeated Holder holders = 6;\n}")
	assert.NotContains(t, code, "= 0;")
	testutil.AssertString(t, "testdata/golden/proto_protofiles-named-union-field-reuse.proto.golden", code)
	fpath := codegen.CreateTempFile(t, code)
	assert.NoError(t, protoc(defaultProtocCmd, fpath, nil), "compile proto file %q", fpath)
}

// TestProtoFilesNamedUnionFieldErrors checks that generation fails with an
// error naming the message and the fields when the branches of a named union
// passed to Field collide with another field of a nested message, by number
// or by name, and that a named union wrapped in a type is valid as array
// elements and map values.
func TestProtoFilesNamedUnionFieldErrors(t *testing.T) {
	cases := []struct {
		name     string
		holder   func(choice any)
		payload  func(holder any)
		expected string
	}{
		{"branch number and field", func(choice any) {
			Field(2, "choice", choice)
			Field(3, "x", String)
		}, nil, `field number 3 in attribute "x" of protocol buffer message "Holder" already exists for attribute "choice.Leaf"; a OneOf passed to Field numbers its branches consecutively from the field number`},
		{"same union twice", func(choice any) {
			Field(1, "first", choice)
			Field(3, "second", choice)
		}, nil, `protocol buffer message "Holder" has two fields named "other": branch "Other" of attribute "first" and branch "Other" of attribute "second"`},
		{"wrapped in collections", func(choice any) {
			Field(1, "choice", choice)
		}, func(holder any) {
			Field(1, "holders", ArrayOf(holder))
			Field(2, "by_name", MapOf(String, holder))
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
				choice := Type("Choice", OneOf(other, leaf))
				holder := Type("Holder", func() {
					c.holder(choice)
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
			assert.Contains(t, code, "message Holder {\n\toneof choice {\n\t\tOther other = 1;\n\t\tLeaf leaf = 2;\n\t}\n}")
			fpath := codegen.CreateTempFile(t, code)
			assert.NoError(t, protoc(defaultProtocCmd, fpath, nil), "compile proto file %q", fpath)
		})
	}
}

// TestGeneratedNamedUnionFieldRoundTrip compiles a generated module whose
// messages carry one named union as optional and required fields, directly
// and nested in objects and array elements, and as a direct payload and
// result. It round-trips the service values through the generated protobuf
// conversions with every union unset and set to each branch.
func TestGeneratedNamedUnionFieldRoundTrip(t *testing.T) {
	const modulePath = "example.com/grpcnamedunionfield"
	root := RunGRPCDSL(t, testdata.NamedUnionFieldReuseDSL)
	dir := t.TempDir()
	renderGRPCModule(t, dir, modulePath, root, resolveGRPCLoomSource(t))
	testDir := filepath.Join(dir, "internal", "roundtrip")
	require.NoError(t, os.MkdirAll(testDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(testDir, "roundtrip_test.go"), []byte(namedUnionFieldRoundTripHarness), 0o600))

	runGRPCGoCommand(t, dir, "mod", "tidy")
	runGRPCGoCommand(t, dir, "build", "./...")
	runGRPCGoCommand(t, dir, "vet", "./...")
	runGRPCGoCommand(t, dir, "test", "./internal/roundtrip")
}

var namedUnionFieldRoundTripHarness = fmt.Sprintf(`package roundtrip

import (
	"testing"

	"github.com/stretchr/testify/require"

	reuse "%[1]s/gen/reuse"
	"%[1]s/gen/grpc/reuse/client"
	pb "%[1]s/gen/grpc/reuse/pb"
	"%[1]s/gen/grpc/reuse/server"
)

func choices() map[string]*reuse.Choice {
	name := "leaf"
	count := 42
	leaf := &reuse.Choice{}
	leaf.SetLeaf(&reuse.Leaf{Name: &name})
	other := &reuse.Choice{}
	other.SetOther(&reuse.Other{Count: &count})
	return map[string]*reuse.Choice{"unset": nil, "leaf": leaf, "other": other}
}

func holders() map[string]*reuse.Holder {
	label := "label"
	cases := map[string]*reuse.Holder{}
	for name, choice := range choices() {
		cases[name] = &reuse.Holder{Label: &label, Choice: choice}
	}
	return cases
}

func envelopes() map[string]*reuse.Envelope {
	id := "id"
	cases := map[string]*reuse.Envelope{}
	for name, choice := range choices() {
		if choice == nil {
			continue
		}
		cases["choice/"+name] = &reuse.Envelope{ID: &id, Choice: *choice}
	}
	for name, holder := range holders() {
		envelope := &reuse.Envelope{Holder: holder, Holders: []*reuse.Holder{holder}}
		envelope.Choice.SetOther(&reuse.Other{})
		cases["holder/"+name] = envelope
	}
	return cases
}

func TestPayloadRoundTrip(t *testing.T) {
	for name, envelope := range envelopes() {
		t.Run(name, func(t *testing.T) {
			var decoded *reuse.Envelope
			require.NotPanics(t, func() {
				decoded = server.NewEchoPayload(client.NewProtoEchoRequest(envelope))
			})
			require.Equal(t, envelope, decoded)
		})
	}
}

func TestResultRoundTrip(t *testing.T) {
	for name, holder := range holders() {
		t.Run(name, func(t *testing.T) {
			var decoded *reuse.Holder
			require.NotPanics(t, func() {
				decoded = client.NewEchoResult(server.NewProtoEchoResponse(holder))
			})
			require.Equal(t, holder, decoded)
		})
	}
}

func TestNamedRoundTrip(t *testing.T) {
	for name, choice := range choices() {
		if choice == nil {
			continue
		}
		t.Run(name, func(t *testing.T) {
			var decoded, payload *reuse.Choice
			require.NotPanics(t, func() {
				decoded = client.NewNamedResult(server.NewProtoChoice(choice))
				payload = server.NewNamedPayload(client.NewProtoChoice(choice))
			})
			require.Equal(t, choice, decoded)
			require.Equal(t, choice, payload)
		})
	}
}

func TestProtoOneofFields(t *testing.T) {
	name := "leaf"
	count := int64(42)
	request := &pb.EchoRequest{
		Choice: &pb.EchoRequest_Other{Other: &pb.Other{Count: &count}},
		Holder: &pb.Holder{Choice: &pb.Holder_Leaf{Leaf: &pb.Leaf{Name: &name}}},
	}
	payload := server.NewEchoPayload(request)
	other, ok := payload.Choice.AsOther()
	require.True(t, ok)
	require.Equal(t, 42, *other.Count)
	leaf, ok := payload.Holder.Choice.AsLeaf()
	require.True(t, ok)
	require.Equal(t, "leaf", *leaf.Name)
}
`, "example.com/grpcnamedunionfield")
