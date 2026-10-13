package codegen

import (
	"testing"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/stretchr/testify/require"
)

func TestJSONNativeZeroProjectionReadiness(t *testing.T) {
	for _, tc := range []struct {
		name               string
		typ                expr.DataType
		zero, defaultValue any
	}{
		{"string", expr.String, "", "fallback"},
		{"boolean", expr.Boolean, false, true},
		{"integer", expr.Int, 0, 7},
	} {
		for _, tag := range []string{"value,omitzero", "value,omitempty,omitzero", "value,omitzero,omitempty"} {
			t.Run(tc.name+"/"+tag, func(t *testing.T) {
				field := &expr.AttributeExpr{Type: tc.typ, DefaultValue: tc.defaultValue, Meta: expr.MetaExpr{"struct:tag:json": {tag}}}
				attribute := &expr.AttributeExpr{Type: &expr.Object{{Name: "value", Attribute: field}}}
				context := expr.NewValueContext()
				occurrence, err := context.NewOccurrence(attribute)
				require.NoError(t, err)
				resolved := context.Resolve(occurrence, context.SupplyValue(expr.ValueInput{Raw: map[string]any{"value": tc.zero}}), expr.ValueRoleExample)
				require.Equal(t, expr.ValueResolved, resolved.Outcome())
				target := &transportir.ValueTarget{Source: &service.ValueData{Context: context, Occurrence: occurrence, Example: resolved}, Codec: expr.ValueCodecJSON}
				planned := representation.BuildValuePlan(target, attribute, httpContext(codegen.NewNameScope(), false, true), expr.ValuePlanRuntime)
				require.NoError(t, planned.Error)
				projected := context.ProjectJSON(resolved, planned.Plan)
				require.Equal(t, expr.ProjectionEmitted, projected.Outcome(), projected.Diagnostics())
				wire, present := projected.JSON()
				require.True(t, present)
				require.JSONEq(t, `{}`, string(wire), "native omitzero follows supplied Go zero, not only absence")
			})
		}
	}
}
