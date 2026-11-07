package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInlineJSONSchemaUntaggedClosure(t *testing.T) {
	for _, tc := range []struct {
		name              string
		named             bool
		closed            bool
		nullable          bool
		inheritedNullable bool
	}{
		{"open", false, false, false, false},
		{"closed", false, true, false, false},
		{"nullable closed", false, true, true, false},
		{"named closed", true, true, false, false},
		{"named nullable closed", true, true, true, false},
		{"inherited nullable closed", true, true, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			union := &Union{Untagged: true, Values: []*NamedAttributeExpr{
				{Name: "Anything", Attribute: &AttributeExpr{Type: Any}},
				{Name: "Text", Attribute: &AttributeExpr{Type: String}},
			}}
			var typ DataType = union
			if tc.named {
				typ = &UserTypeExpr{TypeName: "Choice", AttributeExpr: &AttributeExpr{Type: union, Nullable: tc.inheritedNullable}}
			}
			attribute := &AttributeExpr{Type: typ, Nullable: tc.nullable}
			if tc.closed {
				attribute.Meta = MetaExpr{unionClosureMeta: {"false"}}
			}
			schema := mustInlineJSONSchema(t, attribute)
			if tc.nullable || tc.inheritedNullable {
				require.NotContains(t, schema, "unevaluatedProperties")
				arms := schema["anyOf"].([]any)
				require.Len(t, arms, 2)
				require.Equal(t, map[string]any{"type": "null"}, arms[1])
				schema = arms[0].(map[string]any)
			}
			if tc.closed {
				require.Equal(t, false, schema["unevaluatedProperties"])
			} else {
				require.NotContains(t, schema, "unevaluatedProperties")
			}
			branches := schema["oneOf"].([]any)
			require.Len(t, branches, 2)
			for _, branch := range branches {
				require.NotContains(t, branch, "unevaluatedProperties")
			}
			require.False(t, union.Closed(), "export must not mutate the declaration")
		})
	}
}
