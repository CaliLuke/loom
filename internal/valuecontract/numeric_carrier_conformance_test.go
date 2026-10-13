package valuecontract

import (
	"math"
	"testing"

	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/require"
)

// Extract the source carrier independently of expr's private numericDomain.
func productionSourceIntegerFormat(kind expr.Kind, machineBits int) any {
	bits := 0
	switch kind {
	case expr.IntKind, expr.UIntKind:
		bits = machineBits
	case expr.Int32Kind, expr.UInt32Kind:
		bits = 32
	case expr.Int64Kind, expr.UInt64Kind:
		bits = 64
	default:
		return "mathematical"
	}
	unsigned := kind == expr.UIntKind || kind == expr.UInt32Kind || kind == expr.UInt64Kind
	name := "signed"
	if unsigned {
		name = "unsigned"
	}
	return referenceConstructor(name, map[string]any{"bits": bits})
}

func productionNumericCarrierCases() []productionCase {
	cases := []struct {
		name string
		kind expr.DataType
		raw  any
	}{
		{"int32 above", expr.Int32, uint32(math.MaxInt32 + 1)},
		{"int32 maximum", expr.Int32, int32(math.MaxInt32)},
		{"int32 minimum", expr.Int32, int32(math.MinInt32)},
		{"uint32 negative", expr.UInt32, -1},
		{"uint32 maximum", expr.UInt32, uint32(math.MaxUint32)},
		{"int64 unsigned overflow", expr.Int64, uint64(math.MaxInt64 + 1)},
		{"int64 maximum", expr.Int64, int64(math.MaxInt64)},
		{"uint64 negative", expr.UInt64, int64(-1)},
		{"uint64 maximum", expr.UInt64, uint64(math.MaxUint64)},
		{"machine uint negative", expr.UInt, -1},
		{"machine int unsigned overflow", expr.Int, ^uint(0)},
		{"wrong source int64", expr.Int32, int64(1)},
		{"float32 overflow", expr.Float32, 1e39},
		{"float32 rounding", expr.Float32, 1.23456789},
		{"float32 underflow", expr.Float32, 1e-100},
	}
	result := make([]productionCase, 0, len(cases))
	for _, tc := range cases {
		result = append(result, productionCase{"carrier/" + tc.name, &expr.AttributeExpr{Type: tc.kind}, tc.raw})
	}
	return result
}

func TestSourceIntegerFormatExtraction(t *testing.T) {
	for _, bits := range []int{32, 64} {
		require.Equal(t, referenceConstructor("signed", map[string]any{"bits": bits}), productionSourceIntegerFormat(expr.IntKind, bits))
		require.Equal(t, referenceConstructor("unsigned", map[string]any{"bits": bits}), productionSourceIntegerFormat(expr.UIntKind, bits))
		require.Equal(t, referenceConstructor("signed", map[string]any{"bits": 32}), productionSourceIntegerFormat(expr.Int32Kind, bits))
		require.Equal(t, "mathematical", productionSourceIntegerFormat(expr.Float32Kind, bits))
	}
}
