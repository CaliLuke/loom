package expr

import (
	"fmt"
	"strconv"

	"github.com/CaliLuke/loom/internal/exactnum"
)

// NumericBound is an immutable exact finite endpoint. Its arithmetic owner is
// shared with runtime schema matching; its zero value is the number zero.
type NumericBound = exactnum.Number

// NumericBoundProjection is a closed native comparison threshold derived from
// an exact endpoint. Empty reports that no finite native value satisfies it.
type NumericBoundProjection struct {
	// Value is an int64, uint64, float32 or float64 closed threshold.
	Value any
	// Empty reports an empty accepted native domain.
	Empty bool
}

// NewNumericBound constructs an exact endpoint from the DSL's admitted builtin
// numeric types or finite numeric string syntax. Integers and strings retain
// exact meaning; floats retain their actual binary value. Builtin uint, named
// source types, pointers and nonfinite values are not admitted.
func NewNumericBound(value any) (NumericBound, error) {
	switch actual := value.(type) {
	case string:
		return exactnum.ParseDSL(actual)
	case float32, float64, int, int8, int16, int32, int64, uint8, uint16, uint32, uint64:
		if number, ok := exactnum.FromScalar(actual); ok {
			return number, nil
		}
	}
	return NumericBound{}, fmt.Errorf("invalid finite numeric bound %#v", value)
}

// ProjectNumericBound lowers an exact endpoint into the declared primitive
// carrier. Generated validators and example synthesis share this projection.
// Integer kinds must have whole-number authored bounds, validated separately.
func ProjectNumericBound(kind Kind, bound NumericBound, lower, exclusive bool) NumericBoundProjection {
	value, ok := bound.Boundary(numericDomain(kind), lower, exclusive)
	return NumericBoundProjection{Value: value, Empty: !ok}
}

func numericDomain(kind Kind) exactnum.Domain {
	switch kind {
	case IntKind:
		return exactnum.Domain{Bits: strconv.IntSize}
	case UIntKind:
		return exactnum.Domain{Bits: strconv.IntSize, Unsigned: true}
	case Int32Kind:
		return exactnum.Domain{Bits: 32}
	case Int64Kind:
		return exactnum.Domain{Bits: 64}
	case UInt32Kind:
		return exactnum.Domain{Bits: 32, Unsigned: true}
	case UInt64Kind:
		return exactnum.Domain{Bits: 64, Unsigned: true}
	case Float32Kind:
		return exactnum.Domain{Bits: 32, Float: true}
	case Float64Kind:
		return exactnum.Domain{Bits: 64, Float: true}
	default:
		panic("numeric projection requires a numeric primitive kind")
	}
}
