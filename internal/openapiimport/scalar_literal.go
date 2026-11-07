package openapiimport

import (
	"fmt"
	"math"
	"strconv"
)

// scalarGoLiteral reports whether value is a supported scalar, separately from
// whether its value can be represented. In particular, non-finite floats are
// scalars but cannot be JSON numbers or Go numeric literals. Callers retain
// their own type eligibility and diagnostic ownership.
func scalarGoLiteral(value any) (literal string, scalar bool, err error) {
	var number float64
	bits := 64
	switch actual := value.(type) {
	case string:
		return strconv.Quote(actual), true, nil
	case bool:
		return strconv.FormatBool(actual), true, nil
	case int:
		return strconv.Itoa(actual), true, nil
	case int8:
		return strconv.FormatInt(int64(actual), 10), true, nil
	case int16:
		return strconv.FormatInt(int64(actual), 10), true, nil
	case int32:
		return strconv.FormatInt(int64(actual), 10), true, nil
	case int64:
		return strconv.FormatInt(actual, 10), true, nil
	case uint:
		return strconv.FormatUint(uint64(actual), 10), true, nil
	case uint8:
		return strconv.FormatUint(uint64(actual), 10), true, nil
	case uint16:
		return strconv.FormatUint(uint64(actual), 10), true, nil
	case uint32:
		return strconv.FormatUint(uint64(actual), 10), true, nil
	case uint64:
		return strconv.FormatUint(actual, 10), true, nil
	case float32:
		number, bits = float64(actual), 32
	case float64:
		number = actual
	default:
		return "", false, nil
	}
	if math.IsNaN(number) || math.IsInf(number, 0) {
		return "", true, fmt.Errorf("non-finite number %v is not a scalar Go literal", number)
	}
	return strconv.FormatFloat(number, 'g', -1, bits), true, nil
}

// scalarLiteral preserves the schema default/enum renderer's scalar carriers.
// Example literals admit additional integer widths and float32, so they consume
// scalarGoLiteral directly. Both paths share representability and conversion.
func scalarLiteral(value any) (string, error) {
	literal, _, err := scalarGoLiteral(value)
	if err != nil {
		return "", err
	}
	switch value.(type) {
	case string, bool, int, int64, float64:
		return literal, nil
	case nil:
		return "nil", nil
	default:
		return "", fmt.Errorf("value of type %T is not a scalar Go literal", value)
	}
}
