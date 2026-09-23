package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/testutil"
	"github.com/CaliLuke/loom/grpc/codegen/testdata"
)

// TestProtoFilesInlineObjectFields checks that fields typed with anonymous
// inline objects, including an object that holds only an anonymous OneOf,
// nested objects, and array elements and map values, refer to generated
// messages that protoc accepts.
func TestProtoFilesInlineObjectFields(t *testing.T) {
	root := RunGRPCDSL(t, testdata.InlineObjectFieldsDSL)
	fs := ProtoFiles("", CreateGRPCServices(root))
	require.Len(t, fs, 1)
	sections := fs[0].AllSections()
	require.GreaterOrEqual(t, len(sections), 3)
	code := sectionCode(t, sections[1:]...)

	assert.NotContains(t, code, "} name = 5;")
	assert.Contains(t, code, "\tUnaryRequestName name = 5;")
	assert.Contains(t, code, "message UnaryRequestName {\n\toneof choice {")
	assert.Contains(t, code, "\t\tChoicePair pair = 3;")
	assert.Contains(t, code, "\tUnaryRequestOuter outer = 6;")
	assert.Contains(t, code, "\tUnaryRequestOuterInner2 inner = 2;")
	assert.Contains(t, code, "\trepeated UnaryRequestItems items = 7;")
	assert.Contains(t, code, "\tmap<string, UnaryRequestIndex> index = 8;")
	assert.Contains(t, code, "\tUnaryRequestOuterInner3 outer_inner = 9;")
	assert.Contains(t, code, "message UnaryRequestOuterInner3 {\n\toptional string q = 1;\n}")
	assert.Contains(t, code, "\tUnaryRequestOuterInner named = 10;")
	assert.Contains(t, code, "message UnaryRequestOuterInner {\n\toptional string y = 1;\n}")
	testutil.AssertString(t, "testdata/golden/proto_protofiles-inline-object-fields.proto.golden", code)
	fpath := codegen.CreateTempFile(t, code)
	assert.NoError(t, protoc(defaultProtocCmd, fpath, nil), "compile proto file %q", fpath)
}

// TestInlineObjectFieldsGeneratedModuleRoundTrips checks that the Go
// conversion code between the service types and the protoc-gen-go types of
// inline object fields compiles and preserves every field in both
// directions.
func TestInlineObjectFieldsGeneratedModuleRoundTrips(t *testing.T) {
	root := RunGRPCDSL(t, testdata.InlineObjectFieldsDSL)
	dir := t.TempDir()
	renderGRPCResponseContractModule(t, dir, "example.com/inlineobject", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "inline_object_test.go"), []byte(inlineObjectRoundTripHarness), 0o600))
	runGRPCGoCommand(t, dir, "mod", "tidy")
	runGRPCGoCommand(t, dir, "vet", "./...")
	runGRPCGoCommand(t, dir, "test", ".")
}

// TestRecursiveInlineObjectGeneratedModuleBuilds checks that a user type that
// reaches itself through an inline object field generates finite conversion
// code that compiles: the message generated for the inline object is
// converted inline, so the cycle goes through a transform helper.
func TestRecursiveInlineObjectGeneratedModuleBuilds(t *testing.T) {
	root := RunGRPCDSL(t, inlineObjectRecursiveGRPCDSL)
	dir := t.TempDir()
	renderGRPCResponseContractModule(t, dir, "example.com/orchard", root)
	runGRPCGoCommand(t, dir, "mod", "tidy")
	runGRPCGoCommand(t, dir, "vet", "./...")
}

const inlineObjectRoundTripHarness = `package inlineobject

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"example.com/inlineobject/gen/grpc/inline_object_fields/client"
	pb "example.com/inlineobject/gen/grpc/inline_object_fields/pb"
	"example.com/inlineobject/gen/grpc/inline_object_fields/server"
)

func TestInlineObjectRoundTrip(t *testing.T) {
	str := func(v string) *string { return &v }
	count := int64(3)
	seven := int64(7)
	request := &pb.UnaryRequest{
		Id:    str("id"),
		Name:  &pb.UnaryRequestName{Choice: &pb.UnaryRequestName_Pair{Pair: &pb.ChoicePair{Key: str("k"), Value: str("v")}}},
		Outer: &pb.UnaryRequestOuter{Label: str("label"), Inner: &pb.UnaryRequestOuterInner2{Depth: 4}},
		Items: []*pb.UnaryRequestItems{{Key: str("a")}, {Key: str("b")}},
		Index: map[string]*pb.UnaryRequestIndex{"x": {Count: &count}},
		OuterInner: &pb.UnaryRequestOuterInner3{Q: str("q")},
		Named:      &pb.UnaryRequestOuterInner{Y: str("y")},
		Grid:       []*pb.ArrayOfUnaryRequestGrid{{Field: []*pb.UnaryRequestGrid{{Cell: str("c")}}}},
		Groups:     map[string]*pb.ArrayOfUnaryRequestGroups{"g": {Field: []*pb.UnaryRequestGroups{{Member: str("m")}}}},
		Rows:       []*pb.UnaryRequestRows{{M: &pb.UnaryRequestRowsM{Z: &seven}}},
	}

	payload := server.NewUnaryPayload(request)
	pair, ok := payload.Name.Choice.AsPair()
	require.True(t, ok)
	require.Equal(t, "k", *pair.Key)
	require.Equal(t, "label", *payload.Outer.Label)
	require.Equal(t, 4, payload.Outer.Inner.Depth)
	require.Len(t, payload.Items, 2)
	require.Equal(t, "b", *payload.Items[1].Key)
	require.Equal(t, 3, *payload.Index["x"].Count)
	require.Equal(t, "q", *payload.OuterInner.Q)
	require.Equal(t, "y", *payload.Named.Y)
	require.Equal(t, "c", *payload.Grid[0][0].Cell)
	require.Equal(t, "m", *payload.Groups["g"][0].Member)
	require.Equal(t, 7, payload.Rows[0].M.Z)

	response := server.NewProtoUnaryResponse(payload)
	result := client.NewUnaryResult(response)
	require.True(t, proto.Equal(request, client.NewProtoUnaryRequest(result)))

	defaulted := server.NewUnaryPayload(&pb.UnaryRequest{
		Outer: &pb.UnaryRequestOuter{},
		Rows:  []*pb.UnaryRequestRows{{M: &pb.UnaryRequestRowsM{}}},
	})
	require.Equal(t, 3, defaulted.Rows[0].M.Z)
}
`
