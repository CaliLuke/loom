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

// TestMappedAttributeRequiredElementNames checks that a mapped attribute names
// its required attributes after the attributes, without the element name
// suffix, whether the object lists them with the suffix, as a type does, or
// without it, as the HTTP mapping DSL does, and that Attribute restores the
// suffix on the keys and the required names.
func TestMappedAttributeRequiredElementNames(t *testing.T) {
	cases := []struct {
		Name             string
		Required         []string
		MappedRequired   []string
		OriginalRequired []string
	}{
		{"suffixed", []string{"req:r", "plain"}, []string{"req", "plain"}, []string{"req:r", "plain"}},
		{"attribute names", []string{"req", "plain"}, []string{"req", "plain"}, []string{"req:r", "plain"}},
		{"both forms", []string{"req:r", "req"}, []string{"req"}, []string{"req:r"}},
		{"none", nil, nil, nil},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			att := &AttributeExpr{
				Type: &Object{
					{Name: "req:r", Attribute: &AttributeExpr{Type: Int, DefaultValue: 3}},
					{Name: "plain", Attribute: &AttributeExpr{Type: String}},
					{Name: "opt:o", Attribute: &AttributeExpr{Type: String}},
				},
				Validation: &ValidationExpr{Required: c.Required},
			}
			source := append([]string(nil), c.Required...)

			mapped := NewMappedAttributeExpr(att)

			require.Equal(t, c.MappedRequired, mapped.Validation.Required)
			require.Equal(t, len(c.MappedRequired) > 0, mapped.IsRequired("req"))
			require.False(t, mapped.IsRequired("opt"))
			require.NotPanics(t, func() {
				require.False(t, mapped.IsRequiredNoDefault("req"), "req has a default value")
			})
			require.Equal(t, "r", mapped.ElemName("req"))
			original := mapped.Attribute()
			require.Equal(t, []string{"req:r", "plain", "opt:o"}, objectKeys(original))
			require.Equal(t, c.OriginalRequired, original.Validation.Required)
			require.Equal(t, source, att.Validation.Required, "the source attribute is unchanged")
		})
	}
}

// TestAttributeName checks that AttributeName drops the element name suffix.
func TestAttributeName(t *testing.T) {
	cases := map[string]string{
		"n:m":   "n",
		"n":     "n",
		"n:m:o": "n",
		"":      "",
	}
	for key, want := range cases {
		require.Equal(t, want, AttributeName(key), key)
	}
}

func objectKeys(att *AttributeExpr) []string {
	keys := make([]string, 0, len(*AsObject(att.Type)))
	for _, nat := range *AsObject(att.Type) {
		keys = append(keys, nat.Name)
	}
	return keys
}
