package expr

import (
	"encoding/json/jsontext"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdditionalDecodeWork(t *testing.T) {
	measure := func(depth int) float64 {
		plan, wire := nestedAdditionalDecodeInput(depth)
		valid := false
		// JSON decoders use sync.Pool, whose race-enabled Put randomly drops
		// entries. Average enough calls that one pool miss cannot dominate
		// the growth comparison after its single warm-up call.
		allocations := testing.AllocsPerRun(100, func() {
			_, valid = decodeJSON(plan, wire)
		})
		require.True(t, valid)
		return allocations
	}
	// Doubling the nesting may increase JSON parsing work, but must not turn
	// one value into two recursive decodes at every level. Compare allocation
	// growth, not machine-dependent wall time or an exact compiler allocation count.
	shallow, deep := measure(4), measure(8)
	t.Logf("allocations per decode: depth4=%.0f depth8=%.0f growth=%.3f", shallow, deep, deep/shallow)
	require.Less(t, deep, 4*shallow, "additional-value decoding must not branch exponentially")
}

func TestAdditionalDecodeRetainsChildren(t *testing.T) {
	plan, wire := nestedAdditionalDecodeInput(8)
	value, valid := decodeJSON(plan, wire)
	require.True(t, valid)
	for range 8 {
		fields := value.Fields()
		require.Len(t, fields, 1)
		require.Equal(t, "x", fields[0].Name)
		value = fields[0].Value
	}
	scalar, present := value.Scalar()
	require.True(t, present)
	require.Equal(t, 1, scalar)
	_, valid = decodeJSON(plan, jsontext.Value(`{"x":{"x":{"x":{"x":{"x":{"x":{"x":{"x":"bad"}}}}}}}}`))
	require.False(t, valid)
}

func nestedAdditionalDecodeInput(depth int) (*valuePlanNode, jsontext.Value) {
	node := &valuePlanNode{kind: IntKind, attribute: &AttributeExpr{Type: Int}}
	wire := jsontext.Value("1")
	for range depth {
		node = &valuePlanNode{
			kind: ObjectKind, attribute: &AttributeExpr{Type: &Object{}},
			additional: node, runtimeUnknown: true, preserveAdditional: true,
		}
		wire = append(append([]byte(`{"x":`), wire...), '}')
	}
	return node, wire
}
