package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestDefaultApplicationScalarValues(t *testing.T) {
	for _, tc := range []struct {
		name     string
		datatype expr.DataType
		fallback any
	}{
		{"boolean", expr.Boolean, true},
		{"integer", expr.Int, 7},
		{"string", expr.String, "fallback"},
		{"named boolean", &expr.UserTypeExpr{TypeName: "Flag", AttributeExpr: &expr.AttributeExpr{Type: expr.Boolean}}, true},
		{"named integer", &expr.UserTypeExpr{TypeName: "Count", AttributeExpr: &expr.AttributeExpr{Type: expr.Int}}, 7},
		{"named string", &expr.UserTypeExpr{TypeName: "Label", AttributeExpr: &expr.AttributeExpr{Type: expr.String}}, "fallback"},
	} {
		for _, pointer := range []bool{false, true} {
			name := tc.name + "/native-value"
			if pointer {
				name = tc.name + "/pointer-presence-control"
			}
			t.Run(name, func(t *testing.T) {
				parent := &expr.AttributeExpr{Type: &expr.Object{{Name: "value", Attribute: &expr.AttributeExpr{Type: tc.datatype, DefaultValue: tc.fallback}}}}
				scope := NewNameScope()
				source := NewAttributeContext(pointer, false, true, "", scope)
				target := NewAttributeContext(false, false, true, "", scope)
				codePlan, err := GoTransform(parent, parent, "source", "target", source, target, "", true)
				code := codePlan.Code
				require.NoError(t, err)
				require.NotContains(t, code, "== zero", "a native zero is a supplied value, not evidence of absence")
				if pointer {
					require.Contains(t, code, "source.Value == nil", "retain actual absence defaulting")
				}
			})
		}
	}
}

func TestDefaultApplicationCarrierControls(t *testing.T) {
	for _, tc := range []struct {
		name        string
		datatype    expr.DataType
		fallback    any
		pointer     bool
		json        bool
		nullable    bool
		required    bool
		targetPtr   bool
		materialize bool
		wantDefault bool
	}{
		{name: "pointer absence", datatype: expr.String, fallback: "fallback", pointer: true, materialize: true, wantDefault: true},
		{name: "native bytes nil", datatype: expr.Bytes, fallback: []byte("fallback"), materialize: true, wantDefault: true},
		{name: "optional absence", datatype: expr.String, fallback: "fallback", json: true, materialize: true, wantDefault: true},
		{name: "nullable absence only", datatype: expr.String, fallback: "fallback", nullable: true, materialize: true, wantDefault: true},
		{name: "required native", datatype: expr.String, fallback: "fallback", required: true, materialize: true},
		{name: "projected pointer", datatype: expr.String, fallback: "fallback", pointer: true, targetPtr: true, materialize: true},
		{name: "deferred defaults", datatype: expr.String, fallback: "fallback", pointer: true},
		{name: "no default", datatype: expr.String, pointer: true, materialize: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			field := &expr.AttributeExpr{Type: tc.datatype, DefaultValue: tc.fallback, Nullable: tc.nullable}
			parent := &expr.AttributeExpr{Type: &expr.Object{{Name: "value", Attribute: field}}}
			if tc.required {
				parent.Validation = &expr.ValidationExpr{Required: []string{"value"}}
			}
			scope := NewNameScope()
			source := NewAttributeContext(tc.pointer, false, true, "", scope)
			source.JSONPresence = tc.json
			target := NewAttributeContext(tc.targetPtr, false, tc.materialize, "", scope)
			codePlan, err := GoTransform(parent, parent, "source", "target", source, target, "", true)
			code := codePlan.Code
			require.NoError(t, err)
			if tc.wantDefault {
				value := expr.DupAtt(field)
				value.Nullable = false
				literalPlan, err := target.DefaultValue(value, tc.fallback)
				literal := literalPlan.Expression
				require.NoError(t, err)
				require.Contains(t, code, literal)
			} else {
				require.NotContains(t, code, `"fallback"`)
			}
			if tc.nullable {
				require.Contains(t, code, "source.Value.IsNull()")
				require.Contains(t, code, "source.Value.Value()")
			}
			require.NotContains(t, code, "== zero")
		})
	}
}
