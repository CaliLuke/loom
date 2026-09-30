package enumvalue

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"math"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/examplevalue"
)

type opaqueEnumCodec struct {
	Text  string
	Calls *int
}

func (value opaqueEnumCodec) MarshalText() ([]byte, error) {
	*value.Calls++
	return []byte(value.Text), nil
}

type plainEnumObject struct {
	Score float64 `json:"rating"`
	Blob  string  `json:"b"`
	Extra bool    `json:"extra"`
}

// EnumEmbedded supplies exported anonymous fields to the shared object resolver.
type EnumEmbedded struct {
	Score float64 `json:"rating"`
}

type embeddedEnumObject struct {
	*EnumEmbedded
	Blob string `json:"b"`
}

type cyclicEnumObject struct {
	Name string `json:"name"`
	Next *cyclicEnumObject
}

func TestNormalizePreservesAnyValuesAndWireNamePrecedence(t *testing.T) {
	object := &expr.AttributeExpr{Type: &expr.Object{
		{Name: "data:wire", Attribute: &expr.AttributeExpr{Type: expr.Any}},
		{Name: "raw", Attribute: &expr.AttributeExpr{Type: expr.Any}},
	}}
	for _, tc := range []struct {
		name      string
		attribute *expr.AttributeExpr
		value     any
		expected  string
	}{
		{"object", object, map[string]any{"data:wire": "discarded", "wire": []byte(nil), "raw": jsontext.Value(`9007199254740993`), "extra": true}, `{"extra":true,"raw":9007199254740993,"wire":""}`},
		{"map", &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.UInt64}, ElemType: &expr.AttributeExpr{Type: expr.Any}}}, map[uint64]any{9007199254740993: []byte(nil), 2: jsontext.Value(`{"exact":18446744073709551615}`)}, `{"2":{"exact":18446744073709551615},"9007199254740993":""}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before, err := json.Marshal(tc.value, json.Deterministic(true))
			require.NoError(t, err)
			for range 32 {
				projected := Normalize(tc.attribute, tc.value)
				encoded, err := json.Marshal(projected, json.Deterministic(true))
				require.NoError(t, err)
				require.Equal(t, tc.expected, string(encoded))
			}
			after, err := json.Marshal(tc.value, json.Deterministic(true))
			require.NoError(t, err)
			require.Equal(t, before, after, "projection mutated its authored value")
		})
	}
}

func TestNormalizeUsesSharedDeclaredValueSemantics(t *testing.T) {
	object := &expr.AttributeExpr{Type: &expr.Object{
		{Name: "blob:b", Attribute: &expr.AttributeExpr{Type: expr.Bytes}},
		{Name: "count", Attribute: &expr.AttributeExpr{Type: expr.Float32}},
		{Name: "raw", Attribute: &expr.AttributeExpr{Type: expr.Any}},
	}}
	tagged := &expr.Union{Values: []*expr.NamedAttributeExpr{
		{Name: "Blob", Attribute: &expr.AttributeExpr{Type: expr.Bytes}},
		{Name: "Count", Attribute: &expr.AttributeExpr{Type: expr.Float32}},
	}}
	untagged := *tagged
	untagged.Untagged = true
	for _, tc := range []struct {
		name      string
		attribute *expr.AttributeExpr
		value     any
		want      any
	}{
		{"bytes text", &expr.AttributeExpr{Type: expr.Bytes}, "hi", []byte("hi")},
		{"float32 precision", &expr.AttributeExpr{Type: expr.Float32}, float64(0.1), float32(0.1)},
		{"array nil becomes empty", &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}}, []string(nil), []any{}},
		{"map nil becomes empty", &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.UInt64}, ElemType: &expr.AttributeExpr{Type: expr.String}}}, map[uint64]string(nil), map[string]any{}},
		{"object wire names children and extras", object, map[string]any{"blob:b": "hi", "count": 0.1, "raw": jsontext.Value(`9007199254740993`), "extra": true}, map[string]any{"b": []byte("hi"), "count": float32(0.1), "raw": jsontext.Value(`9007199254740993`), "extra": true}},
		{"tagged union branch", &expr.AttributeExpr{Type: tagged}, examplevalue.Union{Branch: 0, Value: "hi"}, map[string]any{"type": "Blob", "value": []byte("hi")}},
		{"untagged union branch", &expr.AttributeExpr{Type: &untagged}, examplevalue.Union{Branch: 1, Value: 0.1}, float32(0.1)},
		{"any exact host value", &expr.AttributeExpr{Type: expr.Any}, map[string]any{"number": uint64(math.MaxUint64), "bytes": []byte(nil)}, map[string]any{"number": uint64(math.MaxUint64), "bytes": []byte(nil)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, Normalize(tc.attribute, tc.value))
		})
	}
}

func TestNormalizeUsesDeclaredShapeWhenRootPredicatesExcludeCarrier(t *testing.T) {
	minimum := 2
	raw := []float64{1.23456789}
	attribute := &expr.AttributeExpr{
		Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.Float32}},
		Validation: &expr.ValidationExpr{
			EnumClauses: [][]any{{[]float64{1.23456789}}},
			MinLength:   &minimum,
		},
	}

	require.Equal(t, []any{float32(1.23456789)}, Normalize(attribute, raw))
	require.Equal(t, []float64{1.23456789}, raw, "normalization must not mutate the authored carrier")
}

func TestNormalizeUsesSharedPlainStructObjectSemantics(t *testing.T) {
	attribute := &expr.AttributeExpr{Type: &expr.Object{
		{Name: "score:rating", Attribute: &expr.AttributeExpr{Type: expr.Float32}},
		{Name: "blob:b", Attribute: &expr.AttributeExpr{Type: expr.Bytes}},
	}}
	raw := &plainEnumObject{Score: 0.1, Blob: "hi", Extra: true}
	projected := Normalize(attribute, raw)
	require.Equal(t, map[string]any{
		"rating": float32(0.1),
		"b":      []byte("hi"),
		"extra":  true,
	}, projected)
	raw.Score, raw.Blob, raw.Extra = 0.2, "changed", false
	require.Equal(t, map[string]any{
		"rating": float32(0.1),
		"b":      []byte("hi"),
		"extra":  true,
	}, projected, "declared projection must own interpreted struct data")

	embedded := &embeddedEnumObject{EnumEmbedded: &EnumEmbedded{Score: 0.1}, Blob: "hi"}
	require.Equal(t, map[string]any{"rating": float32(0.1), "b": []byte("hi")}, Normalize(attribute, embedded))
	embedded.EnumEmbedded = nil
	require.Equal(t, map[string]any{"b": []byte("hi")}, Normalize(attribute, embedded))
}

func TestNormalizeKeepsCodecAndCycleFallbacksBorrowed(t *testing.T) {
	attribute := &expr.AttributeExpr{Type: &expr.Object{
		{Name: "name", Attribute: &expr.AttributeExpr{Type: expr.String}},
	}}
	calls := 0
	codec := &opaqueEnumCodec{Text: "opaque", Calls: &calls}
	require.Same(t, codec, Normalize(attribute, codec))
	require.Zero(t, calls)

	cycle := &cyclicEnumObject{Name: "root"}
	cycle.Next = cycle
	require.Same(t, cycle, Normalize(attribute, cycle), "unsupported pointer fallback stays borrowed")
}

func TestNormalizeKeepsOpaqueAndInvalidRawBoundaries(t *testing.T) {
	attribute := &expr.AttributeExpr{Type: &expr.Object{
		{Name: "blob:b", Attribute: &expr.AttributeExpr{Type: expr.Bytes}},
	}}
	calls := 0
	opaque := opaqueEnumCodec{Text: "hi", Calls: &calls}
	pointer := &opaqueEnumCodec{Text: "hi", Calls: &calls}

	require.Equal(t, opaque, Normalize(attribute, opaque))
	require.Same(t, pointer, Normalize(attribute, pointer))
	require.Zero(t, calls)

	invalidWithOpaque := map[string]any{
		"blob:b": 42,
		"extra":  jsontext.Value(`{"exact":18446744073709551615}`),
	}
	fallback := Normalize(attribute, invalidWithOpaque)
	require.Equal(t, map[string]any{
		"b":     42,
		"extra": jsontext.Value(`{"exact":18446744073709551615}`),
	}, fallback)
	require.NotEqual(t, reflect.ValueOf(invalidWithOpaque).Pointer(), reflect.ValueOf(fallback).Pointer(),
		"the invalid semantic fallback still uses the shared structural projection")
}
