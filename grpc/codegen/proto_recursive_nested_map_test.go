package codegen

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// TestRecursiveNestedCollectionMessages checks designs whose recursive named
// collection Index holds the type Holder, which holds Index, in a map or
// array nested in another map or array, and designs that nest the recursive
// named map Index in a map or array of a method result or payload. The
// messages that wrap the nested collections are converted inline, and the
// variables of nested map loops stay distinct, so the designs generate,
// protoc accepts them and the generated module compiles.
func TestRecursiveNestedCollectionMessages(t *testing.T) {
	cases := []struct {
		Name     string
		DSL      func()
		Contains []string
	}{
		{"map of arrays field", recursiveNestedCollectionDSL(mapOfArraysIndex, "", nil), []string{"message Index {\n\tmap<string, ArrayOfHolder> field = 1;\n}", "message ArrayOfHolder {\n\trepeated Holder field = 1;\n}"}},
		{"map of arrays result", recursiveNestedCollectionDSL(mapOfArraysIndex, "result", nil), []string{"rpc M (MRequest) returns (Index);"}},
		{"map of arrays payload", recursiveNestedCollectionDSL(mapOfArraysIndex, "payload", nil), []string{"rpc M (Index) returns (MResponse);"}},
		{"map of arrays streaming result", recursiveNestedCollectionDSL(mapOfArraysIndex, "streaming result", nil), []string{"rpc M (MRequest) returns (stream Index);"}},
		{"array of maps field", recursiveNestedCollectionDSL(arrayOfMapsIndex, "", nil), []string{"message Index {\n\trepeated MapOfStringHolder field = 1;\n}", "message MapOfStringHolder {\n\tmap<string, Holder> field = 1;\n}"}},
		{"array of maps result", recursiveNestedCollectionDSL(arrayOfMapsIndex, "result", nil), []string{"rpc M (MRequest) returns (Index);"}},
		{"map of maps field", recursiveNestedCollectionDSL(mapOfMapsIndex, "", nil), []string{"message Index {\n\tmap<string, MapOfStringHolder> field = 1;\n}"}},
		{"map of maps result", recursiveNestedCollectionDSL(mapOfMapsIndex, "result", nil), []string{"rpc M (MRequest) returns (Index);"}},
		{"map of index result", recursiveNestedCollectionDSL(mapIndex, "result", mapOfIndex), []string{"message MResponse {\n\tmap<string, Index> field = 1;\n}"}},
		{"map of index payload", recursiveNestedCollectionDSL(mapIndex, "payload", mapOfIndex), []string{"message MRequest {\n\tmap<string, Index> field = 1;\n}"}},
		{"array of index result", recursiveNestedCollectionDSL(mapIndex, "result", arrayOfIndex), []string{"message MResponse {\n\trepeated Index field = 1;\n}"}},
		{"map of maps of index result", recursiveNestedCollectionDSL(mapIndex, "result", mapOfMapsOfIndex), []string{"message MapOfStringIndex {\n\tmap<string, Index> field = 1;\n}"}},
		{"map of arrays of index result", recursiveNestedCollectionDSL(mapIndex, "result", mapOfArraysOfIndex), []string{"message ArrayOfIndex {\n\trepeated Index field = 1;\n}"}},
		{"map of map of arrays index result", recursiveNestedCollectionDSL(mapOfArraysIndex, "result", mapOfIndex), []string{"message MResponse {\n\tmap<string, Index> field = 1;\n}"}},
		{"map of map of maps index result", recursiveNestedCollectionDSL(mapOfMapsIndex, "result", mapOfIndex), []string{"message MResponse {\n\tmap<string, Index> field = 1;\n}"}},
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
			renderGRPCModule(t, dir, "example.com/recursivenested", RunGRPCDSL(t, c.DSL), resolveGRPCLoomSource(t))
			runGRPCGoCommand(t, dir, "mod", "tidy")
			runGRPCGoCommand(t, dir, "vet", "./...")
		})
	}
}

// TestGeneratedRecursiveNestedCollectionRoundTrip compiles the modules
// generated for methods whose payload and result are the type Holder, which
// holds the recursive named collection Index of nested collections of
// Holder, and for methods whose result nests the recursive named map Index
// in maps, and round-trips nested values through the generated protobuf
// conversions.
func TestGeneratedRecursiveNestedCollectionRoundTrip(t *testing.T) {
	cases := []struct {
		Name    string
		DSL     func()
		Harness string
	}{
		{"map of arrays", recursiveNestedCollectionDSL(mapOfArraysIndex, "", nil), fmt.Sprintf(recursiveHolderRoundTripHarness, "%[1]s",
			`svc.Index{"a": {leaf, holder("b", svc.Index{"c": {leaf}, "d": {}})}, "e": {}}`)},
		{"array of maps", recursiveNestedCollectionDSL(arrayOfMapsIndex, "", nil), fmt.Sprintf(recursiveHolderRoundTripHarness, "%[1]s",
			`svc.Index{{"a": leaf}, {"b": holder("b", svc.Index{{"c": leaf}, {}})}}`)},
		{"map of maps", recursiveNestedCollectionDSL(mapOfMapsIndex, "", nil), fmt.Sprintf(recursiveHolderRoundTripHarness, "%[1]s",
			`svc.Index{"a": {"b": leaf}, "c": {"d": holder("d", svc.Index{"e": {"f": leaf}, "g": {}})}}`)},
		{"map of index", recursiveNestedCollectionDSL(mapIndex, "result", mapOfIndex), fmt.Sprintf(recursiveIndexResultRoundTripHarness, "%[1]s",
			"map[string]svc.Index", `{"a": {"b": leaf, "c": holder("c", svc.Index{"d": leaf, "e": holder("e", svc.Index{})})}, "f": {}}`)},
		{"map of maps of index", recursiveNestedCollectionDSL(mapIndex, "result", mapOfMapsOfIndex), fmt.Sprintf(recursiveIndexResultRoundTripHarness, "%[1]s",
			"map[string]map[string]svc.Index", `{"a": {"b": {"c": leaf, "d": holder("d", svc.Index{"e": leaf})}, "f": {}}, "g": {}}`)},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			runGeneratedRoundTrip(t, "example.com/recursivenested", c.DSL, c.Harness)
		})
	}
}

// TestNestedCollectionMessageInlined checks that the message that wraps an
// array or map nested in a map or array is a collection message that
// transform code converts inline, while the type Holder that reaches itself
// through that message still converts through a helper.
func TestNestedCollectionMessageInlined(t *testing.T) {
	root := RunGRPCDSL(t, recursiveNestedCollectionDSL(mapOfArraysIndex, "", nil))
	sd := CreateGRPCServices(root).Get("svc")
	holder := makeProtoBufMessage(&expr.AttributeExpr{Type: root.UserType("Holder")}, "Holder", sd)
	index := holder.Find("index")
	require.NotNil(t, index)
	indexType, ok := index.Type.(expr.UserType)
	require.True(t, ok, "index type %T", index.Type)
	assert.True(t, isCollectionMessage(indexType))
	elem := expr.AsMap(unwrapAttr(indexType.Attribute()).Type).ElemType
	wrapper, ok := elem.Type.(expr.UserType)
	require.True(t, ok, "map value type %T", elem.Type)
	assert.Equal(t, "ArrayOfHolder", wrapper.Name())
	assert.True(t, isCollectionMessage(wrapper))
	assert.False(t, isInlineRecursive(elem))
	assert.True(t, isInlineRecursive(expr.AsArray(unwrapAttr(wrapper.Attribute()).Type).ElemType))
}

// TestTransformMapElemVar checks that the variable of the converted value of
// a map loop never shadows the variable that the target map starts with.
func TestTransformMapElemVar(t *testing.T) {
	cases := []struct {
		Suffix string
		MapVar string
		Want   string
	}{
		{"", "res.Index", "tv"},
		{"b", "message.Field", "tvb"},
		{"b", "tvb.Field", "tvc"},
		{"b", "tvb[i].Field", "tvc"},
		{"b", "tvbc.Field", "tvb"},
		{"", "tv.Field", "tva"},
		{"b", "tvb", "tvc"},
	}
	for _, c := range cases {
		assert.Equal(t, c.Want, transformMapElemVar(c.Suffix, c.MapVar), "suffix %q, map %q", c.Suffix, c.MapVar)
	}
}

// recursiveNestedCollectionDSL declares the type Holder whose index field
// holds the named collection Index that index returns for Holder. The method
// m uses Holder as its payload and result when place is empty. Otherwise it
// uses the value that use returns for Index, or Index itself when use is nil,
// as the payload, the result or the streaming result that place names, and
// Holder as the other of the payload and the result.
func recursiveNestedCollectionDSL(index func(any) any, place string, use func(any) any) func() {
	return func() {
		h := Type("Holder", func() {
			Field(1, "label", String)
			Field(2, "index", "Index")
		})
		i := Type("Index", index(h))
		var v any = i
		if use != nil {
			v = use(i)
		}
		Service("svc", func() {
			Method("m", func() {
				switch place {
				case "":
					Payload(h)
					Result(h)
				case "payload":
					Payload(v)
					Result(h)
				case "result":
					Payload(h)
					Result(v)
				case "streaming result":
					Payload(h)
					StreamingResult(v)
				}
				GRPC(func() {})
			})
		})
	}
}

func mapOfArraysIndex(h any) any {
	return MapOf(String, ArrayOf(h))
}

func arrayOfMapsIndex(h any) any {
	return ArrayOf(MapOf(String, h))
}

func mapOfMapsIndex(h any) any {
	return MapOf(String, MapOf(String, h))
}

func mapIndex(h any) any {
	return MapOf(String, h)
}

func mapOfIndex(i any) any {
	return MapOf(String, i)
}

func arrayOfIndex(i any) any {
	return ArrayOf(i)
}

func mapOfMapsOfIndex(i any) any {
	return MapOf(String, MapOf(String, i))
}

func mapOfArraysOfIndex(i any) any {
	return MapOf(String, ArrayOf(i))
}

const recursiveHolderRoundTripHarness = `package roundtrip

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
	for name, value := range map[string]*svc.Holder{
		"leaf":   leaf,
		"empty":  holder("empty", svc.Index{}),
		"nested": holder("nested", %[2]s),
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, value, server.NewMPayload(client.NewProtoMRequest(value)))
			require.Equal(t, value, client.NewMResult(server.NewProtoMResponse(value)))
		})
	}
}
`

const recursiveIndexResultRoundTripHarness = `package roundtrip

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
	for name, value := range map[string]%[2]s{
		"empty":  {},
		"nested": %[3]s,
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, value, client.NewMResult(server.NewProtoMResponse(value)))
		})
	}
}
`
