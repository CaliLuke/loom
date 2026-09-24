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

// TestProtoFilesArrayAlias checks that a named array is the message that
// wraps the array in its "field" attribute wherever it appears: as a message
// field, an array element, a map value, a union branch and directly as a
// payload and result, and that a named array of a named array wraps the array
// itself as a message field and as a direct payload and result.
func TestProtoFilesArrayAlias(t *testing.T) {
	code := protoFileCode(t, testdata.ArrayAliasDSL)

	assert.Contains(t, code, "rpc List (Tags) returns (Tags);")
	assert.Contains(t, code, "rpc Extend (More) returns (More);")
	assert.Contains(t, code, "message EchoRequest {\n\tTags labels = 1;\n\tTags required_labels = 2;\n\tLeaves leaf_list = 3;\n\tMore more = 4;\n\trepeated Tags label_lists = 5;\n\tmap<string, Tags> labels_by_key = 6;\n\toneof pick {\n\t\tTags tags = 7;\n\t\tLeaves leaves = 8;\n\t}\n\toneof detail {\n\t\tTags names = 9;\n\t\tDetailWords words = 10;\n\t\tsint64 count = 11;\n\t}\n}")
	for _, message := range []string{
		"message Tags {\n\trepeated string field = 1;\n}",
		"message Leaves {\n\trepeated Leaf field = 1;\n}",
		"message More {\n\trepeated string field = 1;\n}",
		"message DetailWords {\n\trepeated string field = 1;\n}",
	} {
		assert.Contains(t, code, message)
	}
	assert.NotContains(t, code, "message TagsTags")
	testutil.AssertString(t, "testdata/golden/proto_protofiles-array-alias.proto.golden", code)
	fpath := codegen.CreateTempFile(t, code)
	assert.NoError(t, protoc(defaultProtocCmd, fpath, nil), "compile proto file %q", fpath)
}

// TestProtoFilesUnionBranchUnion checks that a union used as a branch of
// another union, named or not, is the message that wraps its oneof, the
// message of the union used directly as a payload or result, and that the
// same named union used as a field remains a oneof of the message that holds
// it.
func TestProtoFilesUnionBranchUnion(t *testing.T) {
	code := protoFileCode(t, testdata.UnionBranchUnionDSL)

	assert.Contains(t, code, "rpc Wrap (Outer) returns (Outer);")
	assert.Contains(t, code, "rpc Pick (Choice) returns (PickResponse);")
	assert.Contains(t, code, "message EchoRequest {\n\toptional string id = 1;\n\toneof wrapped {\n\t\tChoice choice = 2;\n\t\tExtra extra = 3;\n\t}\n\toneof nested {\n\t\tLeaf leaf = 4;\n\t\tExtraOrOther extra_or_other = 5;\n\t}\n\toneof block {\n\t\tChoice picked = 6;\n\t\tstring text = 7;\n\t}\n\tHolder holder = 8;\n\trepeated Holder holders = 9;\n}")
	assert.Contains(t, code, "message Holder {\n\toneof pick {\n\t\tLeaf leaf = 1;\n\t\tOther other = 2;\n\t}\n\toneof outer {\n\t\tChoice choice = 3;\n\t\tExtra extra = 4;\n\t}\n}")
	for _, message := range []string{
		"message Choice {\n\toneof field {\n\t\tLeaf leaf = 1;\n\t\tOther other = 2;\n\t}\n}",
		"message ExtraOrOther {\n\toneof field {\n\t\tExtra extra = 1;\n\t\tOther other = 2;\n\t}\n}",
		"message Outer {\n\toneof field {\n\t\tChoice choice = 1;\n\t\tExtra extra = 2;\n\t}\n}",
	} {
		assert.Contains(t, code, message)
	}
	assert.NotContains(t, code, "= 0;")
	testutil.AssertString(t, "testdata/golden/proto_protofiles-union-branch-union.proto.golden", code)
	fpath := codegen.CreateTempFile(t, code)
	assert.NoError(t, protoc(defaultProtocCmd, fpath, nil), "compile proto file %q", fpath)
}

// TestProtoFilesArrayAliasAndUnionBranchShapes checks that each shape of the
// issue alone, with no direct payload or result declaring the message of the
// named array or union, generates the message that the field or branch refers
// to and a proto file that protoc accepts.
func TestProtoFilesArrayAliasAndUnionBranchShapes(t *testing.T) {
	cases := []struct {
		name     string
		payload  func(leaf, tags, choice, extra, outer any)
		contains []string
	}{
		{"array alias field", func(_, tags, _, _, _ any) {
			Field(1, "labels", tags)
		}, []string{"\tTags labels = 1;", "message Tags {\n\trepeated string field = 1;\n}"}},
		{"array alias branch", func(leaf, tags, _, _, _ any) {
			Field(1, "pick", OneOf(tags, leaf))
		}, []string{"\toneof pick {\n\t\tTags tags = 1;\n\t\tLeaf leaf = 2;\n\t}", "message Tags {\n\trepeated string field = 1;\n}"}},
		{"named union branch of named union", func(_, _, _, _, outer any) {
			Field(1, "outer", outer)
		}, []string{"\toneof outer {\n\t\tChoice choice = 1;\n\t\tExtra extra = 2;\n\t}", "message Choice {\n\toneof field {\n\t\tLeaf leaf = 1;\n\t\tOther other = 2;\n\t}\n}"}},
		{"named union branch of constructor union", func(_, _, choice, extra, _ any) {
			Field(1, "outer", OneOf(choice, extra))
		}, []string{"\toneof outer {\n\t\tChoice choice = 1;\n\t\tExtra extra = 2;\n\t}", "message Choice {\n\toneof field {\n\t\tLeaf leaf = 1;\n\t\tOther other = 2;\n\t}\n}"}},
		{"named union branch of block union", func(_, _, choice, extra, _ any) {
			OneOf("outer", func() {
				Field(1, "a", choice)
				Field(2, "b", extra)
			})
		}, []string{"\toneof outer {\n\t\tChoice a = 1;\n\t\tExtra b = 2;\n\t}", "message Choice {\n\toneof field {\n\t\tLeaf leaf = 1;\n\t\tOther other = 2;\n\t}\n}"}},
		{"constructor union branch of constructor union", func(leaf, tags, _, extra, _ any) {
			Field(1, "outer", OneOf(extra, OneOf(leaf, tags)))
		}, []string{"\toneof outer {\n\t\tExtra extra = 1;\n\t\tLeafOrTags leaf_or_tags = 2;\n\t}", "message LeafOrTags {\n\toneof field {\n\t\tLeaf leaf = 1;\n\t\tTags tags = 2;\n\t}\n}", "message Tags {\n\trepeated string field = 1;\n}"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code := protoFileCode(t, func() {
				leaf := Type("Leaf", func() {
					Field(1, "name", String)
				})
				other := Type("Other", func() {
					Field(1, "count", Int)
				})
				extra := Type("Extra", func() {
					Field(1, "flag", Boolean)
				})
				tags := Type("Tags", ArrayOf(String))
				choice := Type("Choice", OneOf(leaf, other))
				outer := Type("Outer", OneOf(choice, extra))
				Service("shapes", func() {
					Method("echo", func() {
						Payload(func() {
							c.payload(leaf, tags, choice, extra, outer)
						})
						GRPC(func() {})
					})
				})
			})
			for _, want := range c.contains {
				assert.Contains(t, code, want)
			}
			assert.NotContains(t, code, "= 0;")
			fpath := codegen.CreateTempFile(t, code)
			assert.NoError(t, protoc(defaultProtocCmd, fpath, nil), "compile proto file %q", fpath)
		})
	}
}

// TestGeneratedArrayAliasRoundTrip compiles a generated module whose messages
// carry named arrays as optional and required fields, array elements, map
// values and union branches, and as a direct payload and result. It
// round-trips the service values through the generated protobuf conversions
// with the arrays unset, empty and set, and with every union branch.
func TestGeneratedArrayAliasRoundTrip(t *testing.T) {
	runGeneratedRoundTrip(t, "example.com/grpcarrayalias", testdata.ArrayAliasDSL, arrayAliasRoundTripHarness)
}

// TestGeneratedUnionBranchUnionRoundTrip compiles a generated module whose
// unions hold named and constructor unions as branches, and round-trips the
// service values through the generated protobuf conversions with every union
// unset and set to each branch, directly and nested.
func TestGeneratedUnionBranchUnionRoundTrip(t *testing.T) {
	runGeneratedRoundTrip(t, "example.com/grpcunionbranchunion", testdata.UnionBranchUnionDSL, unionBranchUnionRoundTripHarness)
}

// runGeneratedRoundTrip renders the gRPC module generated for dsl, adds the
// round-trip test harness to it, then builds, vets and tests it.
func runGeneratedRoundTrip(t *testing.T, modulePath string, dsl func(), harness string) {
	t.Helper()
	root := RunGRPCDSL(t, dsl)
	dir := t.TempDir()
	renderGRPCModule(t, dir, modulePath, root, resolveGRPCLoomSource(t))
	testDir := filepath.Join(dir, "internal", "roundtrip")
	require.NoError(t, os.MkdirAll(testDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(testDir, "roundtrip_test.go"), []byte(fmt.Sprintf(harness, modulePath)), 0o600))

	runGRPCGoCommand(t, dir, "mod", "tidy")
	runGRPCGoCommand(t, dir, "build", "./...")
	runGRPCGoCommand(t, dir, "vet", "./...")
	runGRPCGoCommand(t, dir, "test", "./internal/roundtrip")
}

const arrayAliasRoundTripHarness = `package roundtrip

import (
	"testing"

	"github.com/stretchr/testify/require"

	arrayalias "%[1]s/gen/arrayalias"
	"%[1]s/gen/grpc/arrayalias/client"
	pb "%[1]s/gen/grpc/arrayalias/pb"
	"%[1]s/gen/grpc/arrayalias/server"
)

func envelopes() map[string]*arrayalias.Envelope {
	name := "leaf"
	leaves := arrayalias.Leaves{{Name: &name}, {}}
	cases := map[string]*arrayalias.Envelope{}
	full := &arrayalias.Envelope{
		Labels:         arrayalias.Tags{"a", "b"},
		RequiredLabels: arrayalias.Tags{"required"},
		LeafList:       leaves,
		More:           arrayalias.More{"more"},
		LabelLists:     []arrayalias.Tags{{"x"}, {}, {"y", "z"}},
		LabelsByKey:    map[string]arrayalias.Tags{"k": {"v"}, "empty": {}},
	}
	full.Pick.SetTags(arrayalias.Tags{"picked"})
	cases["full"] = full
	empty := &arrayalias.Envelope{Labels: arrayalias.Tags{}, RequiredLabels: arrayalias.Tags{}, LeafList: arrayalias.Leaves{}}
	empty.Pick.SetTags(arrayalias.Tags{})
	cases["empty"] = empty
	unset := &arrayalias.Envelope{}
	unset.Pick.SetLeaves(leaves)
	cases["unset"] = unset
	for name, set := range map[string]func(*arrayalias.Detail){
		"names": func(d *arrayalias.Detail) { d.SetNames(arrayalias.Tags{"n"}) },
		"words": func(d *arrayalias.Detail) { d.SetWords(arrayalias.DetailWords{"w1", "w2"}) },
		"count": func(d *arrayalias.Detail) { d.SetCount(3) },
	} {
		envelope := &arrayalias.Envelope{Detail: &arrayalias.Detail{}}
		envelope.Pick.SetLeaves(arrayalias.Leaves{{Name: &name}})
		set(envelope.Detail)
		cases["detail/"+name] = envelope
	}
	return cases
}

func TestPayloadRoundTrip(t *testing.T) {
	for name, envelope := range envelopes() {
		t.Run(name, func(t *testing.T) {
			var decoded *arrayalias.Envelope
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
			var decoded *arrayalias.Envelope
			require.NotPanics(t, func() {
				decoded = client.NewEchoResult(server.NewProtoEchoResponse(envelope))
			})
			require.Equal(t, envelope, decoded)
		})
	}
}

func TestDirectRoundTrip(t *testing.T) {
	for name, tags := range map[string]arrayalias.Tags{"empty": {}, "set": {"a", "b"}} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tags, server.NewListPayload(client.NewProtoTags(tags)))
			require.Equal(t, tags, client.NewListResult(server.NewProtoTags(tags)))
			more := arrayalias.More(tags)
			require.Equal(t, more, server.NewExtendPayload(client.NewProtoMore(more)))
			require.Equal(t, more, client.NewExtendResult(server.NewProtoMore(more)))
			require.Equal(t, []string(tags), client.NewProtoMore(more).GetField())
		})
	}
}

func TestProtoMessages(t *testing.T) {
	request := client.NewProtoEchoRequest(envelopes()["full"])
	require.Equal(t, []string{"a", "b"}, request.GetLabels().GetField())
	require.Equal(t, []string{"more"}, request.GetMore().GetField())
	require.Equal(t, []string{"y", "z"}, request.GetLabelLists()[2].GetField())
	require.Equal(t, []string{"v"}, request.GetLabelsByKey()["k"].GetField())
	pick, ok := request.GetPick().(*pb.EchoRequest_Tags)
	require.True(t, ok)
	require.Equal(t, []string{"picked"}, pick.Tags.GetField())

	payload := server.NewEchoPayload(&pb.EchoRequest{
		Pick:   &pb.EchoRequest_Leaves{Leaves: &pb.Leaves{Field: []*pb.Leaf{{}}}},
		Detail: &pb.EchoRequest_Words{Words: &pb.DetailWords{Field: []string{"w"}}},
	})
	leaves, ok := payload.Pick.AsLeaves()
	require.True(t, ok)
	require.Len(t, leaves, 1)
	words, ok := payload.Detail.AsWords()
	require.True(t, ok)
	require.Equal(t, arrayalias.DetailWords{"w"}, words)
}
`

const unionBranchUnionRoundTripHarness = `package roundtrip

import (
	"testing"

	"github.com/stretchr/testify/require"

	nestedunion "%[1]s/gen/nestedunion"
	"%[1]s/gen/grpc/nestedunion/client"
	pb "%[1]s/gen/grpc/nestedunion/pb"
	"%[1]s/gen/grpc/nestedunion/server"
)

func choices() map[string]*nestedunion.Choice {
	name := "leaf"
	count := 42
	leaf := &nestedunion.Choice{}
	leaf.SetLeaf(&nestedunion.Leaf{Name: &name})
	other := &nestedunion.Choice{}
	other.SetOther(&nestedunion.Other{Count: &count})
	return map[string]*nestedunion.Choice{"leaf": leaf, "other": other}
}

func outers() map[string]*nestedunion.Outer {
	flag := true
	cases := map[string]*nestedunion.Outer{}
	for name, choice := range choices() {
		outer := &nestedunion.Outer{}
		outer.SetChoice(choice)
		cases["choice/"+name] = outer
	}
	extra := &nestedunion.Outer{}
	extra.SetExtra(&nestedunion.Extra{Flag: &flag})
	cases["extra"] = extra
	return cases
}

func holders() map[string]*nestedunion.Holder {
	cases := map[string]*nestedunion.Holder{}
	for name, outer := range outers() {
		cases["outer/"+name] = &nestedunion.Holder{Outer: *outer}
	}
	for name, choice := range choices() {
		holder := &nestedunion.Holder{Pick: choice}
		holder.Outer.SetChoice(choice)
		cases["pick/"+name] = holder
	}
	return cases
}

func envelopes() map[string]*nestedunion.Envelope {
	id := "id"
	text := "text"
	count := 7
	cases := map[string]*nestedunion.Envelope{"unset": {ID: &id}}
	for name, choice := range choices() {
		wrapped := &nestedunion.ChoiceOrExtra{}
		wrapped.SetChoice(choice)
		block := &nestedunion.Block{}
		block.SetPicked(choice)
		cases["choice/"+name] = &nestedunion.Envelope{Wrapped: wrapped, Block: block}
	}
	extra := &nestedunion.ChoiceOrExtra{}
	extra.SetExtra(&nestedunion.Extra{})
	textBlock := &nestedunion.Block{}
	textBlock.SetText(nestedunion.BlockText(text))
	cases["extra"] = &nestedunion.Envelope{Wrapped: extra, Block: textBlock}
	inner := &nestedunion.ExtraOrOther{}
	inner.SetOther(&nestedunion.Other{Count: &count})
	nested := &nestedunion.ExtraOrOtherOrLeaf{}
	nested.SetExtraOrOther(inner)
	cases["nested/union"] = &nestedunion.Envelope{Nested: nested}
	leaf := &nestedunion.ExtraOrOtherOrLeaf{}
	leaf.SetLeaf(&nestedunion.Leaf{})
	cases["nested/leaf"] = &nestedunion.Envelope{Nested: leaf}
	for name, holder := range holders() {
		cases["holder/"+name] = &nestedunion.Envelope{Holder: holder, Holders: []*nestedunion.Holder{holder, holder}}
	}
	return cases
}

func TestEnvelopeRoundTrip(t *testing.T) {
	for name, envelope := range envelopes() {
		t.Run(name, func(t *testing.T) {
			var payload, result *nestedunion.Envelope
			require.NotPanics(t, func() {
				payload = server.NewEchoPayload(client.NewProtoEchoRequest(envelope))
				result = client.NewEchoResult(server.NewProtoEchoResponse(envelope))
			})
			require.Equal(t, envelope, payload)
			require.Equal(t, envelope, result)
			require.NoError(t, server.ValidateEchoRequest(client.NewProtoEchoRequest(envelope)))
		})
	}
}

func TestOuterRoundTrip(t *testing.T) {
	for name, outer := range outers() {
		t.Run(name, func(t *testing.T) {
			var payload, result *nestedunion.Outer
			require.NotPanics(t, func() {
				payload = server.NewWrapPayload(client.NewProtoOuter(outer))
				result = client.NewWrapResult(server.NewProtoOuter(outer))
			})
			require.Equal(t, outer, payload)
			require.Equal(t, outer, result)
			require.NoError(t, server.ValidateOuter(client.NewProtoOuter(outer)))
		})
	}
}

func TestChoiceRoundTrip(t *testing.T) {
	for name, choice := range choices() {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, choice, server.NewPickPayload(client.NewProtoChoice(choice)))
		})
	}
	for name, holder := range holders() {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, holder, client.NewPickResult(server.NewProtoPickResponse(holder)))
		})
	}
}

func TestProtoMessages(t *testing.T) {
	name := "leaf"
	request := &pb.EchoRequest{
		Wrapped: &pb.EchoRequest_Choice{Choice: &pb.Choice{Field: &pb.Choice_Leaf{Leaf: &pb.Leaf{Name: &name}}}},
		Nested:  &pb.EchoRequest_ExtraOrOther{ExtraOrOther: &pb.ExtraOrOther{Field: &pb.ExtraOrOther_Extra{Extra: &pb.Extra{}}}},
		Block:   &pb.EchoRequest_Picked{Picked: &pb.Choice{Field: &pb.Choice_Other{Other: &pb.Other{}}}},
	}
	require.NoError(t, server.ValidateEchoRequest(request))
	payload := server.NewEchoPayload(request)
	choice, ok := payload.Wrapped.AsChoice()
	require.True(t, ok)
	leaf, ok := choice.AsLeaf()
	require.True(t, ok)
	require.Equal(t, "leaf", *leaf.Name)
	inner, ok := payload.Nested.AsExtraOrOther()
	require.True(t, ok)
	_, ok = inner.AsExtra()
	require.True(t, ok)
	picked, ok := payload.Block.AsPicked()
	require.True(t, ok)
	_, ok = picked.AsOther()
	require.True(t, ok)

	empty := &pb.EchoRequest{Wrapped: &pb.EchoRequest_Choice{Choice: &pb.Choice{}}}
	require.Error(t, server.ValidateEchoRequest(empty), "a union branch that holds a union requires one of its branches")
}
`
