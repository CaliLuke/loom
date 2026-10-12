package expr

import (
	"fmt"
	"math"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

type admissionNamedInt int

func TestNumericValueCarrierAdmission(t *testing.T) {
	wide, below := int64(1<<40), int64(-2147483649)
	for _, tc := range []struct {
		name string
		kind DataType
		raw  any
		want ValueOutcome
	}{
		{"int32 overflow", Int32, uint32(1 << 31), ValueInvalid},
		{"int32 underflow", Int32, int(below), ValueInvalid},
		{"uint negative", UInt, -5, ValueInvalid},
		{"uint32 overflow", UInt32, uint(wide), ValueInvalid},
		{"uint32 negative", UInt32, -1, ValueInvalid},
		{"int64 unsigned overflow", Int64, uint64(math.MaxUint64), ValueInvalid},
		{"uint64 signed negative", UInt64, int64(-1), ValueInvalid},
		{"int32 maximum", Int32, math.MaxInt32, ValueResolved},
		{"int32 minimum", Int32, math.MinInt32, ValueResolved},
		{"uint32 maximum", UInt32, uint32(math.MaxUint32), ValueResolved},
		{"int64 maximum", Int64, int64(math.MaxInt64), ValueResolved},
		{"uint64 maximum", UInt64, uint64(math.MaxUint64), ValueResolved},
		{"uint zero", UInt, 0, ValueResolved},
		{"float32 overflow", Float32, 1e39, ValueInvalid},
		{"float32 maximum", Float32, float32(math.MaxFloat32), ValueResolved},
		{"float32 rounds", Float32, 1.23456789, ValueResolved},
		{"float32 rounding edge above maximum", Float32, math.Nextafter(float64(math.MaxFloat32), math.Inf(1)), ValueResolved},
		{"float64 maximum", Float64, math.MaxFloat64, ValueResolved},
		{"float32 underflows", Float32, 1e-100, ValueResolved},
		{"float nonfinite", Float64, math.Inf(1), ValueInvalid},
		{"named source still rejected", Int, admissionNamedInt(1), ValueInvalid},
		{"int64 source cannot enter int32", Int32, int64(1), ValueInvalid},
		{"any has no integer carrier", Any, uint64(math.MaxUint64), ValueResolved},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if strconv.IntSize == 32 && (tc.name == "int32 underflow" || tc.name == "uint32 overflow") {
				t.Skip("source literal requires a 64-bit machine integer; fixed-width and host-width controls are separate")
			}
			t.Run("primitive compatibility", func(t *testing.T) {
				require.Equal(t, tc.want == ValueResolved, tc.kind.IsCompatible(tc.raw))
			})
			context := NewValueContext()
			occurrence, err := context.NewOccurrence(&AttributeExpr{Type: tc.kind})
			require.NoError(t, err)
			for _, role := range []ValueRole{ValueRoleDefault, ValueRoleEnum, ValueRoleExample} {
				t.Run(fmt.Sprint(role), func(t *testing.T) {
					result := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: tc.raw}), role)
					require.Equal(t, tc.want, result.Outcome(), "%v", result.Diagnostics())
				})
			}
		})
	}
}

func TestNumericValueCarrierNestedAdmission(t *testing.T) {
	alias := &UserTypeExpr{TypeName: "Narrow", AttributeExpr: &AttributeExpr{Type: Int32}}
	for _, tc := range []struct {
		name string
		kind DataType
		raw  any
	}{
		{"alias", alias, uint32(1 << 31)},
		{"array", &Array{ElemType: &AttributeExpr{Type: alias}}, []uint32{1 << 31}},
		{"object", &Object{{Name: "n", Attribute: &AttributeExpr{Type: alias}}}, map[string]any{"n": uint32(1 << 31)}},
		{"map value", &Map{KeyType: &AttributeExpr{Type: String}, ElemType: &AttributeExpr{Type: alias}}, map[string]uint32{"n": 1 << 31}},
		{"map key", &Map{KeyType: &AttributeExpr{Type: alias}, ElemType: &AttributeExpr{Type: String}}, map[uint32]string{1 << 31: "n"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			context := NewValueContext()
			occurrence, err := context.NewOccurrence(&AttributeExpr{Type: tc.kind})
			require.NoError(t, err)
			for _, role := range []ValueRole{ValueRoleDefault, ValueRoleEnum, ValueRoleExample} {
				t.Run(fmt.Sprint(role), func(t *testing.T) {
					result := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: tc.raw}), role)
					require.Equal(t, ValueInvalid, result.Outcome(), "%v", result.Diagnostics())
				})
			}
		})
	}
}

func TestNumericValueCarrierOpaqueBoundary(t *testing.T) {
	for _, test := range []struct {
		name        string
		number      uint32
		wantFailure bool
	}{
		{"known invalid sibling", 1 << 31, true},
		{"known valid sibling", 1, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls atomic.Int32
			attribute := &AttributeExpr{Type: &Object{
				{Name: "opaque", Attribute: &AttributeExpr{Type: Any}},
				{Name: "number", Attribute: &AttributeExpr{Type: Int32}},
			}}
			context := NewValueContext()
			occurrence, err := context.NewOccurrence(attribute)
			require.NoError(t, err)
			source := context.SupplyValue(ValueInput{Raw: map[string]any{
				"opaque": valuePlanOpaqueEnumCodec{calls: &calls},
				"number": test.number,
			}})
			result := context.ResolveDeclaredShape(occurrence, source)
			require.Equal(t, ValueUnsupported, result.Outcome())
			require.Equal(t, test.wantFailure, result.checkableFailure)
			require.Zero(t, calls.Load())
		})
	}
}

func TestNumericValueDeferredDefaultAdmission(t *testing.T) {
	attribute := &AttributeExpr{}
	attribute.SetDefault(uint32(1 << 31))
	attribute.Type = Int32
	_, err := NewValueContext().NewOccurrence(attribute)
	require.ErrorContains(t, err, "default value")
}
