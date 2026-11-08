package ir

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestSchemaRequiredNamesUsesEffectiveObjectContract(t *testing.T) {
	root := codegen.RunDSL(t, testdata.InheritedRequiredProjectionDSL)
	var row, nested *expr.AttributeExpr
	for _, typ := range root.Types {
		switch typ.Name() {
		case "Row":
			row = typ.Attribute()
		case "DerivedGroup":
			nested = expr.AsArray(typ.Attribute().Find("items").Type).ElemType.Type.(expr.UserType).Attribute()
		}
	}
	require.NotNil(t, row)
	require.NotNil(t, nested)
	require.ElementsMatch(t, []string{"extra", "id"}, schemaRequiredNames(nested))
	require.Equal(t, fingerprintAttribute(row, false), fingerprintAttribute(nested, false), "detached composition copies keep the same schema identity")
	for _, s := range root.API.HTTP.Services {
		if s.Name() != "uploads" {
			continue
		}
		body := s.HTTPEndpoints[0].Body.Type.(expr.UserType).Attribute()
		require.ElementsMatch(t, []string{"file", "filename"}, schemaRequiredNames(body))
		context := expr.NewValueContext()
		occurrence, err := context.NewOccurrence(body)
		require.NoError(t, err)
		plan, err := context.NewValuePlan(occurrence, expr.ValuePlanRequest{TargetOccurrence: occurrence, Codec: expr.ValueCodecMultipart, Use: expr.ValuePlanSchema})
		require.NoError(t, err)
		require.ElementsMatch(t, []string{"file", "filename"}, schemaRequiredNames(plan.Root().Attribute()), "captured transport roots preserve effective requirements")
	}
	// Named occurrences retain their own requirements beside the declaration.
	overlay := &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "Row", AttributeExpr: row}, Validation: &expr.ValidationExpr{Required: []string{"extra"}}}
	require.Equal(t, []string{"extra"}, schemaRequiredNames(overlay))
	require.ElementsMatch(t, []string{"extra", "id"}, row.Validation.Required, "queries cannot mutate the declaration")
}
