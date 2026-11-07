package ir

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestEmptyValidationHasNoSchemaIdentity(t *testing.T) {
	plain := &expr.AttributeExpr{Type: expr.String}
	constraints, err := expr.EffectiveConstraintsFor(plain)
	require.NoError(t, err)
	for _, tc := range []struct {
		name  string
		rules *expr.ValidationExpr
	}{
		{"allocated", &expr.ValidationExpr{}},
		{"lowered", constraints.Validation().Lowered()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			selected := &expr.AttributeExpr{Type: expr.String, Validation: tc.rules}
			assert.Equal(t, fingerprintAttribute(plain, false), fingerprintAttribute(selected, false))
			analyzer := NewAnalyzer(nil, false)
			assert.Equal(t, analyzer.FingerprintAttribute(plain), analyzer.FingerprintAttribute(selected))
			require.NoError(t, analyzer.Err())
			named := &expr.UserTypeExpr{TypeName: "Value", AttributeExpr: plain}
			occurrence := &expr.AttributeExpr{Type: named, Validation: tc.rules}
			assert.False(t, hasUserTypeConstraintOverlay(occurrence, named), "storage for no predicates must not create a reference overlay")
		})
	}
}

func TestNonemptyValidationRetainsSchemaIdentity(t *testing.T) {
	zero := 0
	for _, tc := range []struct {
		name  string
		rules *expr.ValidationExpr
	}{
		{"zero length", &expr.ValidationExpr{MinLength: &zero}},
		{"pattern", &expr.ValidationExpr{Pattern: "x"}},
		{"enum", &expr.ValidationExpr{Values: []any{"x"}}},
		{"empty enum domain", &expr.ValidationExpr{EnumClauses: [][]any{{}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plain := &expr.AttributeExpr{Type: expr.String}
			selected := &expr.AttributeExpr{Type: expr.String, Validation: tc.rules}
			assert.NotEqual(t, fingerprintAttribute(plain, false), fingerprintAttribute(selected, false))
			named := &expr.UserTypeExpr{TypeName: "Value", AttributeExpr: plain}
			assert.True(t, hasUserTypeConstraintOverlay(&expr.AttributeExpr{Type: named, Validation: tc.rules}, named))
		})
	}
}

func TestCheckedValidationKeepsInheritedEnumWithoutLocalRules(t *testing.T) {
	named := &expr.UserTypeExpr{
		TypeName: "Choice",
		AttributeExpr: &expr.AttributeExpr{
			Type:       expr.String,
			Validation: &expr.ValidationExpr{Values: []any{"x"}},
		},
	}
	attribute := &expr.AttributeExpr{Type: named}
	encoder := newCanonicalSchemaEncoder()
	encoder.checked = &schemaFingerprintAnalysis{context: expr.NewValueContext()}
	input := encoder.captureInput(attribute)
	encoder.writeValidation(attribute, input)
	require.NoError(t, encoder.checked.err)
	plain := newCanonicalSchemaEncoder()
	plain.writeValidation(&expr.AttributeExpr{Type: expr.String}, schemaFingerprintInput{})
	require.NotEqual(t, plain.Bytes(), encoder.Bytes(), "checked inherited enums survive nil local validation")
	declared := newCanonicalSchemaEncoder()
	declared.writeValidation(named.Attribute(), schemaFingerprintInput{})
	require.Equal(t, declared.Bytes(), encoder.Bytes())
}

func TestRequiredValidationRetainsSchemaIdentity(t *testing.T) {
	plain := fingerprintObject(&expr.NamedAttributeExpr{Name: "name", Attribute: &expr.AttributeExpr{Type: expr.String}})
	required := expr.DupAtt(plain)
	required.Validation = &expr.ValidationExpr{Required: []string{"name"}}
	require.NotEqual(t, fingerprintAttribute(plain, false), fingerprintAttribute(required, false))
}
