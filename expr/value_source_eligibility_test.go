package expr

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

type (
	eligibilityInt       int
	eligibilityFloat     float64
	eligibilityString    string
	eligibilityBool      bool
	eligibilityBytes     []byte
	eligibilityByte      uint8
	eligibilityByteArray [1]byte
	eligibilityFormatted float64
)

func (eligibilityFormatted) String() string {
	return "17"
}

// Admission belongs to the existing source matcher. Reflection and literal
// normalization must not widen the concrete Go types it accepts.
func TestValueResolvePrimitiveEligibility(t *testing.T) {
	values := []any{
		true, int(1), int8(1), int16(1), int32(1), int64(1),
		uint(1), uint8(1), uint16(1), uint32(1), uint64(1),
		float32(1), float64(1), "one", []byte{1},
		eligibilityInt(1), eligibilityFloat(1), eligibilityString("one"),
		eligibilityBool(true), eligibilityFormatted(1), eligibilityBytes{1},
		[]eligibilityByte{1}, [1]byte{1}, eligibilityByteArray{1},
	}
	for _, typ := range []Primitive{Boolean, Int, Int32, Int64, UInt, UInt32, UInt64, Float32, Float64, String, Bytes, Any} {
		for _, raw := range values {
			t.Run(fmt.Sprintf("%s/%T", typ.Name(), raw), func(t *testing.T) {
				checkValueSourceEligibility(t, &AttributeExpr{Type: typ}, raw)
			})
		}
	}
}

func TestValueResolveSequenceEligibility(t *testing.T) {
	values := []any{
		[]byte{1}, []byte{}, [1]byte{1}, [0]byte{},
		eligibilityBytes{1}, eligibilityBytes{}, eligibilityByteArray{1},
		[]eligibilityByte{1}, []eligibilityByte{}, [1]eligibilityByte{1}, [0]eligibilityByte{},
	}
	for _, element := range []Primitive{UInt, Float64, Any} {
		for _, raw := range values {
			t.Run(fmt.Sprintf("%s/%T/%v", element.Name(), raw, raw), func(t *testing.T) {
				checkValueSourceEligibility(t, &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: element}}}, raw)
			})
		}
	}
}

func checkValueSourceEligibility(t *testing.T, attribute *AttributeExpr, raw any) {
	t.Helper()
	accepted := exampleMatchesAttribute(attribute, raw)
	choice := &Union{Values: []*NamedAttributeExpr{{Name: "Candidate", Attribute: attribute}}}
	canonical := CanonicalizeExample(&AttributeExpr{Type: choice}, raw)
	_, selected := canonical.(map[string]any)
	require.Equal(t, accepted, selected, "public legacy branch selection")
	for _, role := range []ValueRole{ValueRoleExample, ValueRoleEnum, ValueRoleDefault} {
		context := NewValueContext()
		occurrence, err := context.NewOccurrence(attribute)
		require.NoError(t, err)
		var result ValueResult
		require.NotPanics(t, func() {
			result = context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: raw}), role)
		})
		want := ValueInvalid
		if accepted {
			want = ValueResolved
		}
		require.Equal(t, want, result.Outcome(), "role %d, legacy admission %t", role, accepted)
		legacy, available := result.LegacyValue()
		require.True(t, available)
		require.Equal(t, raw, legacy)
	}
}
