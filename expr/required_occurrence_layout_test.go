package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMappedOccurrenceRetainsDeclarationLayout(t *testing.T) {
	field := &AttributeExpr{Type: String}
	declaration := &UserTypeExpr{TypeName: "Base", AttributeExpr: &AttributeExpr{
		Type: &Object{{Name: "name:wire", Attribute: field}},
	}}
	occurrence := &AttributeExpr{Type: declaration, Validation: &ValidationExpr{Required: []string{"name"}}}
	mapped := NewMappedAttributeExpr(occurrence)
	for _, tc := range []struct {
		name   string
		mapped *MappedAttributeExpr
	}{
		{"mapped", mapped}, {"copy", DupMappedAtt(mapped)},
		{"restored", NewMappedAttributeExpr(mapped.Attribute())},
	} {
		t.Run(tc.name, func(t *testing.T) {
			facts := tc.mapped.FieldFacts("name")
			require.True(t, facts.Required)
			require.False(t, facts.LayoutRequired)
			require.False(t, NewObjectFields(tc.mapped.AttributeExpr).FieldFacts("name").LayoutRequired)
			require.True(t, ServiceFieldLayout(tc.mapped.AttributeExpr, "name", field, false, true).Pointer)
			require.Equal(t, "wire", facts.ElementName)
		})
	}
}
