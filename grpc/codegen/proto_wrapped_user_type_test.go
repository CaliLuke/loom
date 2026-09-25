package codegen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// TestWrapAttrUserTypeIdentity checks that the message that wraps a named
// primitive, a named union or an anonymous collection is a new user type
// with its own identifier. Copies and examples share user types by
// identifier, so a wrapper that kept the identifier of the named type would
// replace the named type wherever else it appears.
func TestWrapAttrUserTypeIdentity(t *testing.T) {
	uid := &expr.UserTypeExpr{TypeName: "UID", AttributeExpr: &expr.AttributeExpr{Type: expr.String}, UID: "UID"}
	leaf := &expr.UserTypeExpr{TypeName: "Leaf", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{}}, UID: "Leaf"}
	union := &expr.UserTypeExpr{
		TypeName: "U",
		AttributeExpr: &expr.AttributeExpr{Type: &expr.Union{
			TypeName: "U",
			Values:   []*expr.NamedAttributeExpr{{Name: "Leaf", Attribute: &expr.AttributeExpr{Type: leaf}}},
		}},
		UID: "U",
	}
	cases := []struct {
		Name     string
		Type     expr.DataType
		WantName string
		WantID   string
	}{
		{"named primitive", uid, "UID", "UID#message"},
		{"named union", union, "U", "U#message"},
		{"array", &expr.Array{ElemType: &expr.AttributeExpr{Type: uid}}, "MResponse", "svc#MResponse"},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			att := &expr.AttributeExpr{Type: c.Type}
			wrapAttr(att, "MResponse", true, &ServiceData{Name: "svc"})
			ut, ok := att.Type.(*expr.UserTypeExpr)
			require.True(t, ok, "wrapper type %T", att.Type)
			assert.Equal(t, c.WantName, ut.Name())
			assert.Equal(t, c.WantID, ut.ID())
			if named, ok := c.Type.(expr.UserType); ok {
				assert.NotSame(t, named, ut)
				assert.NotEqual(t, named.ID(), ut.ID())
			}
		})
	}
	assert.Equal(t, expr.String, uid.Attribute().Type, "named primitive left unchanged")
}

// TestNamedPrimitiveMessage checks designs that use a named primitive
// directly as a payload, a result and a streaming result, and as a field, an
// array element and a map value elsewhere. The message that wraps the direct
// use must not replace the named primitive in the other positions, so the
// designs generate, protoc accepts them, and the generated module compiles
// and round-trips the values through the generated conversions.
func TestNamedPrimitiveMessage(t *testing.T) {
	var code string
	require.NoError(t, generationError(func() {
		code = protoFileCode(t, namedPrimitiveMessageDSL)
	}))
	assert.Contains(t, code, "message UID {\n\tstring field = 1;\n}")
	assert.Contains(t, code, "rpc A (UID) returns (AResponse);")
	assert.Contains(t, code, "rpc B (BRequest) returns (UID);")
	assert.Contains(t, code, "rpc C (CRequest) returns (stream UID);")
	assert.Contains(t, code, "message AResponse {\n\tstring id = 1;\n\trepeated string ids = 2;\n\tmap<string, string> by_key = 3;\n}")
	fpath := codegen.CreateTempFile(t, code)
	assert.NoError(t, protoc(defaultProtocCmd, fpath, nil), "compile proto file %q", fpath)
	runGeneratedRoundTrip(t, "example.com/namedprimitive", namedPrimitiveMessageDSL, namedPrimitiveRoundTripHarness)
}

// TestRecursiveMapResult checks designs where a named map whose values reach
// the map again through a field is used directly as a method payload,
// result, streaming payload or streaming result, alone or through a named
// alias. The message that wraps the direct use is the message of the named
// map, so generation terminates, protoc accepts the proto file and the
// generated module compiles.
func TestRecursiveMapResult(t *testing.T) {
	cases := []struct {
		Name     string
		DSL      func()
		Contains []string
	}{
		{"result", func() { recursiveMapResultDSL(Payload, Result) }, []string{"rpc M (MRequest) returns (Index);"}},
		{"payload", func() { recursiveMapResultDSL(Result, Payload) }, []string{"rpc M (Index) returns (MResponse);"}},
		{"streaming result", func() { recursiveMapResultDSL(Payload, StreamingResult) }, []string{"rpc M (MRequest) returns (stream Index);"}},
		{"streaming payload", func() { recursiveMapResultDSL(Result, StreamingPayload) }, []string{"rpc M (stream MStreamingRequest) returns (MResponse);"}},
		{"alias result", recursiveMapAliasResultDSL, []string{"rpc M (MRequest) returns (More);", "message More {\n\tmap<string, Holder> field = 1;\n}"}},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			var code string
			require.NoError(t, generationError(func() {
				code = protoFileCode(t, c.DSL)
			}))
			for _, want := range c.Contains {
				assert.Contains(t, code, want)
			}
			assert.Contains(t, code, "message Holder {\n\toptional string label = 1;\n\tIndex index = 2;\n}")
			fpath := codegen.CreateTempFile(t, code)
			require.NoError(t, protoc(defaultProtocCmd, fpath, nil), "compile proto file %q", fpath)
			dir := t.TempDir()
			renderGRPCModule(t, dir, "example.com/recursivemapresult", RunGRPCDSL(t, c.DSL), resolveGRPCLoomSource(t))
			runGRPCGoCommand(t, dir, "mod", "tidy")
			runGRPCGoCommand(t, dir, "vet", "./...")
		})
	}
}

// TestGeneratedRecursiveMapResultRoundTrip compiles the module generated for
// a method whose payload is the type Holder and whose result is the named
// map Index of Holder that Holder holds, and round-trips nested values
// through the generated protobuf conversions.
func TestGeneratedRecursiveMapResultRoundTrip(t *testing.T) {
	runGeneratedRoundTrip(t, "example.com/recursivemapresult", func() { recursiveMapResultDSL(Payload, Result) }, recursiveMapResultRoundTripHarness)
}

// recursiveMapResultDSL declares the type Holder whose index field holds the
// named map Index of Holder. The method m passes Holder to holder and Index
// to index, such as Payload and Result.
func recursiveMapResultDSL(holder, index func(any, ...any)) {
	h := Type("Holder", func() {
		Field(1, "label", String)
		Field(2, "index", "Index")
	})
	i := Type("Index", MapOf(String, h))
	Service("svc", func() {
		Method("m", func() {
			holder(h)
			index(i)
			GRPC(func() {})
		})
	})
}

// recursiveMapAliasResultDSL uses the alias More of the recursive named map
// Index as the result of the method m.
func recursiveMapAliasResultDSL() {
	h := Type("Holder", func() {
		Field(1, "label", String)
		Field(2, "index", "Index")
	})
	i := Type("Index", MapOf(String, h))
	more := Type("More", i)
	Service("svc", func() {
		Method("m", func() {
			Payload(h)
			Result(more)
			GRPC(func() {})
		})
	})
}

// namedPrimitiveMessageDSL uses the named primitive UID directly as the
// payload of a, the result of b and the streaming result of c, and as a
// field, an array element and a map value of Holder.
func namedPrimitiveMessageDSL() {
	uid := Type("UID", String, func() {
		MinLength(2)
	})
	holder := Type("Holder", func() {
		Field(1, "id", uid)
		Field(2, "ids", ArrayOf(uid))
		Field(3, "by_key", MapOf(String, uid))
		Required("id")
	})
	Service("svc", func() {
		Method("a", func() {
			Payload(uid)
			Result(holder)
			GRPC(func() {})
		})
		Method("b", func() {
			Payload(holder)
			Result(uid)
			GRPC(func() {})
		})
		Method("c", func() {
			Payload(holder)
			StreamingResult(uid)
			GRPC(func() {})
		})
	})
}

const recursiveMapResultRoundTripHarness = `package roundtrip

import (
	"testing"

	"github.com/stretchr/testify/require"

	"%[1]s/gen/grpc/svc/client"
	"%[1]s/gen/grpc/svc/server"
	svc "%[1]s/gen/svc"
)

func holder(label string, index svc.Index) *svc.Holder {
	return &svc.Holder{Label: &label, Index: index}
}

func TestRoundTrip(t *testing.T) {
	leaf := holder("leaf", nil)
	nested := holder("nested", svc.Index{"leaf": leaf, "empty": holder("empty", svc.Index{})})
	for name, index := range map[string]svc.Index{
		"empty":  {},
		"flat":   {"a": leaf},
		"nested": {"n": nested, "deep": holder("deep", svc.Index{"n": nested})},
	} {
		t.Run(name, func(t *testing.T) {
			var result svc.Index
			require.NotPanics(t, func() {
				result = client.NewMResult(server.NewProtoIndex(index))
			})
			require.Equal(t, index, result)
			payload := holder("payload", index)
			require.Equal(t, payload, server.NewMPayload(client.NewProtoMRequest(payload)))
		})
	}
}
`

const namedPrimitiveRoundTripHarness = `package roundtrip

import (
	"testing"

	"github.com/stretchr/testify/require"

	"%[1]s/gen/grpc/svc/client"
	pb "%[1]s/gen/grpc/svc/pb"
	"%[1]s/gen/grpc/svc/server"
	svc "%[1]s/gen/svc"
)

func TestDirectRoundTrip(t *testing.T) {
	id := svc.UID("abc")
	require.Equal(t, "abc", client.NewProtoUID(id).GetField())
	require.Equal(t, id, server.NewAPayload(client.NewProtoUID(id)))
	require.Equal(t, id, client.NewBResult(server.NewProtoUID(id)))
	require.Equal(t, id, client.NewUIDUID(server.NewProtoUIDUID(id)))
}

func TestHolderRoundTrip(t *testing.T) {
	holder := &svc.Holder{
		ID:    "abc",
		Ids:   []svc.UID{"de", "fgh"},
		ByKey: map[string]svc.UID{"k": "ij"},
	}
	require.Equal(t, holder, server.NewBPayload(client.NewProtoBRequest(holder)))
	require.Equal(t, holder, server.NewCPayload(client.NewProtoCRequest(holder)))
	require.Equal(t, holder, client.NewAResult(server.NewProtoAResponse(holder)))
}

func TestValidation(t *testing.T) {
	require.NoError(t, server.ValidateUID(&pb.UID{Field: "ab"}))
	require.Error(t, server.ValidateUID(&pb.UID{Field: "a"}))
	require.NoError(t, client.ValidateUID(&pb.UID{Field: "ab"}))
	require.Error(t, client.ValidateUID(&pb.UID{Field: "a"}))
	require.NoError(t, server.ValidateBRequest(&pb.BRequest{Id: "ab", Ids: []string{"cd"}}))
	require.Error(t, server.ValidateBRequest(&pb.BRequest{Id: "a"}))
	require.Error(t, server.ValidateBRequest(&pb.BRequest{Id: "ab", Ids: []string{"c"}}))
}
`
