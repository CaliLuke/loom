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

func TestJSONOmissionReadinessBodyTags(t *testing.T) {
	for _, tc := range []struct {
		name       string
		attribute  *expr.AttributeExpr
		required   bool
		useDefault bool
		want       string
	}{
		{"optional string", &expr.AttributeExpr{Type: expr.String}, false, true, `json:"value,omitzero"`},
		{"optional bytes", &expr.AttributeExpr{Type: expr.Bytes}, false, true, `json:"value,omitzero"`},
		{"optional array", &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}}, false, true, `json:"value,omitzero"`},
		{"optional map", &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: expr.String}}}, false, true, `json:"value,omitzero"`},
		{"optional object", &expr.AttributeExpr{Type: &expr.Object{}}, false, true, `json:"value,omitzero"`},
		{"required scalar control", &expr.AttributeExpr{Type: expr.String}, true, true, `json:"value"`},
		{"materialized default control", &expr.AttributeExpr{Type: expr.String, DefaultValue: "fallback"}, false, true, `json:"value"`},
		{"nullable control", &expr.AttributeExpr{Type: expr.String, Nullable: true}, false, true, `json:"value,omitzero"`},
		{"name metadata", &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{"struct:tag:json:name": {"wire"}}}, false, true, `json:"wire,omitzero"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parent := &expr.AttributeExpr{Type: &expr.Object{{Name: "value", Attribute: tc.attribute}}}
			if tc.required {
				parent.Validation = &expr.ValidationExpr{Required: []string{"value"}}
			}
			definition := goTypeDef(codegen.NewNameScope(), parent, false, tc.useDefault, false)
			require.Contains(t, definition, tc.want)
		})
	}
}

func TestJSONOmissionReadinessValueProjection(t *testing.T) {
	for _, tc := range []struct {
		name  string
		typ   expr.DataType
		empty any
		wire  string
	}{
		{"string", expr.String, "", `{"value":""}`},
		{"bytes", expr.Bytes, []byte{}, `{"value":""}`},
		{"array", &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}, []string{}, `{"value":[]}`},
		{"map", &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: expr.String}}, map[string]string{}, `{"value":{}}`},
		{"object", &expr.Object{}, map[string]any{}, `{"value":{}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			attribute := &expr.AttributeExpr{Type: &expr.Object{{Name: "value", Attribute: &expr.AttributeExpr{Type: tc.typ}}}}
			context := expr.NewValueContext()
			occurrence, err := context.NewOccurrence(attribute)
			require.NoError(t, err)
			resolved := context.Resolve(occurrence, context.SupplyValue(expr.ValueInput{Raw: map[string]any{"value": tc.empty}}), expr.ValueRoleExample)
			require.Equal(t, expr.ValueResolved, resolved.Outcome())
			target := &transportir.ValueTarget{Source: &service.ValueData{Context: context, Occurrence: occurrence, Example: resolved}, Codec: expr.ValueCodecJSON}
			planned := representation.BuildValuePlan(target, attribute, httpContext(codegen.NewNameScope(), false, true), expr.ValuePlanRuntime)
			require.NoError(t, planned.Error)
			projected := context.ProjectJSON(resolved, planned.Plan)
			require.Equal(t, expr.ProjectionEmitted, projected.Outcome(), projected.Diagnostics())
			wire, present := projected.JSON()
			require.True(t, present)
			require.JSONEq(t, tc.wire, string(wire))
		})
	}
}
