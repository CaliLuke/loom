package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidationDomainAdmission(t *testing.T) {
	negative, zero, one := -1, 0, 1
	bound, err := NewNumericBound("9007199254740993.125")
	require.NoError(t, err)
	for _, tc := range []struct {
		name   string
		rules  ValidationExpr
		reject bool
	}{
		{"exclusive equality", ValidationExpr{ExclusiveMinimum: &bound, ExclusiveMaximum: &bound}, true},
		{"negative minimum length", ValidationExpr{MinLength: &negative}, true},
		{"negative maximum length", ValidationExpr{MaxLength: &negative}, true},
		{"inclusive equality", ValidationExpr{Minimum: &bound, Maximum: &bound}, false},
		{"mixed lower equality", ValidationExpr{Minimum: &bound, ExclusiveMaximum: &bound}, true},
		{"mixed upper equality", ValidationExpr{ExclusiveMinimum: &bound, Maximum: &bound}, true},
		{"zero lengths", ValidationExpr{MinLength: &zero, MaxLength: &zero}, false},
		{"proper lengths", ValidationExpr{MinLength: &zero, MaxLength: &one}, false},
		{"reversed lengths", ValidationExpr{MinLength: &one, MaxLength: &zero}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			errs := tc.rules.Validate("domain - ", nil)
			require.Equal(t, tc.reject, len(errs.Errors) > 0, "%v", errs.Errors)
		})
	}
}

func TestEffectiveConstraintDomainAdmission(t *testing.T) {
	negative, zero, two := -1, 0, 2
	low, err := NewNumericBound("9007199254740993.125")
	require.NoError(t, err)
	high, err := NewNumericBound("9007199254740993.126")
	require.NoError(t, err)
	for _, tc := range []struct {
		name        string
		kind        DataType
		base, local *ValidationExpr
		reject      bool
	}{
		{"inherited exclusive equality", Float64, &ValidationExpr{ExclusiveMinimum: &low}, &ValidationExpr{ExclusiveMaximum: &low}, true},
		{"inherited mixed equality", Float64, &ValidationExpr{Minimum: &low}, &ValidationExpr{ExclusiveMaximum: &low}, true},
		{"inherited reversed interval", Float64, &ValidationExpr{Minimum: &high}, &ValidationExpr{Maximum: &low}, true},
		{"inherited reversed lengths", String, &ValidationExpr{MinLength: &two}, &ValidationExpr{MaxLength: &zero}, true},
		{"masked negative length", String, &ValidationExpr{MinLength: &negative}, &ValidationExpr{MinLength: &zero}, true},
		{"negative maximum length", String, &ValidationExpr{MaxLength: &negative}, nil, true},
		{"exact narrow proper interval", Float64, &ValidationExpr{ExclusiveMinimum: &low}, &ValidationExpr{ExclusiveMaximum: &high}, false},
		{"equal inclusive interval", Float64, &ValidationExpr{Minimum: &low}, &ValidationExpr{Maximum: &low}, false},
		{"zero lengths", String, &ValidationExpr{MinLength: &zero}, &ValidationExpr{MaxLength: &zero}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base := namedScalar("DomainBase", tc.kind, tc.base)
			before := cloneEffectiveValidation(base.Attribute().Validation)
			attribute := &AttributeExpr{Type: base, Validation: tc.local}
			_, err := EffectiveConstraintsFor(attribute)
			if tc.reject {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, before, base.Attribute().Validation, "authored base must remain unchanged")
		})
	}
}
