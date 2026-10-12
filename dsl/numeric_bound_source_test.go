package dsl_test

import (
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/require"
)

// This compatibility control records source eligibility independently of exact
// extraction. It deliberately does not widen admission using reflect.Kind.
func TestNumericBoundSourceAdmission(t *testing.T) {
	type namedInteger int64
	type namedFloat float64
	type namedText string
	for _, test := range []struct {
		name     string
		value    any
		accepted bool
	}{
		{"int", int(1), true}, {"int8", int8(1), true},
		{"int16", int16(1), true}, {"int32", int32(1), true},
		{"int64", int64(1), true}, {"uint", uint(1), false},
		{"uint8", uint8(1), true}, {"uint16", uint16(1), true},
		{"uint32", uint32(1), true}, {"uint64", uint64(1), true},
		{"float32", float32(0.1), true}, {"float64", float64(0.1), true},
		{"named integer", namedInteger(1), false},
		{"named float", namedFloat(1), false},
		{"named text", namedText("1"), false},
		{"pointer", new(int64(1)), false}, {"nil", nil, false},
		{"decimal", "1.25", true}, {"exponent", "125e-2", true},
		{"hexadecimal", "0x1.4p0", true}, {"underscore", "1_000", true},
		{"leading dot", ".5", true}, {"trailing dot", "1.", true},
		{"positive sign", "+1", true}, {"negative zero", "-0", true},
		{"ratio", "1/2", false}, {"whitespace", " 1 ", false},
		{"malformed", "1e", false}, {"overflow", "1e309", false},
		{"underflow", "1e-400", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			eval.SetupTestContext(t)
			attribute := &expr.AttributeExpr{Type: Float64}
			require.Equal(t, test.accepted, eval.Execute(func() {
				Minimum(test.value)
			}, attribute))
		})
	}
}
