package openapiimport

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScalarLiteralRejectsNonfinite(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value any
	}{
		{"float32 positive infinity", float32(math.Inf(1))},
		{"float32 negative infinity", float32(math.Inf(-1))},
		{"float32 NaN", float32(math.NaN())},
		{"float64 positive infinity", math.Inf(1)},
		{"float64 negative infinity", math.Inf(-1)},
		{"float64 NaN", math.NaN()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, value := range []any{tc.value, []any{tc.value}, map[string]any{"value": tc.value}} {
				_, err := exampleLiteral(value)
				require.ErrorContains(t, err, "non-finite number")
			}
			_, err := scalarLiteral(tc.value)
			require.ErrorContains(t, err, "non-finite number")
		})
	}
}

func TestScalarLiteralRetainsFiniteValues(t *testing.T) {
	for _, tc := range []struct {
		name         string
		value        any
		want         string
		schemaScalar bool
	}{
		{"string NaN", "NaN", `"NaN"`, true},
		{"string infinity", ".inf", `".inf"`, true},
		{"false", false, "false", true},
		{"signed integer", int64(-23), "-23", true},
		{"unsigned integer", uint64(23), "23", false},
		{"float32 finite", float32(1.25), "1.25", false},
		{"float64 finite", 1.25, "1.25", true},
		{"float32 negative zero", float32(math.Copysign(0, -1)), "-0", false},
		{"float64 negative zero", math.Copysign(0, -1), "-0", true},
		{"float32 maximum", float32(math.MaxFloat32), "3.4028235e+38", false},
		{"float64 maximum", math.MaxFloat64, "1.7976931348623157e+308", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			literal, err := scalarLiteral(tc.value)
			if tc.schemaScalar {
				require.NoError(t, err)
				require.Equal(t, tc.want, literal)
			} else {
				require.ErrorContains(t, err, "is not a scalar Go literal")
			}
			example, err := exampleLiteral(tc.value)
			require.NoError(t, err)
			require.Equal(t, tc.want, example)
		})
	}
}
