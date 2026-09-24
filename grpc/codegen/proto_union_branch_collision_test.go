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

// TestProtoFilesUnionBranchCollision checks that the oneofs and oneof
// branches of a message take names that differ from the names of the other
// fields, oneofs and oneof branches of the message. A regular field keeps its
// name, the first oneof or branch to use a name keeps it, and a later branch
// takes the name of its union field as a prefix.
func TestProtoFilesUnionBranchCollision(t *testing.T) {
	code := protoFileCode(t, testdata.UnionBranchCollisionDSL)

	for _, message := range []string{
		"message EchoRequest {\n\toneof a {\n\t\tstring string_ = 1;\n\t\tsint64 int64_ = 2;\n\t}\n\toneof b {\n\t\tbool boolean = 3;\n\t\tsint64 b_int64 = 4;\n\t}\n\toneof c {\n\t\tsint64 c_int64 = 5;\n\t\tLeaf c_leaf = 6;\n\t}\n\toptional string leaf = 7;\n\toneof d {\n\t\tsint64 d_int64 = 8;\n\t\tstring d_leaf = 9;\n\t}\n\toneof pick {\n\t\tLeaf pick_leaf = 10;\n\t\tOther other = 11;\n\t}\n\tHolder holder = 12;\n\trepeated Holder holders = 13;\n}",
		"message Holder {\n\toneof first {\n\t\tLeaf leaf = 1;\n\t\tOther other = 2;\n\t}\n\toneof second {\n\t\tLeaf second_leaf = 3;\n\t\tOther second_other = 4;\n\t}\n\toneof tag {\n\t\tstring code = 5;\n\t\tOther tag_other = 6;\n\t}\n\toneof alt {\n\t\tstring alt_code = 7;\n\t\tOther alt_other = 8;\n\t}\n}",
		"message GrowRequest {\n\toneof leaf_oneof {\n\t\tLeaf leaf = 1;\n\t\tOther other = 2;\n\t}\n\toneof choice {\n\t\tLeaf choice_leaf = 3;\n\t\tOther choice_other = 4;\n\t}\n}",
		"message MatchRequest {\n\toneof leaf_oneof_oneof {\n\t\tLeaf leaf = 1;\n\t\tOther other = 2;\n\t}\n\toptional string leaf_oneof = 3;\n}",
		"message MatchResponse {\n\toneof u {\n\t\tLeaf leaf = 1;\n\t\tOther other = 2;\n\t}\n\toneof w {\n\t\tLeaf w_u = 3;\n\t\tOther v = 4;\n\t}\n}",
	} {
		assert.Contains(t, code, message)
	}
	testutil.AssertString(t, "testdata/golden/proto_protofiles-union-branch-collision.proto.golden", code)
	fpath := codegen.CreateTempFile(t, code)
	assert.NoError(t, protoc(defaultProtocCmd, fpath, nil), "compile proto file %q", fpath)
}

// TestProtoFilesUnionBranchCollisionShapes checks each collision alone: the
// oneof and branch names that the message takes and that protoc accepts the
// proto file.
func TestProtoFilesUnionBranchCollisionShapes(t *testing.T) {
	cases := []struct {
		name     string
		payload  func(leaf, other, choice any)
		contains []string
	}{
		{"primitive branches of constructor unions", func(_, _, _ any) {
			Field(1, "a", OneOf(String, Int64))
			Field(3, "b", OneOf(Boolean, Int64))
		}, []string{"\toneof a {\n\t\tstring string_ = 1;\n\t\tsint64 int64_ = 2;\n\t}\n\toneof b {\n\t\tbool boolean = 3;\n\t\tsint64 b_int64 = 4;\n\t}"}},
		{"three unions share a branch", func(leaf, _, _ any) {
			Field(1, "a", OneOf(String, Int64))
			Field(3, "b", OneOf(Boolean, Int64))
			Field(5, "c", OneOf(Int64, leaf))
		}, []string{"\toneof b {\n\t\tbool boolean = 3;\n\t\tsint64 b_int64 = 4;\n\t}\n\toneof c {\n\t\tsint64 c_int64 = 5;\n\t\tLeaf leaf = 6;\n\t}"}},
		{"object branches of constructor unions", func(leaf, other, _ any) {
			Field(1, "a", OneOf(leaf, other))
			Field(3, "b", OneOf(other, leaf))
		}, []string{"\toneof a {\n\t\tLeaf leaf = 1;\n\t\tOther other = 2;\n\t}\n\toneof b {\n\t\tOther b_other = 3;\n\t\tLeaf b_leaf = 4;\n\t}"}},
		{"named union twice", func(_, _, choice any) {
			Field(1, "first", choice)
			Field(3, "second", choice)
		}, []string{"\toneof first {\n\t\tLeaf leaf = 1;\n\t\tOther other = 2;\n\t}\n\toneof second {\n\t\tLeaf second_leaf = 3;\n\t\tOther second_other = 4;\n\t}"}},
		{"block branches", func(leaf, _, _ any) {
			OneOf("a", func() {
				Field(1, "id", Int64)
				Field(2, "leaf", leaf)
			})
			OneOf("b", func() {
				Field(3, "id", String)
				Field(4, "leaf", leaf)
			})
		}, []string{"\toneof a {\n\t\tsint64 id = 1;\n\t\tLeaf leaf = 2;\n\t}\n\toneof b {\n\t\tstring b_id = 3;\n\t\tLeaf b_leaf = 4;\n\t}"}},
		{"regular field declared after the branch", func(leaf, other, _ any) {
			Field(1, "pick", OneOf(leaf, other))
			Field(3, "leaf", String)
		}, []string{"\toneof pick {\n\t\tLeaf pick_leaf = 1;\n\t\tOther other = 2;\n\t}\n\toptional string leaf = 3;"}},
		{"prefixed branch name taken by a regular field", func(_, _, _ any) {
			Field(1, "a", OneOf(String, Int64))
			Field(3, "b", OneOf(Boolean, Int64))
			Field(5, "b_int64", String)
		}, []string{"\toneof b {\n\t\tbool boolean = 3;\n\t\tsint64 b_b_int64 = 4;\n\t}\n\toptional string b_int64 = 5;"}},
		{"union field named like its branch", func(leaf, other, _ any) {
			Field(1, "leaf", OneOf(leaf, other))
		}, []string{"\toneof leaf_oneof {\n\t\tLeaf leaf = 1;\n\t\tOther other = 2;\n\t}"}},
		{"named union field named like its branch", func(_, _, choice any) {
			Field(1, "leaf", choice)
		}, []string{"\toneof leaf_oneof {\n\t\tLeaf leaf = 1;\n\t\tOther other = 2;\n\t}"}},
		{"renamed oneof taken by a regular field", func(_, _, choice any) {
			Field(1, "leaf", choice)
			Field(3, "leaf_oneof", String)
		}, []string{"\toneof leaf_oneof_oneof {\n\t\tLeaf leaf = 1;\n\t\tOther other = 2;\n\t}\n\toptional string leaf_oneof = 3;"}},
		{"oneof named like a branch of another oneof", func(leaf, other, _ any) {
			Field(1, "u", OneOf(leaf, other))
			OneOf("w", func() {
				Field(3, "u", leaf)
				Field(4, "v", other)
			})
		}, []string{"\toneof u {\n\t\tLeaf leaf = 1;\n\t\tOther other = 2;\n\t}\n\toneof w {\n\t\tLeaf w_u = 3;\n\t\tOther v = 4;\n\t}"}},
		{"branch named like a later oneof", func(leaf, other, _ any) {
			OneOf("w", func() {
				Field(1, "u", leaf)
				Field(2, "v", other)
			})
			Field(3, "u", OneOf(leaf, other))
		}, []string{"\toneof w {\n\t\tLeaf u = 1;\n\t\tOther v = 2;\n\t}\n\toneof u_oneof {\n\t\tLeaf leaf = 3;\n\t\tOther other = 4;\n\t}"}},
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
				choice := Type("Choice", OneOf(leaf, other))
				Service("collision", func() {
					Method("echo", func() {
						Payload(func() {
							c.payload(leaf, other, choice)
						})
						GRPC(func() {})
					})
				})
			})
			for _, want := range c.contains {
				assert.Contains(t, code, want)
			}
			fpath := codegen.CreateTempFile(t, code)
			assert.NoError(t, protoc(defaultProtocCmd, fpath, nil), "compile proto file %q", fpath)
		})
	}
}

// TestTypeFilesUnionFieldNamedLikeBranch checks that the Go conversion and
// validation code of a union field named like one of its branches, whose
// oneof protoc names "leaf_oneof", refers to the Go field of the renamed
// oneof in constructor and named form.
func TestTypeFilesUnionFieldNamedLikeBranch(t *testing.T) {
	cases := []struct {
		name  string
		union func(leaf, other, choice any) any
	}{
		{"constructor", func(leaf, other, _ any) any { return OneOf(leaf, other) }},
		{"named", func(_, _, choice any) any { return choice }},
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
				choice := Type("Choice", OneOf(leaf, other))
				Service("collision", func() {
					Method("echo", func() {
						Payload(func() {
							Field(1, "leaf", c.union(leaf, other, choice))
							Required("leaf")
						})
						Result(func() {
							Field(1, "leaf", c.union(leaf, other, choice))
						})
						GRPC(func() {})
					})
				})
			})
			services := CreateGRPCServices(root)
			var code string
			for _, f := range append(ServerTypeFiles("", services), ClientTypeFiles("", services)...) {
				code += sectionCode(t, f.AllSections()[1:]...)
			}
			for _, want := range []string{
				"if message.LeafOneof == nil {",
				"switch val := message.LeafOneof.(type) {",
				"case *collisionpb.EchoRequest_Leaf:",
				"message.LeafOneof = &collisionpb.EchoResponse_Leaf{Leaf: ",
				"message.LeafOneof = &collisionpb.EchoRequest_Other{Other: ",
			} {
				assert.Contains(t, code, want)
			}
			assert.NotContains(t, code, "message.Leaf ")
			assert.NotContains(t, code, "message.Leaf.")
		})
	}
}

// TestTypeFilesMappedUnionFieldName checks that the Go conversion code of a
// union field whose attribute name has a mapping suffix, such as "pick:p",
// refers to the oneof and to the renamed oneof fields of the message.
func TestTypeFilesMappedUnionFieldName(t *testing.T) {
	root := RunGRPCDSL(t, func() {
		leaf := Type("Leaf", func() {
			Field(1, "name", String)
		})
		other := Type("Other", func() {
			Field(1, "count", Int)
		})
		Service("collision", func() {
			Method("echo", func() {
				Payload(func() {
					Field(1, "pick:p", OneOf(leaf, other))
					Field(3, "leaf", String)
				})
				GRPC(func() {})
			})
		})
	})
	services := CreateGRPCServices(root)
	fs := ProtoFiles("", services)
	require.Len(t, fs, 1)
	code := sectionCode(t, fs[0].AllSections()[1:]...)
	assert.Contains(t, code, "\toneof pick {\n\t\tLeaf pick_leaf = 1;\n\t\tOther other = 2;\n\t}\n\toptional string leaf = 3;")
	var goCode string
	for _, f := range ClientTypeFiles("", services) {
		goCode += sectionCode(t, f.AllSections()[1:]...)
	}
	assert.Contains(t, goCode, "message.Pick = &collisionpb.EchoRequest_PickLeaf{PickLeaf: ")
	assert.Contains(t, goCode, "message.Pick = &collisionpb.EchoRequest_Other{Other: ")
	assert.NotContains(t, goCode, "message. =")
}

// TestGeneratedUnionBranchCollisionRoundTrip compiles a generated module whose
// messages have oneofs and oneof fields renamed to make their names unique. It
// round-trips the service values through the generated protobuf conversions
// with every union unset and set to each branch, directly and nested, builds
// the protocol buffer messages with the renamed Go fields and wrapper types,
// and runs the generated validations on them.
func TestGeneratedUnionBranchCollisionRoundTrip(t *testing.T) {
	runGeneratedRoundTrip(t, "example.com/grpcunioncollision", testdata.UnionBranchCollisionDSL, unionBranchCollisionRoundTripHarness)
}

const unionBranchCollisionRoundTripHarness = `package roundtrip

import (
	"testing"

	"github.com/stretchr/testify/require"

	collision "%[1]s/gen/collision"
	"%[1]s/gen/grpc/collision/client"
	pb "%[1]s/gen/grpc/collision/pb"
	"%[1]s/gen/grpc/collision/server"
)

func choices() map[string]*collision.Choice {
	name := "leaf"
	count := 42
	leaf := &collision.Choice{}
	leaf.SetLeaf(&collision.Leaf{Name: &name})
	other := &collision.Choice{}
	other.SetOther(&collision.Other{Count: &count})
	return map[string]*collision.Choice{"leaf": leaf, "other": other}
}

func envelopes() map[string]*collision.Envelope {
	name := "leaf"
	label := "label"
	cases := map[string]*collision.Envelope{}
	text := &collision.Envelope{Leaf: &label}
	text.A.SetString("text")
	cases["a/string"] = text
	number := &collision.Envelope{}
	number.A.SetInt64(1)
	cases["a/int64"] = number
	for branch, set := range map[string]func(*collision.Envelope){
		"b/boolean": func(e *collision.Envelope) { e.B = &collision.BooleanOrInt64{}; e.B.SetBoolean(true) },
		"b/int64":   func(e *collision.Envelope) { e.B = &collision.BooleanOrInt64{}; e.B.SetInt64(2) },
		"c/int64":   func(e *collision.Envelope) { e.C = &collision.Int64OrLeaf{}; e.C.SetInt64(3) },
		"c/leaf":    func(e *collision.Envelope) { e.C = &collision.Int64OrLeaf{}; e.C.SetLeaf(&collision.Leaf{Name: &name}) },
		"d/int64":   func(e *collision.Envelope) { e.D = &collision.D{}; e.D.SetInt64(4) },
		"d/leaf":    func(e *collision.Envelope) { e.D = &collision.D{}; e.D.SetLeaf("dd") },
	} {
		envelope := &collision.Envelope{Leaf: &label}
		envelope.A.SetInt64(5)
		set(envelope)
		cases[branch] = envelope
	}
	for name, choice := range choices() {
		envelope := &collision.Envelope{Pick: choice}
		envelope.A.SetString("pick")
		cases["pick/"+name] = envelope
		for second, other := range choices() {
			holder := &collision.Holder{First: *choice, Second: other, Tag: &collision.Tagged{}, Alt: &collision.Tagged{}}
			holder.Tag.SetCode("tag")
			holder.Alt.SetOther(&collision.Other{})
			envelope := &collision.Envelope{Holder: holder, Holders: []*collision.Holder{holder, {First: *other}}}
			envelope.A.SetString("holder")
			cases["holder/"+name+"/"+second] = envelope
		}
	}
	return cases
}

func TestEnvelopeRoundTrip(t *testing.T) {
	for name, envelope := range envelopes() {
		t.Run(name, func(t *testing.T) {
			var payload, result *collision.Envelope
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

func TestTreeRoundTrip(t *testing.T) {
	for name, choice := range choices() {
		for pick, other := range choices() {
			t.Run(name+"/"+pick, func(t *testing.T) {
				tree := &collision.Tree{Choice: other}
				if leaf, ok := choice.AsLeaf(); ok {
					tree.Leaf.SetLeaf(leaf)
				} else {
					value, _ := choice.AsOther()
					tree.Leaf.SetOther(value)
				}
				require.Equal(t, tree, server.NewGrowPayload(client.NewProtoGrowRequest(tree)))
				require.Equal(t, tree, client.NewGrowResult(server.NewProtoGrowResponse(tree)))
				require.NoError(t, server.ValidateGrowRequest(client.NewProtoGrowRequest(tree)))
			})
		}
	}
}

func TestMatchRoundTrip(t *testing.T) {
	label := "label"
	for name, choice := range choices() {
		t.Run(name, func(t *testing.T) {
			named := &collision.Named{Leaf: *choice, LeafOneof: &label}
			require.Equal(t, named, server.NewMatchPayload(client.NewProtoMatchRequest(named)))
			require.NoError(t, server.ValidateMatchRequest(client.NewProtoMatchRequest(named)))
			pair := &collision.Pair{U: &collision.LeafOrOther{}, W: &collision.W{}}
			if leaf, ok := choice.AsLeaf(); ok {
				pair.U.SetLeaf(leaf)
				pair.W.SetU(leaf)
			} else {
				other, _ := choice.AsOther()
				pair.U.SetOther(other)
				pair.W.SetV(other)
			}
			require.Equal(t, pair, client.NewMatchResult(server.NewProtoMatchResponse(pair)))
		})
	}
}

func TestProtoMessages(t *testing.T) {
	name := "leaf"
	label := "label"
	request := &pb.EchoRequest{
		A:      &pb.EchoRequest_Int64_{Int64_: 1},
		B:      &pb.EchoRequest_BInt64{BInt64: 2},
		C:      &pb.EchoRequest_CLeaf{CLeaf: &pb.Leaf{Name: &name}},
		Leaf:   &label,
		D:      &pb.EchoRequest_DLeaf{DLeaf: "dd"},
		Pick:   &pb.EchoRequest_PickLeaf{PickLeaf: &pb.Leaf{}},
		Holder: &pb.Holder{First: &pb.Holder_Other{Other: &pb.Other{}}, Second: &pb.Holder_SecondLeaf{SecondLeaf: &pb.Leaf{}}},
	}
	require.NoError(t, server.ValidateEchoRequest(request))
	payload := server.NewEchoPayload(request)
	a, ok := payload.A.AsInt64()
	require.True(t, ok)
	require.Equal(t, int64(1), a)
	b, ok := payload.B.AsInt64()
	require.True(t, ok)
	require.Equal(t, int64(2), b)
	c, ok := payload.C.AsLeaf()
	require.True(t, ok)
	require.Equal(t, "leaf", *c.Name)
	require.Equal(t, "label", *payload.Leaf)
	d, ok := payload.D.AsLeaf()
	require.True(t, ok)
	require.Equal(t, collision.DLeaf("dd"), d)
	_, ok = payload.Pick.AsLeaf()
	require.True(t, ok)
	_, ok = payload.Holder.First.AsOther()
	require.True(t, ok)
	_, ok = payload.Holder.Second.AsLeaf()
	require.True(t, ok)

	request.D = &pb.EchoRequest_DLeaf{DLeaf: "d"}
	require.ErrorContains(t, server.ValidateEchoRequest(request), "message.d.value", "the renamed d_leaf branch keeps its length validation")
	request.A = nil
	require.ErrorContains(t, server.ValidateEchoRequest(request), "\"a\"")
	require.ErrorContains(t, server.ValidateHolder(&pb.Holder{Second: &pb.Holder_SecondOther{SecondOther: &pb.Other{}}}), "\"first\"")
	holder := &pb.Holder{First: &pb.Holder_Leaf{Leaf: &pb.Leaf{}}, Tag: &pb.Holder_Code{Code: "ok"}, Alt: &pb.Holder_AltCode{AltCode: "ok"}}
	require.NoError(t, server.ValidateHolder(holder))
	holder.Alt = &pb.Holder_AltCode{AltCode: "x"}
	require.ErrorContains(t, server.ValidateHolder(holder), "holder.alt.value", "the second field of a named union validates its own renamed branch")
	holder.Alt = &pb.Holder_AltOther{AltOther: &pb.Other{}}
	holder.Tag = &pb.Holder_Code{Code: "x"}
	require.ErrorContains(t, server.ValidateHolder(holder), "holder.tag.value")
	decoded := server.NewEchoPayload(&pb.EchoRequest{A: &pb.EchoRequest_String_{String_: "s"}, Holder: &pb.Holder{Tag: &pb.Holder_TagOther{TagOther: &pb.Other{}}, Alt: &pb.Holder_AltCode{AltCode: "ok"}}})
	_, ok = decoded.Holder.Tag.AsOther()
	require.True(t, ok)
	code, ok := decoded.Holder.Alt.AsCode()
	require.True(t, ok)
	require.Equal(t, collision.Code("ok"), code)

	grow := &pb.GrowRequest{
		LeafOneof: &pb.GrowRequest_Other{Other: &pb.Other{}},
		Choice:    &pb.GrowRequest_ChoiceLeaf{ChoiceLeaf: &pb.Leaf{Name: &name}},
	}
	require.NoError(t, server.ValidateGrowRequest(grow))
	tree := server.NewGrowPayload(grow)
	_, ok = tree.Leaf.AsOther()
	require.True(t, ok)
	leaf, ok := tree.Choice.AsLeaf()
	require.True(t, ok)
	require.Equal(t, "leaf", *leaf.Name)
	require.ErrorContains(t, server.ValidateGrowRequest(&pb.GrowRequest{}), "\"leaf\"")

	match := &pb.MatchRequest{LeafOneofOneof: &pb.MatchRequest_Leaf{Leaf: &pb.Leaf{}}, LeafOneof: &label}
	require.NoError(t, server.ValidateMatchRequest(match))
	named := server.NewMatchPayload(match)
	_, ok = named.Leaf.AsLeaf()
	require.True(t, ok)
	require.Equal(t, "label", *named.LeafOneof)
	require.ErrorContains(t, server.ValidateMatchRequest(&pb.MatchRequest{LeafOneof: &label}), "\"leaf\"")

	pair := client.NewMatchResult(&pb.MatchResponse{
		U: &pb.MatchResponse_Other{Other: &pb.Other{}},
		W: &pb.MatchResponse_WU{WU: &pb.Leaf{Name: &name}},
	})
	_, ok = pair.U.AsOther()
	require.True(t, ok)
	u, ok := pair.W.AsU()
	require.True(t, ok)
	require.Equal(t, "leaf", *u.Name)
}
`
