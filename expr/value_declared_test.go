package expr

import (
	"encoding/json/jsontext"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

type DeclaredEmbedded struct {
	Name string `json:"name"`
}

type DeclaredNested struct {
	Blob string `json:"b"`
}

type declaredRecord struct {
	DeclaredEmbedded
	Score   float64         `json:"rating"`
	Nested  *DeclaredNested `json:"nested"`
	Ignored string          `json:"-"`
}

type declaredCycle struct {
	Next *declaredCycle `json:"next"`
}

type declaredOptionalEmbedded struct {
	*DeclaredEmbedded
}

type declaredCodec struct {
	Calls *int
}

func (value declaredCodec) MarshalJSON() ([]byte, error) {
	(*value.Calls)++
	return []byte(`"codec"`), nil
}

func TestDeclaredValueUsesOwnedPlainStructSemantics(t *testing.T) {
	nested := &Object{{Name: "blob:b", Attribute: &AttributeExpr{Type: Bytes}}}
	record := &Object{
		{Name: "name", Attribute: &AttributeExpr{Type: String}},
		{Name: "score:rating", Attribute: &AttributeExpr{Type: Float32}},
		{Name: "nested", Attribute: &AttributeExpr{Type: nested}},
	}
	attribute := &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: record}}}
	raw := []declaredRecord{{
		DeclaredEmbedded: DeclaredEmbedded{Name: "first"},
		Score:            1.23456789,
		Nested:           &DeclaredNested{Blob: "hi"},
		Ignored:          "ignored",
	}}
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	source := context.SupplyValue(ValueInput{Raw: raw})
	raw[0].Score = 9
	raw[0].Nested.Blob = "changed"
	result := context.Resolve(occurrence, source, ValueRoleEnum)
	require.Equal(t, ValueResolved, result.Outcome(), "%v", result.Diagnostics())
	declared, ok := result.DeclaredJSONValue()
	require.True(t, ok)
	require.Equal(t, []any{map[string]any{
		"name": "first", "rating": float32(1.2345679),
		"nested": map[string]any{"b": []byte("hi")},
	}}, declared)
}

func TestDeclaredValuePlainStructUnionAndFieldSelection(t *testing.T) {
	object := &Object{{Name: "value", Attribute: &AttributeExpr{Type: String}}}
	union := &Union{Values: []*NamedAttributeExpr{
		{Name: "record", Attribute: &AttributeExpr{Type: object}},
		{Name: "text", Attribute: &AttributeExpr{Type: String}},
	}}
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(&AttributeExpr{Type: union})
	require.NoError(t, err)
	result := context.Resolve(
		occurrence,
		context.SupplyValue(ValueInput{Raw: struct {
			Value string `json:"value"`
		}{Value: "selected"}}),
		ValueRoleEnum,
	)
	require.Equal(t, ValueResolved, result.Outcome())
	declared, ok := result.DeclaredJSONValue()
	require.True(t, ok)
	require.Equal(t, map[string]any{"type": "record", "value": map[string]any{"value": "selected"}}, declared)

	for _, test := range []struct {
		name     string
		required bool
		outcome  ValueOutcome
	}{
		{name: "optional ambiguous field is omitted", outcome: ValueResolved},
		{name: "required ambiguous field is missing", required: true, outcome: ValueInvalid},
	} {
		t.Run(test.name, func(t *testing.T) {
			attribute := &AttributeExpr{Type: object}
			if test.required {
				attribute.Validation = &ValidationExpr{Required: []string{"value"}}
			}
			occurrence, err := context.NewOccurrence(attribute)
			require.NoError(t, err)
			result := context.Resolve(
				occurrence,
				context.SupplyValue(ValueInput{Raw: ambiguousDeclaredStruct()}),
				ValueRoleEnum,
			)
			require.Equal(t, test.outcome, result.Outcome())
		})
	}

	closed := &AttributeExpr{Type: object, Meta: MetaExpr{"openapi:additionalProperties": []string{"false"}}}
	closedOccurrence, err := context.NewOccurrence(closed)
	require.NoError(t, err)
	result = context.Resolve(
		closedOccurrence,
		context.SupplyValue(ValueInput{Raw: struct {
			Value string `json:"value"`
			Extra string `json:"extra"`
		}{Value: "value", Extra: "extra"}}),
		ValueRoleEnum,
	)
	require.Equal(t, ValueInvalid, result.Outcome())

	embeddedObject := &Object{{Name: "name", Attribute: &AttributeExpr{Type: String}}}
	for _, test := range []struct {
		name string
		raw  declaredOptionalEmbedded
		want map[string]any
	}{
		{name: "nil embedded pointer is omitted", raw: declaredOptionalEmbedded{}, want: map[string]any{}},
		{name: "non-nil embedded pointer is promoted", raw: declaredOptionalEmbedded{
			DeclaredEmbedded: &DeclaredEmbedded{Name: "promoted"},
		}, want: map[string]any{"name": "promoted"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			embeddedOccurrence, occurrenceErr := context.NewOccurrence(&AttributeExpr{Type: embeddedObject})
			require.NoError(t, occurrenceErr)
			embeddedResult := context.Resolve(
				embeddedOccurrence,
				context.SupplyValue(ValueInput{Raw: test.raw}),
				ValueRoleEnum,
			)
			require.Equal(t, ValueResolved, embeddedResult.Outcome())
			declared, declaredOK := embeddedResult.DeclaredJSONValue()
			require.True(t, declaredOK)
			require.Equal(t, test.want, declared)
		})
	}
}

func ambiguousDeclaredStruct() any {
	typeOf := reflect.StructOf([]reflect.StructField{
		{Name: "First", Type: reflect.TypeFor[string](), Tag: `json:"value"`},
		{Name: "Second", Type: reflect.TypeFor[string](), Tag: `json:"value"`},
	})
	value := reflect.New(typeOf).Elem()
	value.Field(0).SetString("a")
	value.Field(1).SetString("b")
	return value.Interface()
}

func TestDeclaredValueRejectsWrongStructAndCyclesWithoutCallingCodecs(t *testing.T) {
	context := NewValueContext()
	wrong, err := context.NewOccurrence(&AttributeExpr{Type: String})
	require.NoError(t, err)
	result := context.Resolve(
		wrong,
		context.SupplyValue(ValueInput{Raw: struct{ Value string }{Value: "wrong"}}),
		ValueRoleEnum,
	)
	require.Equal(t, ValueInvalid, result.Outcome())
	_, ok := result.DeclaredJSONValue()
	require.False(t, ok)

	object := &Object{{Name: "next", Attribute: &AttributeExpr{Type: Any}}}
	occurrence, err := context.NewOccurrence(&AttributeExpr{Type: object})
	require.NoError(t, err)
	cycle := &declaredCycle{}
	cycle.Next = cycle
	result = context.Resolve(
		occurrence,
		context.SupplyValue(ValueInput{Raw: cycle}),
		ValueRoleEnum,
	)
	require.Equal(t, ValueInvalid, result.Outcome())
	_, ok = result.DeclaredJSONValue()
	require.False(t, ok)

	calls := 0
	codecOccurrence, err := context.NewOccurrence(&AttributeExpr{Type: Any})
	require.NoError(t, err)
	codec := declaredCodec{Calls: &calls}
	result = context.Resolve(
		codecOccurrence,
		context.SupplyValue(ValueInput{Raw: codec}),
		ValueRoleEnum,
	)
	require.Equal(t, ValueUnsupported, result.Outcome())
	declared, ok := result.DeclaredJSONValue()
	require.True(t, ok)
	require.Equal(t, codec, declared)
	require.Zero(t, calls)

	container := &Object{
		{Name: "blob", Attribute: &AttributeExpr{Type: Bytes}},
		{Name: "raw", Attribute: &AttributeExpr{Type: Any}},
	}
	containerOccurrence, err := context.NewOccurrence(&AttributeExpr{Type: container})
	require.NoError(t, err)
	rawJSON := jsontext.Value(`{"value":true}`)
	result = context.Resolve(
		containerOccurrence,
		context.SupplyValue(ValueInput{Raw: map[string]any{"blob": "hi", "raw": rawJSON}}),
		ValueRoleEnum,
	)
	require.Equal(t, ValueUnsupported, result.Outcome())
	declared, ok = result.DeclaredJSONValue()
	require.True(t, ok)
	require.Equal(t, map[string]any{"blob": []byte("hi"), "raw": rawJSON}, declared)
}

func TestResolveDeclaredShapeSuppressesOnlyRootPredicates(t *testing.T) {
	minimum := 2
	attribute := &AttributeExpr{
		Type: &Array{ElemType: &AttributeExpr{Type: Float32}},
		Validation: &ValidationExpr{
			MinLength:   &minimum,
			EnumClauses: [][]any{{[]float64{1.23456789}}},
		},
	}
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	source := context.SupplyValue(ValueInput{Raw: []float64{1.23456789}})
	require.Equal(t, ValueInvalid, context.Resolve(occurrence, source, ValueRoleEnum).Outcome())

	shape := context.ResolveDeclaredShape(occurrence, source)
	require.Equal(t, ValueResolved, shape.Outcome())
	declared, ok := shape.DeclaredJSONValue()
	require.True(t, ok)
	require.Equal(t, []any{float32(1.2345679)}, declared)

	foreign := NewValueContext().SupplyValue(ValueInput{Raw: []float64{1.23456789}})
	require.Equal(t, ValueInvalid, context.ResolveDeclaredShape(occurrence, foreign).Outcome())
}

func TestResolveDeclaredShapeRetainsNestedAndUnionConstraints(t *testing.T) {
	requiredChild := &Object{{Name: "value", Attribute: &AttributeExpr{Type: String}}}
	root := &Object{{Name: "child", Attribute: &AttributeExpr{
		Type:       requiredChild,
		Validation: &ValidationExpr{Required: []string{"value"}},
	}}}
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(&AttributeExpr{Type: root})
	require.NoError(t, err)
	source := context.SupplyValue(ValueInput{Raw: map[string]any{"child": map[string]any{}}})
	require.Equal(t, ValueInvalid, context.ResolveDeclaredShape(occurrence, source).Outcome())

	ambiguous := &Union{Untagged: true, Values: []*NamedAttributeExpr{
		{Name: "first", Attribute: &AttributeExpr{Type: String}},
		{Name: "second", Attribute: &AttributeExpr{Type: String}},
	}}
	occurrence, err = context.NewOccurrence(&AttributeExpr{Type: ambiguous})
	require.NoError(t, err)
	source = context.SupplyValue(ValueInput{Raw: "value"})
	require.Equal(t, ValueAmbiguous, context.ResolveDeclaredShape(occurrence, source).Outcome())

	selected := &Union{Untagged: true, Values: []*NamedAttributeExpr{
		{Name: "a", Attribute: &AttributeExpr{Type: String, Validation: &ValidationExpr{Pattern: "^a"}}},
		{Name: "b", Attribute: &AttributeExpr{Type: String, Validation: &ValidationExpr{Pattern: "^b"}}},
	}}
	occurrence, err = context.NewOccurrence(&AttributeExpr{Type: selected})
	require.NoError(t, err)
	source = context.SupplyValue(ValueInput{Raw: "apple"})
	require.Equal(t, ValueResolved, context.ResolveDeclaredShape(occurrence, source).Outcome())
}
