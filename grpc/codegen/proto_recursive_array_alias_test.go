package codegen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/grpc/codegen/testdata"
)

// TestProtoFilesRecursiveArrayAlias checks that generation terminates for a
// type that reaches itself through a named array field, a named array union
// branch and a named union branch, and that protoc accepts the proto file.
func TestProtoFilesRecursiveArrayAlias(t *testing.T) {
	var code string
	err := generationError(func() {
		code = protoFileCode(t, testdata.RecursiveArrayAliasDSL)
	})
	require.NoError(t, err)

	node := "{\n\toptional string id = 1;\n\tNodes children = 2;\n\toneof kids {\n\t\tsint64 int = 3;\n\t\tNodes nodes = 4;\n\t}\n\toneof next {\n\t\tbool boolean = 5;\n\t\tLink link = 6;\n\t}\n}"
	assert.Contains(t, code, "message WalkRequest "+node)
	assert.Contains(t, code, "message Node "+node)
	assert.Contains(t, code, "message Nodes {\n\trepeated Node field = 1;\n}")
	assert.Contains(t, code, "message Link {\n\toneof field {\n\t\tNode node = 1;\n\t\tNodes nodes = 2;\n\t}\n}")
	fpath := codegen.CreateTempFile(t, code)
	assert.NoError(t, protoc(defaultProtocCmd, fpath, nil), "compile proto file %q", fpath)
}

// TestProtoFilesRecursiveArrayMessageShapes checks that generation
// terminates, with every Node message referring to the Nodes message, for a
// type that reaches itself through a named array used alone as each position
// of a method or as array elements and map values, and that protoc accepts
// the proto file.
func TestProtoFilesRecursiveArrayMessageShapes(t *testing.T) {
	cases := []struct {
		name   string
		field  func()
		method func(node, nodes, leaf any)
		want   string
	}{
		{"payload", kidsField, func(_, nodes, _ any) {
			Payload(nodes)
		}, "rpc Walk (Nodes) returns (WalkResponse);"},
		{"result", kidsField, func(_, nodes, _ any) {
			Result(nodes)
		}, "rpc Walk (WalkRequest) returns (Nodes);"},
		{"streaming result", kidsField, func(_, nodes, _ any) {
			StreamingResult(nodes)
		}, "rpc Walk (WalkRequest) returns (stream Nodes);"},
		{"streaming payload and result", kidsField, func(_, nodes, leaf any) {
			StreamingPayload(leaf)
			Result(nodes)
		}, "rpc Walk (stream WalkStreamingRequest) returns (Nodes);"},
		{"array elements", func() {
			Field(2, "grid", ArrayOf("Nodes"))
		}, func(node, _, _ any) {
			Payload(node)
		}, "\trepeated Nodes grid = 2;"},
		{"map values", func() {
			Field(2, "index", MapOf(String, "Nodes"))
		}, func(node, _, _ any) {
			Payload(node)
		}, "\tmap<string, Nodes> index = 2;"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var code string
			err := generationError(func() {
				code = protoFileCode(t, func() {
					leaf := Type("Leaf", func() {
						Field(1, "name", String)
					})
					node := Type("Node", func() {
						Field(1, "id", String)
						c.field()
					})
					nodes := Type("Nodes", ArrayOf(node))
					Service("recursive", func() {
						Method("walk", func() {
							c.method(node, nodes, leaf)
							GRPC(func() {})
						})
					})
				})
			})
			require.NoError(t, err)
			assert.Contains(t, code, c.want)
			assert.Contains(t, code, "message Nodes {\n\trepeated Node field = 1;\n}")
			assert.NotContains(t, code, "message NodesNodes")
			fpath := codegen.CreateTempFile(t, code)
			assert.NoError(t, protoc(defaultProtocCmd, fpath, nil), "compile proto file %q", fpath)
		})
	}
}

// kidsField declares the field of Node that holds the named array Nodes.
func kidsField() {
	Field(2, "kids", "Nodes")
}

// TestGeneratedRecursiveArrayMessageRoundTrip compiles the module generated
// for a type that reaches itself through a named array used as a field, as
// array elements and as map values, and as a unary and streaming payload and
// result, and round-trips nested values through the generated protobuf
// conversions.
func TestGeneratedRecursiveArrayMessageRoundTrip(t *testing.T) {
	runGeneratedRoundTrip(t, "example.com/grpcrecursivemessage", testdata.RecursiveArrayMessageDSL, recursiveArrayMessageRoundTripHarness)
}

// TestGeneratedRecursiveArrayAliasRoundTrip compiles the module generated
// for a type that reaches itself through named arrays and a named union
// branch, and round-trips nested values through the generated protobuf
// conversions.
func TestGeneratedRecursiveArrayAliasRoundTrip(t *testing.T) {
	runGeneratedRoundTrip(t, "example.com/grpcrecursivearray", testdata.RecursiveArrayAliasDSL, recursiveArrayAliasRoundTripHarness)
}

const recursiveArrayAliasRoundTripHarness = `package roundtrip

import (
	"testing"

	"github.com/stretchr/testify/require"

	"%[1]s/gen/grpc/recursivearray/client"
	"%[1]s/gen/grpc/recursivearray/server"
	recursivearray "%[1]s/gen/recursivearray"
)

func node(id string) *recursivearray.Node {
	return &recursivearray.Node{ID: &id}
}

func nodes() map[string]*recursivearray.Node {
	cases := map[string]*recursivearray.Node{"leaf": node("leaf")}

	children := node("children")
	grandchild := node("grandchild")
	grandchild.Children = recursivearray.Nodes{node("deep")}
	children.Children = recursivearray.Nodes{node("a"), grandchild}
	cases["children"] = children

	kids := node("kids")
	kids.Kids = &recursivearray.IntOrNodes{}
	inner := node("inner")
	inner.Kids = &recursivearray.IntOrNodes{}
	inner.Kids.SetInt(3)
	kids.Kids.SetNodes(recursivearray.Nodes{inner})
	cases["kids"] = kids

	linked := node("linked")
	linked.Next = &recursivearray.BooleanOrLink{}
	link := &recursivearray.Link{}
	target := node("target")
	target.Next = &recursivearray.BooleanOrLink{}
	target.Next.SetBoolean(true)
	link.SetNode(target)
	linked.Next.SetLink(link)
	cases["next/node"] = linked

	listed := node("listed")
	listed.Next = &recursivearray.BooleanOrLink{}
	list := &recursivearray.Link{}
	list.SetNodes(recursivearray.Nodes{children})
	listed.Next.SetLink(list)
	cases["next/nodes"] = listed
	return cases
}

func TestRoundTrip(t *testing.T) {
	for name, value := range nodes() {
		t.Run(name, func(t *testing.T) {
			var payload, result *recursivearray.Node
			require.NotPanics(t, func() {
				payload = server.NewWalkPayload(client.NewProtoWalkRequest(value))
				result = client.NewWalkResult(server.NewProtoWalkResponse(value))
			})
			require.Equal(t, value, payload)
			require.Equal(t, value, result)
			require.NoError(t, server.ValidateWalkRequest(client.NewProtoWalkRequest(value)))
		})
	}
}
`

const recursiveArrayMessageRoundTripHarness = `package roundtrip

import (
	"testing"

	"github.com/stretchr/testify/require"

	"%[1]s/gen/grpc/recursivemessage/client"
	pb "%[1]s/gen/grpc/recursivemessage/pb"
	"%[1]s/gen/grpc/recursivemessage/server"
	recursivemessage "%[1]s/gen/recursivemessage"
)

func node(id string) *recursivemessage.Node {
	return &recursivemessage.Node{ID: &id}
}

func trees() map[string]recursivemessage.Nodes {
	kids := node("kids")
	kids.Kids = recursivemessage.Nodes{node("a"), node("b")}
	grid := node("grid")
	grid.Grid = []recursivemessage.Nodes{{kids}, {node("c")}}
	index := node("index")
	index.Index = map[string]recursivemessage.Nodes{"k": {grid, node("d")}}
	return map[string]recursivemessage.Nodes{
		"flat":  {node("x"), node("y")},
		"kids":  {kids},
		"grid":  {grid},
		"index": {index},
	}
}

func TestNodeRoundTrip(t *testing.T) {
	for name, tree := range trees() {
		for i, value := range tree {
			t.Run(name, func(t *testing.T) {
				var payload, result *recursivemessage.Node
				require.NotPanics(t, func() {
					payload = server.NewEchoPayload(client.NewProtoEchoRequest(value))
					result = client.NewEchoResult(server.NewProtoEchoResponse(value))
				})
				require.Equal(t, tree[i], payload)
				require.Equal(t, tree[i], result)
			})
		}
	}
}

func TestNodesRoundTrip(t *testing.T) {
	for name, tree := range trees() {
		t.Run(name, func(t *testing.T) {
			var payload, result, streamed recursivemessage.Nodes
			var message *pb.Nodes
			require.NotPanics(t, func() {
				payload = server.NewListPayload(client.NewProtoNodes(tree))
				result = client.NewListResult(server.NewProtoNodes(tree))
				message = server.NewProtoNodesNodes(tree)
				streamed = client.NewNodesNodes(message)
			})
			require.Equal(t, tree, payload)
			require.Equal(t, tree, result)
			require.Equal(t, tree, streamed)
			require.Len(t, message.GetField(), len(tree))
		})
	}
}
`
