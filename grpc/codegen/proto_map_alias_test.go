package codegen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/testutil"
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/grpc/codegen/testdata"
)

// TestProtoFilesMapAlias checks that a named map is the message that wraps
// the map in its "field" attribute wherever it appears: as a message field,
// an array element, a map value and directly as a payload, streaming payload
// and result, and that a named map of a named map wraps the map itself.
func TestProtoFilesMapAlias(t *testing.T) {
	code := protoFileCode(t, testdata.MapAliasDSL)

	assert.Contains(t, code, "rpc Lookup (Index) returns (Index);")
	assert.Contains(t, code, "rpc Extend (More) returns (More);")
	assert.Contains(t, code, "rpc Upload (stream UploadStreamingRequest) returns (Index);")
	assert.Contains(t, code, "rpc UploadMore (stream Index) returns (UploadMoreResponse);")
	assert.Contains(t, code, "message UploadStreamingRequest {\n\tmap<string, sint64> field = 1;\n}")
	assert.Contains(t, code, "message EchoRequest {\n\tIndex index = 1;\n\tIndex required_index = 2;\n\tLeafIndex leaf_index = 3;\n\tTagIndex tag_index = 4;\n\tMore more = 5;\n\trepeated Index indexes = 6;\n\tmap<string, Index> index_by_key = 7;\n\tLimited limited = 8;\n}")
	for _, message := range []string{
		"message Index {\n\tmap<string, sint64> field = 1;\n}",
		"message LeafIndex {\n\tmap<string, Leaf> field = 1;\n}",
		"message TagIndex {\n\tmap<sint64, Tags> field = 1;\n}",
		"message Tags {\n\trepeated string field = 1;\n}",
		"message More {\n\tmap<string, sint64> field = 1;\n}",
		"message Limited {\n\tmap<string, string> field = 1;\n}",
	} {
		assert.Contains(t, code, message)
	}
	assert.NotContains(t, code, "message Indexmap")
	assert.NotContains(t, code, "message MoreIndex")
	testutil.AssertString(t, "testdata/golden/proto_protofiles-map-alias.proto.golden", code)
	fpath := codegen.CreateTempFile(t, code)
	assert.NoError(t, protoc(defaultProtocCmd, fpath, nil), "compile proto file %q", fpath)
}

// TestProtoFilesMapAliasShapes checks that each position of a named map
// alone, with no other position declaring its message, generates the message
// that wraps the map and a proto file that protoc accepts.
func TestProtoFilesMapAliasShapes(t *testing.T) {
	cases := []struct {
		name     string
		method   func(index, more any)
		contains []string
	}{
		{"field", func(index, _ any) {
			Payload(func() {
				Field(1, "index", index)
			})
		}, []string{"\tIndex index = 1;"}},
		{"alias field", func(_, more any) {
			Payload(func() {
				Field(1, "more", more)
			})
		}, []string{"\tMore more = 1;", "message More {\n\tmap<string, sint64> field = 1;\n}"}},
		{"array element", func(index, _ any) {
			Payload(func() {
				Field(1, "indexes", ArrayOf(index))
			})
		}, []string{"\trepeated Index indexes = 1;"}},
		{"map value", func(index, _ any) {
			Payload(func() {
				Field(1, "by_key", MapOf(String, index))
			})
		}, []string{"\tmap<string, Index> by_key = 1;"}},
		{"payload", func(index, _ any) {
			Payload(index)
		}, []string{"rpc Echo (Index) returns (EchoResponse);"}},
		{"result", func(index, _ any) {
			Result(index)
		}, []string{"rpc Echo (EchoRequest) returns (Index);"}},
		{"streaming result", func(index, _ any) {
			StreamingResult(index)
		}, []string{"rpc Echo (EchoRequest) returns (stream Index);"}},
		{"alias payload", func(_, more any) {
			Payload(more)
		}, []string{"rpc Echo (More) returns (EchoResponse);", "message More {\n\tmap<string, sint64> field = 1;\n}"}},
		{"streaming payload", func(index, _ any) {
			StreamingPayload(index)
			Result(index)
		}, []string{"rpc Echo (stream EchoStreamingRequest) returns (Index);", "message EchoStreamingRequest {\n\tmap<string, sint64> field = 1;\n}"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code := protoFileCode(t, func() {
				index := Type("Index", MapOf(String, Int))
				more := Type("More", index)
				Service("shapes", func() {
					Method("echo", func() {
						c.method(index, more)
						GRPC(func() {})
					})
				})
			})
			for _, want := range c.contains {
				assert.Contains(t, code, want)
			}
			if c.name != "alias field" && c.name != "alias payload" {
				assert.Contains(t, code, "message Index {\n\tmap<string, sint64> field = 1;\n}")
			}
			assert.NotContains(t, code, "map<string, sint64>\n")
			fpath := codegen.CreateTempFile(t, code)
			assert.NoError(t, protoc(defaultProtocCmd, fpath, nil), "compile proto file %q", fpath)
		})
	}
}

// TestProtoFilesRecursiveMapMessage checks that generation terminates for a
// type that reaches itself through a named map used as a field, as array
// elements and as map values, and that protoc accepts the proto file.
func TestProtoFilesRecursiveMapMessage(t *testing.T) {
	var code string
	err := generationError(func() {
		code = protoFileCode(t, testdata.RecursiveMapMessageDSL)
	})
	require.NoError(t, err)

	assert.Contains(t, code, "message Node {\n\toptional string id = 1;\n\tNodeIndex kids = 2;\n\trepeated NodeIndex grid = 3;\n\tmap<string, NodeIndex> index = 4;\n}")
	assert.Contains(t, code, "message NodeIndex {\n\tmap<string, Node> field = 1;\n}")
	assert.Contains(t, code, "rpc List (NodeIndex) returns (NodeIndex);")
	assert.Contains(t, code, "rpc Watch (WatchRequest) returns (stream NodeIndex);")
	fpath := codegen.CreateTempFile(t, code)
	assert.NoError(t, protoc(defaultProtocCmd, fpath, nil), "compile proto file %q", fpath)
}

// TestGeneratedMapAliasRoundTrip compiles a generated module whose messages
// carry named maps as optional and required fields, array elements and map
// values, and as a direct payload, streaming payload and result. It
// round-trips the service values through the generated protobuf conversions
// with the maps unset, empty and set, and checks the map validations.
func TestGeneratedMapAliasRoundTrip(t *testing.T) {
	runGeneratedRoundTrip(t, "example.com/grpcmapalias", testdata.MapAliasDSL, mapAliasRoundTripHarness)
}

// TestGeneratedRecursiveMapMessageRoundTrip compiles the module generated for
// a type that reaches itself through a named map used as a field, as array
// elements and as map values, and as a unary and streaming payload and
// result, and round-trips nested values through the generated protobuf
// conversions.
func TestGeneratedRecursiveMapMessageRoundTrip(t *testing.T) {
	runGeneratedRoundTrip(t, "example.com/grpcrecursivemap", testdata.RecursiveMapMessageDSL, recursiveMapMessageRoundTripHarness)
}

const mapAliasRoundTripHarness = `package roundtrip

import (
	"testing"

	"github.com/stretchr/testify/require"

	"%[1]s/gen/grpc/mapalias/client"
	pb "%[1]s/gen/grpc/mapalias/pb"
	"%[1]s/gen/grpc/mapalias/server"
	mapalias "%[1]s/gen/mapalias"
)

func envelopes() map[string]*mapalias.Envelope {
	name := "leaf"
	return map[string]*mapalias.Envelope{
		"full": {
			Index:         mapalias.Index{"a": 1, "b": 2},
			RequiredIndex: mapalias.Index{"required": 3},
			LeafIndex:     mapalias.LeafIndex{"leaf": {Name: &name}, "empty": {}},
			TagIndex:      mapalias.TagIndex{1: {"x", "y"}, 2: {}},
			More:          mapalias.More{"more": 4},
			Indexes:       []mapalias.Index{{"x": 5}, {}, {"y": 6, "z": 7}},
			IndexByKey:    map[string]mapalias.Index{"k": {"v": 8}, "empty": {}},
			Limited:       mapalias.Limited{"a": "x", "b": "y"},
		},
		"empty": {
			Index:         mapalias.Index{},
			RequiredIndex: mapalias.Index{},
			LeafIndex:     mapalias.LeafIndex{},
			TagIndex:      mapalias.TagIndex{},
			More:          mapalias.More{},
		},
		"unset": {RequiredIndex: mapalias.Index{"only": 1}},
	}
}

func TestPayloadRoundTrip(t *testing.T) {
	for name, envelope := range envelopes() {
		t.Run(name, func(t *testing.T) {
			var decoded *mapalias.Envelope
			require.NotPanics(t, func() {
				decoded = server.NewEchoPayload(client.NewProtoEchoRequest(envelope))
			})
			require.Equal(t, envelope, decoded)
			require.NoError(t, server.ValidateEchoRequest(client.NewProtoEchoRequest(envelope)))
		})
	}
}

func TestResultRoundTrip(t *testing.T) {
	for name, envelope := range envelopes() {
		t.Run(name, func(t *testing.T) {
			var decoded *mapalias.Envelope
			require.NotPanics(t, func() {
				decoded = client.NewEchoResult(server.NewProtoEchoResponse(envelope))
			})
			require.Equal(t, envelope, decoded)
		})
	}
}

func TestDirectRoundTrip(t *testing.T) {
	for name, index := range map[string]mapalias.Index{"empty": {}, "set": {"a": 1, "b": 2}} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, index, server.NewLookupPayload(client.NewProtoIndex(index)))
			require.Equal(t, index, client.NewLookupResult(server.NewProtoIndex(index)))
			more := mapalias.More(index)
			require.Equal(t, more, server.NewExtendPayload(client.NewProtoMore(more)))
			require.Equal(t, more, client.NewExtendResult(server.NewProtoMore(more)))
			require.Len(t, client.NewProtoMore(more).GetField(), len(index))
		})
	}
}

func TestStreamingRoundTrip(t *testing.T) {
	for name, index := range map[string]mapalias.Index{"empty": {}, "set": {"a": 1, "b": 2}} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, index, server.NewUploadStreamingRequestUploadStreamingRequest(client.NewProtoIndexUploadStreamingRequest(index)))
			require.Equal(t, index, client.NewIndexIndex(server.NewProtoIndexIndex(index)))
			more := mapalias.More(index)
			require.Equal(t, more, server.NewIndexIndex(client.NewProtoMoreIndex(more)))
		})
	}
}

func TestProtoMessages(t *testing.T) {
	request := client.NewProtoEchoRequest(envelopes()["full"])
	require.Equal(t, map[string]int64{"a": 1, "b": 2}, request.GetIndex().GetField())
	require.Equal(t, map[string]int64{"more": 4}, request.GetMore().GetField())
	require.Equal(t, map[string]int64{"y": 6, "z": 7}, request.GetIndexes()[2].GetField())
	require.Equal(t, map[string]int64{"v": 8}, request.GetIndexByKey()["k"].GetField())
	require.Equal(t, []string{"x", "y"}, request.GetTagIndex().GetField()[1].GetField())
	require.Equal(t, "leaf", request.GetLeafIndex().GetField()["leaf"].GetName())

	payload := server.NewEchoPayload(&pb.EchoRequest{
		RequiredIndex: &pb.Index{Field: map[string]int64{"r": 1}},
		Indexes:       []*pb.Index{{Field: map[string]int64{"w": 2}}},
	})
	require.Equal(t, mapalias.Index{"r": 1}, payload.RequiredIndex)
	require.Equal(t, []mapalias.Index{{"w": 2}}, payload.Indexes)
}

func TestValidation(t *testing.T) {
	for name, limited := range map[string]map[string]string{
		"too long":    {"a": "x", "b": "y", "c": "z"},
		"empty value": {"a": ""},
	} {
		t.Run(name, func(t *testing.T) {
			request := &pb.EchoRequest{
				RequiredIndex: &pb.Index{},
				Limited:       &pb.Limited{Field: limited},
			}
			require.Error(t, server.ValidateEchoRequest(request))
		})
	}
	require.NoError(t, server.ValidateEchoRequest(&pb.EchoRequest{RequiredIndex: &pb.Index{}, Limited: &pb.Limited{Field: map[string]string{"a": "x"}}}))
}
`

const recursiveMapMessageRoundTripHarness = `package roundtrip

import (
	"testing"

	"github.com/stretchr/testify/require"

	"%[1]s/gen/grpc/recursivemap/client"
	pb "%[1]s/gen/grpc/recursivemap/pb"
	"%[1]s/gen/grpc/recursivemap/server"
	recursivemap "%[1]s/gen/recursivemap"
)

func node(id string) *recursivemap.Node {
	return &recursivemap.Node{ID: &id}
}

func trees() map[string]recursivemap.NodeIndex {
	kids := node("kids")
	kids.Kids = recursivemap.NodeIndex{"a": node("a"), "b": node("b")}
	grid := node("grid")
	grid.Grid = []recursivemap.NodeIndex{{"k": kids}, {"c": node("c")}}
	index := node("index")
	index.Index = map[string]recursivemap.NodeIndex{"k": {"g": grid, "d": node("d")}}
	return map[string]recursivemap.NodeIndex{
		"flat":  {"x": node("x"), "y": node("y")},
		"kids":  {"kids": kids},
		"grid":  {"grid": grid},
		"index": {"index": index},
	}
}

func TestNodeRoundTrip(t *testing.T) {
	for name, tree := range trees() {
		for key, value := range tree {
			t.Run(name+"/"+key, func(t *testing.T) {
				var payload, result *recursivemap.Node
				require.NotPanics(t, func() {
					payload = server.NewEchoPayload(client.NewProtoEchoRequest(value))
					result = client.NewEchoResult(server.NewProtoEchoResponse(value))
				})
				require.Equal(t, value, payload)
				require.Equal(t, value, result)
			})
		}
	}
}

func TestNodeIndexRoundTrip(t *testing.T) {
	for name, tree := range trees() {
		t.Run(name, func(t *testing.T) {
			var payload, result, streamed recursivemap.NodeIndex
			var message *pb.NodeIndex
			require.NotPanics(t, func() {
				payload = server.NewListPayload(client.NewProtoNodeIndex(tree))
				result = client.NewListResult(server.NewProtoNodeIndex(tree))
				message = server.NewProtoNodeIndexNodeIndex(tree)
				streamed = client.NewNodeIndexNodeIndex(message)
			})
			require.Equal(t, tree, payload)
			require.Equal(t, tree, result)
			require.Equal(t, tree, streamed)
			require.Len(t, message.GetField(), len(tree))
		})
	}
}
`
