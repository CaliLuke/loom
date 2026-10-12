package expr

import (
	"fmt"
	"strconv"

	"github.com/CaliLuke/loom/internal/exactnum"
)

// primitiveValue admits the concrete source type before normalizing to the
// declared numeric carrier. Compatibility and resolved values share this owner.
// Any has no declared carrier and keeps its separate raw-value contract.
func primitiveValue(primitive Primitive, raw any) (any, bool) {
	if !primitiveSourceCompatible(primitive, raw) {
		return nil, false
	}
	switch primitive {
	case Int, Int32, Int64, UInt, UInt32, UInt64:
		number, finite := exactnum.FromScalar(raw)
		minimum, maximum := numericDomain(primitive.Kind()).Limits()
		lower, lowerOK := exactnum.FromScalar(minimum)
		upper, upperOK := exactnum.FromScalar(maximum)
		if !finite || !lowerOK || !upperOK || number.Cmp(lower) < 0 || number.Cmp(upper) > 0 {
			return nil, false
		}
	case Float32, Float64:
		if _, finite := exactnum.FromScalar(raw); !finite {
			return nil, false
		}
		bits := numericDomain(primitive.Kind()).Bits
		// Preserve authored-literal normalization, including rounding and
		// underflow. Exact float bound semantics are a separate contract.
		number, err := strconv.ParseFloat(fmt.Sprint(raw), bits)
		if err != nil {
			return nil, false
		}
		if bits == 32 {
			return float32(number), true
		}
		return number, true
	case Bytes:
		if text, ok := raw.(string); ok {
			return []byte(text), true
		}
		bytes := raw.([]byte)
		normalized := make([]byte, len(bytes))
		copy(normalized, bytes)
		return normalized, true
	}
	return raw, true
}

// primitiveSourceCompatible preserves concrete Go source identity. In
// particular reflection must not admit named scalars or int64 into Int32.
func primitiveSourceCompatible(primitive Primitive, raw any) bool {
	if primitive == Any {
		return true
	}
	switch raw.(type) {
	case bool:
		return primitive == Boolean
	case int, int8, int16, int32, uint, uint8, uint16, uint32:
		return primitive == Int || primitive == Int32 || primitive == Int64 ||
			primitive == UInt || primitive == UInt32 || primitive == UInt64 ||
			primitive == Float32 || primitive == Float64
	case int64, uint64:
		return primitive == Int64 || primitive == UInt64 || primitive == Float32 || primitive == Float64
	case float32, float64:
		return primitive == Float32 || primitive == Float64
	case string:
		return primitive == String || primitive == Bytes
	case []byte:
		return primitive == Bytes
	}
	return false
}
