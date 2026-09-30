package valuecontract

import (
	"encoding/json/jsontext"
	"slices"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

type productionCase struct {
	name      string
	attribute *expr.AttributeExpr
	raw       any
}

func checkProductionSourceConformance(t *testing.T, executable string) {
	t.Helper()
	cases := productionResolutionCases()
	require.NotEmpty(t, cases)
	for _, tc := range cases {
		for _, role := range []struct {
			name  string
			value expr.ValueRole
		}{
			{"authoredExample", expr.ValueRoleExample},
			{"enumMember", expr.ValueRoleEnum},
			{"defaultValue", expr.ValueRoleDefault},
		} {
			t.Run(tc.name+"/"+role.name, func(t *testing.T) {
				context := expr.NewValueContext()
				occurrence, err := context.NewOccurrence(tc.attribute)
				require.NoError(t, err)
				input := newProductionInput(t)
				graph := newProductionGraph(t, input)
				root := graph.capture(tc.attribute, occurrence)
				graph.ranks()
				supplied := input.encode(tc.raw)
				request := map[string]any{
					"declarations": graph.declarations, "root": root,
					"supplied": map[string]any{"source": map[string]any{"occurrence": root, "origin": 1, "role": role.name}, "value": supplied},
					"codecs":   input.codecs(), "checks": []any{}, "projection": nil,
				}
				actual := referenceDecode[map[string]jsontext.Value](t, runReference(t, executable, []any{referenceConstructor("evaluate", map[string]any{"request": request})})[0])
				require.Equal(t, "true", string(actual["declarationsValid"]), "reference graph invalid or missing codec boundary: %s", productionJSON(t, actual))
				source := context.SupplyValue(expr.ValueInput{Raw: tc.raw})
				result := context.Resolve(occurrence, source, role.value)
				require.True(t, source.ID() == result.SourceID())
				require.True(t, occurrence.ID() == result.OccurrenceID())
				require.Equal(t, role.value, result.Role())
				if tc.name == "map enum integer host normalization" {
					require.Equal(t, expr.ValueResolved, result.Outcome())
				}
				expected := graph.output(input, result, tc.raw)
				// Canonicalize JSON syntax through jsontext values, never float64:
				// arbitrary integer coefficients and all identity paths stay exact.
				require.Equal(t, productionCanonical(t, jsontext.Value(productionJSON(t, expected))), productionCanonical(t, actual["resolved"]))
			})
		}
	}
}

func productionResolutionCases() []productionCase {
	object := func(name string, typ expr.DataType) *expr.AttributeExpr {
		return &expr.AttributeExpr{Type: &expr.Object{{Name: name, Attribute: &expr.AttributeExpr{Type: typ}}}, Validation: &expr.ValidationExpr{Required: []string{name}}}
	}
	choice := &expr.Union{Values: []*expr.NamedAttributeExpr{
		{Name: "A", Attribute: object("a", expr.String)},
		{Name: "B", Attribute: object("b", expr.Boolean)},
	}}
	array := &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.UInt}}
	bytesChoice := &expr.Union{Values: []*expr.NamedAttributeExpr{
		{Name: "Bytes", Attribute: &expr.AttributeExpr{Type: expr.Bytes}},
		{Name: "Array", Attribute: &expr.AttributeExpr{Type: array}},
	}}
	recursive := &expr.UserTypeExpr{TypeName: "Recursive", AttributeExpr: &expr.AttributeExpr{}}
	recursive.Type = &expr.Object{
		{Name: "name", Attribute: &expr.AttributeExpr{Type: expr.String}},
		{Name: "next", Attribute: &expr.AttributeExpr{Type: recursive, Nullable: true}},
	}
	recursive.Validation = &expr.ValidationExpr{Required: []string{"name"}}
	cycle := map[string]any{"name": "root"}
	cycle["next"] = cycle
	child := object("required", expr.String)
	partial := &expr.AttributeExpr{Type: &expr.Object{{Name: "child", Attribute: child}}, Validation: &expr.ValidationExpr{Required: []string{"child"}}}
	cases := []productionCase{
		{"string", &expr.AttributeExpr{Type: expr.String}, "hi"},
		{"integer exact", &expr.AttributeExpr{Type: expr.Int64}, uint64(9007199254740993)},
		{"invalid scalar", &expr.AttributeExpr{Type: expr.Boolean}, "true"},
		{"null scalar", &expr.AttributeExpr{Type: expr.String}, nil},
		{"nullable scalar", &expr.AttributeExpr{Type: expr.String, Nullable: true}, nil},
		{"bytes literal", &expr.AttributeExpr{Type: expr.Bytes}, "aGk="},
		{"nil bytes", &expr.AttributeExpr{Type: expr.Bytes}, []byte(nil)},
		{"nil array", &expr.AttributeExpr{Type: array}, []uint(nil)},
		{"empty array", &expr.AttributeExpr{Type: array}, []uint{}},
		{"byte slice as array", &expr.AttributeExpr{Type: array}, []byte("hi")},
		{"byte array as array", &expr.AttributeExpr{Type: array}, [2]byte{104, 105}},
		{"byte slice union ambiguity", &expr.AttributeExpr{Type: bytesChoice}, []byte("hi")},
		{"byte array union", &expr.AttributeExpr{Type: bytesChoice}, [2]byte{104, 105}},
		{"Any bytes", &expr.AttributeExpr{Type: expr.Any}, []byte("hi")},
		{"Any byte array", &expr.AttributeExpr{Type: expr.Any}, [2]byte{104, 105}},
		{"Any nil map", &expr.AttributeExpr{Type: expr.Any}, map[int]string(nil)},
		{"Any heterogeneous map", &expr.AttributeExpr{Type: expr.Any}, map[any]string{1: "one", true: "true"}},
		{"map collision", &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.Any}, ElemType: &expr.AttributeExpr{Type: expr.String}}}, map[any]string{1: "one", "1": "string"}},
		{"map mixed float widths", &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.Any}, ElemType: &expr.AttributeExpr{Type: expr.String}}}, map[any]string{float32(.1): "narrow", float64(float32(.1)): "wide"}},
		{"array required Any null", &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.Any}, NonNullableElems: true}}, []any{nil}},
		{"partial object", object("name", expr.String), map[string]any{}},
		{"null required field", object("name", expr.String), map[string]any{"name": nil}},
		{"nested missing path", partial, map[string]any{"child": map[string]any{}}},
		{"unknown object field", object("name", expr.String), map[string]any{"name": "hi", "extra": []int32{1, 2}}},
		{"valid aliases", object("name:wire", expr.String), map[string]any{"name:wire": "loser", "wire": "winner"}},
		{"invalid losing alias", object("name:wire", expr.String), map[string]any{"name:wire": 1, "wire": "winner"}},
		{"overlapping aliases", &expr.AttributeExpr{Type: &expr.Object{{Name: "a:b", Attribute: &expr.AttributeExpr{Type: expr.String}}, {Name: "b:c", Attribute: &expr.AttributeExpr{Type: expr.Int}}}}, map[string]any{"b": 1}},
		{"complete union", &expr.AttributeExpr{Type: choice}, map[string]any{"a": "hi"}},
		{"ambiguous partial union", &expr.AttributeExpr{Type: choice}, map[string]any{}},
		{"recursive value", &expr.AttributeExpr{Type: recursive}, map[string]any{"name": "root", "next": map[string]any{"name": "child", "next": nil}}},
		{"recursive cycle", &expr.AttributeExpr{Type: recursive}, cycle},
		{"opaque Any", &expr.AttributeExpr{Type: expr.Any}, struct{ Name string }{"opaque"}},
		{"scalar enum valid", &expr.AttributeExpr{Type: expr.String, Validation: &expr.ValidationExpr{Values: []any{"allowed"}}}, "allowed"},
		{"scalar enum invalid", &expr.AttributeExpr{Type: expr.String, Validation: &expr.ValidationExpr{Values: []any{"allowed"}}}, "other"},
		{"array enum invalid", &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}, Validation: &expr.ValidationExpr{Values: []any{[]string{"allowed"}}}}, []string{"other"}},
		{"Any host map enum self", &expr.AttributeExpr{Type: expr.Any, Validation: &expr.ValidationExpr{Values: []any{map[int]string{1: "one"}}}}, map[int]string{1: "one"}},
		{"Any host map enum mismatch", &expr.AttributeExpr{Type: expr.Any, Validation: &expr.ValidationExpr{Values: []any{map[int]string{1: "one"}}}}, map[int64]string{1: "one"}},
	}
	return slices.Concat(cases, productionNestedCases(), productionNativeByteCases())
}

func productionNestedCases() []productionCase {
	leaf := func(name string) *expr.AttributeExpr {
		return &expr.AttributeExpr{Type: &expr.Object{{Name: name, Attribute: &expr.AttributeExpr{Type: expr.String}}}, Validation: &expr.ValidationExpr{Required: []string{name}}}
	}
	inner := &expr.Union{Values: []*expr.NamedAttributeExpr{{Name: "A", Attribute: leaf("a")}, {Name: "B", Attribute: leaf("b")}}}
	nested := &expr.AttributeExpr{Type: &expr.Object{{Name: "choice", Attribute: &expr.AttributeExpr{Type: inner}}}, Validation: &expr.ValidationExpr{Required: []string{"choice"}}}
	outer := func(other *expr.AttributeExpr) *expr.AttributeExpr {
		return &expr.AttributeExpr{Type: &expr.Union{Values: []*expr.NamedAttributeExpr{{Name: "Nested", Attribute: nested}, {Name: "Other", Attribute: other}}}}
	}
	open := &expr.AttributeExpr{Type: &expr.Object{}}
	preferred := &expr.AttributeExpr{Type: &expr.Object{{Name: "b", Attribute: &expr.AttributeExpr{Type: expr.String}}, {Name: "required", Attribute: &expr.AttributeExpr{Type: expr.String}}}, Validation: &expr.ValidationExpr{Required: []string{"required"}}}
	anyChoice := &expr.AttributeExpr{Type: &expr.Union{Values: []*expr.NamedAttributeExpr{{Name: "Object", Attribute: open}, {Name: "Any", Attribute: &expr.AttributeExpr{Type: expr.Any}}}}}
	maxZero := 0
	cyclic := []any{nil}
	cyclic[0] = cyclic
	return []productionCase{
		{"nested ambiguity complete alternative", outer(open), map[string]any{"choice": map[string]any{}}},
		{"nested ambiguity preferred fallback", outer(leaf("other")), map[string]any{"choice": map[string]any{}}},
		{"nested ambiguity competing preference", outer(preferred), map[string]any{"choice": map[string]any{}, "b": "present"}},
		{"nested unique complete", outer(leaf("other")), map[string]any{"choice": map[string]any{"a": "one"}}},
		{"Any object preference", anyChoice, map[string]any{"unknown": "value"}},
		{"bounded cycle error priority", &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.Any}}, Validation: &expr.ValidationExpr{MaxLength: &maxZero}}, cyclic},
		{"array enum valid", &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}, Validation: &expr.ValidationExpr{Values: []any{[]string{"allowed"}}}}, []string{"allowed"}},
		{"map enum canonical spelling", &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.Any}, ElemType: &expr.AttributeExpr{Type: expr.String}}, Validation: &expr.ValidationExpr{Values: []any{map[any]string{float32(.1): "value"}}}}, map[any]string{float64(float32(.1)): "value"}},
		{"map enum integer host normalization", &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.Int}, ElemType: &expr.AttributeExpr{Type: expr.String}}, Validation: &expr.ValidationExpr{Values: []any{map[int32]string{1: "value"}}}}, map[int]string{1: "value"}},
		{"Any nested dynamic host enum mismatch", &expr.AttributeExpr{Type: expr.Any, Validation: &expr.ValidationExpr{Values: []any{map[int]any{1: int(1)}}}}, map[int]any{1: int64(1)}},
	}
}

type productionNamedBytes []byte
type productionNamedByteArray [2]byte
type productionDefinedByte byte
type productionStringByte byte
type productionCustomBytes []byte

var productionStringByteCalls atomic.Uint64

func (productionStringByte) String() string {
	productionStringByteCalls.Add(1)
	panic("inadmissible source Stringer must not run")
}

func (productionCustomBytes) MarshalJSON() ([]byte, error) {
	panic("source capture must not invoke a custom codec")
}

func productionNativeByteCases() []productionCase {
	values := []struct {
		name string
		raw  any
	}{
		{"native slice", []byte{104, 105}}, {"native nil slice", []byte(nil)}, {"native empty slice", []byte{}},
		{"native array", [2]byte{104, 105}}, {"native empty array", [0]byte{}},
		{"named slice", productionNamedBytes{104, 105}}, {"named nil slice", productionNamedBytes(nil)}, {"named empty slice", productionNamedBytes{}},
		{"named array", productionNamedByteArray{104, 105}},
		{"defined element slice", []productionDefinedByte{104, 105}}, {"defined element array", [2]productionDefinedByte{104, 105}},
		{"Stringer element slice", []productionStringByte{104, 105}}, {"Stringer element array", [2]productionStringByte{104, 105}},
		{"custom boundary", productionCustomBytes{104, 105}},
	}
	array := func(element expr.DataType) expr.DataType {
		return &expr.Array{ElemType: &expr.AttributeExpr{Type: element}}
	}
	types := []struct {
		name string
		data expr.DataType
	}{
		{"Any", expr.Any}, {"ArrayUInt", array(expr.UInt)}, {"ArrayFloat", array(expr.Float64)}, {"ArrayAny", array(expr.Any)}, {"Bytes", expr.Bytes},
		{"BytesArrayUnion", &expr.Union{Values: []*expr.NamedAttributeExpr{{Name: "Bytes", Attribute: &expr.AttributeExpr{Type: expr.Bytes}}, {Name: "Array", Attribute: &expr.AttributeExpr{Type: array(expr.UInt)}}}}},
	}
	cases := make([]productionCase, 0, len(values)*len(types))
	for _, value := range values {
		for _, typ := range types {
			cases = append(cases, productionCase{"byte shape/" + value.name + "/" + typ.name, &expr.AttributeExpr{Type: typ.data}, value.raw})
		}
	}
	for _, value := range []struct {
		name string
		raw  any
	}{
		{"array", [2]byte{104, 105}}, {"named slice", productionNamedBytes{104, 105}}, {"nil slice", []byte(nil)},
	} {
		attribute := &expr.AttributeExpr{Type: expr.Any, Validation: &expr.ValidationExpr{Values: []any{value.raw}}}
		cases = append(cases, productionCase{"byte enum/" + value.name, attribute, value.raw})
	}
	return cases
}

func TestProductionRejectedStringerNotInvoked(t *testing.T) {
	productionStringByteCalls.Store(0)
	input := newProductionInput(t)
	input.encode([]productionStringByte{104, 105})
	require.Zero(t, productionStringByteCalls.Load())
}
