package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSchemaValidationDoesNotAdmitUnsupportedCodecs(t *testing.T) {
	for _, schemaFirst := range []bool{false, true} {
		name := "runtime-first"
		if schemaFirst {
			name = "schema-first"
		}
		t.Run(name, func(t *testing.T) {
			union := &AttributeExpr{Type: &Union{TypeName: "ScalarChoice", Untagged: true, Values: []*NamedAttributeExpr{
				{Name: "text", Attribute: &AttributeExpr{Type: String}},
				{Name: "number", Attribute: &AttributeExpr{Type: Int}},
			}}}
			attribute := &AttributeExpr{Type: &Map{KeyType: &AttributeExpr{Type: String}, ElemType: &AttributeExpr{
				Type: &Array{ElemType: &AttributeExpr{Type: &Object{{Name: "choice", Attribute: union}}}},
			}}}
			checkSchema := func() {
				require.Empty(t, attribute.validateSchema("documentation", attribute).Errors)
			}
			checkRuntime := func() {
				require.ErrorContains(t, attribute.Validate("runtime", attribute), "must be a concrete named object or array type")
			}
			if schemaFirst {
				checkSchema()
				checkRuntime()
			} else {
				checkRuntime()
				checkSchema()
			}
		})
	}
}

func TestSchemaValidationPreservesContractChecks(t *testing.T) {
	for _, tc := range []struct {
		name      string
		attribute *AttributeExpr
		message   string
	}{
		{
			name:      "required field",
			attribute: &AttributeExpr{Type: &Object{}, Validation: &ValidationExpr{Required: []string{"missing"}}},
			message:   `required field "missing" does not exist`,
		},
		{
			name: "duplicate union identity",
			attribute: &AttributeExpr{Type: &Union{TypeName: "Choice", Untagged: true, Values: []*NamedAttributeExpr{
				{Name: "same", Attribute: &AttributeExpr{Type: String}},
				{Name: "same", Attribute: &AttributeExpr{Type: Int}},
			}}},
			message: "repeats logical name",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.ErrorContains(t, tc.attribute.validateSchema("documentation", tc.attribute), tc.message)
		})
	}
}
