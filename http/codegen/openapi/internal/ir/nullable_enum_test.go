package ir

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestNullableEnumOccurrenceRefinement(t *testing.T) {
	for _, nullable := range []bool{false, true} {
		base := &expr.UserTypeExpr{TypeName: "Status", AttributeExpr: &expr.AttributeExpr{
			Type: expr.String, Nullable: nullable,
			Validation: &expr.ValidationExpr{Values: []any{"open", "closed"}},
		}}
		attribute := &expr.AttributeExpr{Type: base, Validation: &expr.ValidationExpr{Values: []any{"open"}}}
		analyzer := NewAnalyzer(expr.NewRandom("nullable-enum"), false)
		schema := analyzer.AnalyzeSchema(attribute)
		if nullable {
			require.Len(t, schema.AnyOf, 2)
			require.Equal(t, "null", schema.AnyOf[1].Type)
			require.Empty(t, schema.Enum)
			schema = schema.AnyOf[0]
		}
		require.Equal(t, []any{"open"}, schema.Enum)
		require.Len(t, schema.AllOf, 1, "retain the base type's constraints")
	}
}
