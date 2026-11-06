package representation

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestRequiredOccurrenceFieldUsesDeclarationPresence(t *testing.T) {
	field := &expr.AttributeExpr{Type: expr.String}
	base := &expr.UserTypeExpr{TypeName: "Base", AttributeExpr: &expr.AttributeExpr{
		Type: &expr.Object{{Name: "name", Attribute: field}},
	}}
	occurrence := &expr.AttributeExpr{Type: base, Validation: &expr.ValidationExpr{Required: []string{"name"}}}
	facts := expr.NewObjectFields(occurrence)
	require.True(t, facts.FieldFacts("name").Required)
	require.False(t, facts.FieldFacts("name").LayoutRequired)
	decoded := ResolveField(facts, "name", field, true, false, true)
	require.True(t, decoded.Layout.WrapOptional)
	require.False(t, decoded.Layout.Pointer)
	encoded := ResolveField(facts, "name", field, false, true, false)
	require.True(t, encoded.Layout.Pointer)
	require.Contains(t, encoded.Encoding.Tags, "omitzero")
}
