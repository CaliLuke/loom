package scripts_test

import (
	"encoding/json/v2"
	"math/big"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/jsonkey"
	loom "github.com/CaliLuke/loom/pkg"
)

// TestValueContractNumericRepresentationOwnership records the actual boundary
// that a decimal coefficient/exponent alone cannot describe. Mathematical enum
// equality, raw key spelling and target decode precision have different owners.
func TestValueContractNumericRepresentationOwnership(t *testing.T) {
	narrow := float32(0.1)
	wide := float64(narrow)
	narrowNumber := new(big.Rat).SetFloat64(float64(narrow))
	wideNumber := new(big.Rat).SetFloat64(wide)
	require.Equal(t, "13421773/134217728", narrowNumber.RatString())
	require.Zero(t, narrowNumber.Cmp(wideNumber))
	for _, tc := range []struct {
		name  string
		value any
		wire  string
	}{
		{name: "Float32", value: narrow, wire: "0.1"},
		{name: "Float64 with same exact value", value: wide, wire: "0.10000000149011612"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wire, err := json.Marshal(tc.value)
			require.NoError(t, err)
			require.Equal(t, tc.wire, string(wire))
			key, ok := jsonkey.Name(reflect.ValueOf(tc.value))
			require.True(t, ok)
			require.Equal(t, tc.wire, key)
		})
	}
	mapAttribute := &expr.AttributeExpr{Type: &expr.Map{
		KeyType:  &expr.AttributeExpr{Type: expr.Any},
		ElemType: &expr.AttributeExpr{Type: expr.String},
	}}
	input := map[any]string{narrow: "narrow", wide: "wide"}
	require.Equal(t, map[string]any{"0.1": "narrow", "0.10000000149011612": "wide"}, expr.CanonicalizeExample(mapAttribute, input))
	arrayWire, err := loom.JSONValueFrom([]any{narrow, wide})
	require.NoError(t, err)
	require.Equal(t, "[0.1,0.10000000149011612]", string(arrayWire))
	// Raw Any map serialization has a different contract from a declared map.
	_, err = loom.JSONValueFrom(input)
	require.Error(t, err)
	union := &expr.AttributeExpr{Type: &expr.Union{
		TypeKey: "kind", ValueKey: "payload",
		Values: []*expr.NamedAttributeExpr{{Name: "raw", Attribute: &expr.AttributeExpr{
			Type: expr.Any, Validation: &expr.ValidationExpr{Values: []any{narrow}},
		}}},
	}}
	require.Equal(t, map[string]any{"kind": "raw", "payload": wide}, expr.CanonicalizeExample(union, wide))
	var narrowTarget float32
	var wideTarget float64
	require.NoError(t, json.Unmarshal([]byte("0.1"), &narrowTarget))
	require.NoError(t, json.Unmarshal([]byte("0.1"), &wideTarget))
	require.NotZero(t, new(big.Rat).SetFloat64(float64(narrowTarget)).Cmp(new(big.Rat).SetFloat64(wideTarget)), "target precision must be selected by the occurrence")
}
