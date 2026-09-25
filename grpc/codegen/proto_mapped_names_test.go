package codegen

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/grpc/codegen/testdata"
)

// TestProtoFilesMappedNames checks that the protocol buffer fields and oneofs
// of attributes declared with a mapping suffix, such as "n:m", are named after
// the part that precedes the colon, and that protoc accepts the proto file.
func TestProtoFilesMappedNames(t *testing.T) {
	code := protoFileCode(t, testdata.MappedNamesDSL)

	assert.Contains(t, code, "message EchoRequest {\n\toptional string n = 1;\n\tsint64 req = 2;\n\toptional sint64 def = 3;\n\toneof pick {\n\t\tstring string_ = 4;\n\t\tsint64 int = 5;\n\t}\n\tLeaf obj = 6;\n\trepeated string list = 7;\n\tmap<string, Leaf> index = 8;\n\toneof choice {\n\t\tstring text = 9;\n\t\tLeaf leaf_branch = 10;\n\t}\n}")
	assert.Contains(t, code, "message Leaf {\n\toptional string leaf = 1;\n\tsint64 count = 2;\n}")
	assert.Contains(t, code, "message StreamRequest {\n\tstring id = 1;\n}")
	assert.NotContains(t, code, ":")
	fpath := codegen.CreateTempFile(t, code)
	assert.NoError(t, protoc(defaultProtocCmd, fpath, nil), "compile proto file %q", fpath)
}

// TestGeneratedMappedNamesRoundTrip compiles a generated module whose
// messages, nested types, unions and streaming messages have attributes
// declared with a mapping suffix. It round-trips the service values through
// the generated protobuf conversions, checks that required attributes are
// values and optional ones pointers, that defaults apply and that the
// generated validation reports missing and invalid fields.
func TestGeneratedMappedNamesRoundTrip(t *testing.T) {
	runGeneratedRoundTrip(t, "example.com/grpcmappednames", testdata.MappedNamesDSL, mappedNamesRoundTripHarness)
}

const mappedNamesRoundTripHarness = `package roundtrip

import (
	"testing"

	"github.com/stretchr/testify/require"

	"%[1]s/gen/grpc/mappednames/client"
	pb "%[1]s/gen/grpc/mappednames/pb"
	"%[1]s/gen/grpc/mappednames/server"
	mappednames "%[1]s/gen/mappednames"
)

func envelopes() map[string]*mappednames.Envelope {
	name := "name"
	leaf := "leaf"
	cases := map[string]*mappednames.Envelope{}
	full := &mappednames.Envelope{
		N:     &name,
		Req:   7,
		Def:   5,
		Obj:   &mappednames.Leaf{Leaf: &leaf, Count: 2},
		List:  []string{"a", "b"},
		Index: map[string]*mappednames.Leaf{"k": {Count: 3}},
	}
	full.Pick.SetString("picked")
	full.Choice = &mappednames.Choice2{}
	full.Choice.SetText("text")
	cases["full"] = full
	minimal := &mappednames.Envelope{Def: 3, Obj: &mappednames.Leaf{}}
	minimal.Pick.SetInt(4)
	cases["minimal"] = minimal
	branch := &mappednames.Envelope{Def: 3, Obj: &mappednames.Leaf{}}
	branch.Pick.SetInt(1)
	branch.Choice = &mappednames.Choice2{}
	branch.Choice.SetLeafBranch(&mappednames.Leaf{Leaf: &leaf, Count: 9})
	cases["leaf branch"] = branch
	return cases
}

func TestRoundTrip(t *testing.T) {
	for name, envelope := range envelopes() {
		t.Run(name, func(t *testing.T) {
			var payload, result, streamed *mappednames.Envelope
			require.NotPanics(t, func() {
				payload = server.NewEchoPayload(client.NewProtoEchoRequest(envelope))
				result = client.NewEchoResult(server.NewProtoEchoResponse(envelope))
				streamed = client.NewStreamResponseEnvelope(server.NewProtoEnvelopeStreamResponse(envelope))
			})
			require.Equal(t, envelope, payload)
			require.Equal(t, envelope, result)
			require.Equal(t, envelope, streamed)
			require.NoError(t, server.ValidateEchoRequest(client.NewProtoEchoRequest(envelope)))
			require.NoError(t, client.ValidateEchoResponse(server.NewProtoEchoResponse(envelope)))
		})
	}
}

func TestStreamingPayload(t *testing.T) {
	leaf := "leaf"
	value := &mappednames.Leaf{Leaf: &leaf, Count: 2}
	require.Equal(t, value, server.NewStreamStreamItemLeaf(client.NewProtoLeafStreamStreamItem(value)))
	id := "id"
	require.Equal(t, &mappednames.StreamPayload{ID: id}, server.NewStreamPayload(client.NewProtoStreamRequest(&mappednames.StreamPayload{ID: id})))
}

func TestProtoMessage(t *testing.T) {
	name := "name"
	message := client.NewProtoEchoRequest(envelopes()["full"])
	require.Equal(t, &name, message.N)
	require.Equal(t, int64(7), message.Req)
	require.Equal(t, int64(5), message.GetDef())
	require.Equal(t, "picked", message.GetString_())
	require.Equal(t, int64(2), message.GetObj().GetCount())
	require.Equal(t, []string{"a", "b"}, message.GetList())
	require.Equal(t, int64(3), message.GetIndex()["k"].GetCount())
	require.Equal(t, "text", message.GetText())

	payload := server.NewEchoPayload(&pb.EchoRequest{Pick: &pb.EchoRequest_Int{Int: 1}, Obj: &pb.Leaf{}})
	require.Equal(t, 3, payload.Def, "the default applies to an unset field")
}

func TestValidation(t *testing.T) {
	short := "x"
	require.Error(t, server.ValidateEchoRequest(&pb.EchoRequest{Obj: &pb.Leaf{}}), "pick is required")
	require.Error(t, server.ValidateEchoRequest(&pb.EchoRequest{Pick: &pb.EchoRequest_Int{Int: 1}}), "obj is required")
	require.Error(t, server.ValidateEchoRequest(&pb.EchoRequest{Pick: &pb.EchoRequest_Int{Int: 1}, Obj: &pb.Leaf{}, N: &short}), "n is too short")
	require.NoError(t, server.ValidateEchoRequest(&pb.EchoRequest{Pick: &pb.EchoRequest_Int{Int: 1}, Obj: &pb.Leaf{}}))
}
`
