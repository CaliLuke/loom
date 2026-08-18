package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewMappedAttributeExprPreservesValidationThroughAliases(t *testing.T) {
	base := &UserTypeExpr{
		TypeName: "Widget",
		AttributeExpr: &AttributeExpr{
			Type:       &Object{{Name: "name", Attribute: &AttributeExpr{Type: String}}},
			Validation: &ValidationExpr{Required: []string{"name"}},
		},
	}
	alias := &UserTypeExpr{
		TypeName:      "NullableWidget",
		AttributeExpr: &AttributeExpr{Type: base, Nullable: true},
	}

	mapped := NewMappedAttributeExpr(&AttributeExpr{Type: alias})

	require.True(t, mapped.IsRequired("name"))
	require.Equal(t, []string{"name"}, mapped.Validation.Required)
}
