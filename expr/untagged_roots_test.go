package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUntaggedRootGrammar(t *testing.T) {
	named := &UserTypeExpr{TypeName: "Record", AttributeExpr: &AttributeExpr{Type: &Object{{Name: "name", Attribute: &AttributeExpr{Type: String}}}}}
	inner := &Union{TypeName: "Inner", Untagged: true, Values: []*NamedAttributeExpr{
		{Name: "Text", Attribute: &AttributeExpr{Type: String}},
		{Name: "Number", Attribute: &AttributeExpr{Type: Int}},
	}}
	for _, tc := range []struct {
		name      string
		attribute *AttributeExpr
		failure   string
	}{
		{"scalar", &AttributeExpr{Type: String}, ""},
		{"any", &AttributeExpr{Type: Any}, ""},
		{"nullable", &AttributeExpr{Type: String, Nullable: true}, ""},
		{"alias", &AttributeExpr{Type: &UserTypeExpr{TypeName: "Alias", AttributeExpr: &AttributeExpr{Type: String}}}, ""},
		{"array", &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: Int}}}, ""},
		{"map", &AttributeExpr{Type: &Map{KeyType: &AttributeExpr{Type: String}, ElemType: &AttributeExpr{Type: Int}}}, ""},
		{"object", &AttributeExpr{Type: named}, ""},
		{"nested", &AttributeExpr{Type: inner}, ""},
		{"inline object", &AttributeExpr{Type: &Object{}}, "root must be"},
		{"tagged", &AttributeExpr{Type: &Union{TypeName: "Tagged", Values: inner.Values}}, "tagged union"},
		{"opaque", &AttributeExpr{Type: String, Meta: MetaExpr{"struct:field:type": {"Custom"}}}, "opaque codec"},
		{"nonstring key", &AttributeExpr{Type: &Map{KeyType: &AttributeExpr{Type: Int}, ElemType: &AttributeExpr{Type: Int}}}, "map keys must be strings"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parent := &AttributeExpr{Type: &Union{TypeName: "Choice", Untagged: true, Values: []*NamedAttributeExpr{{Name: "First", Attribute: tc.attribute}, {Name: "Second", Attribute: &AttributeExpr{Type: Boolean}}}}}
			errors := validateUntaggedBranches(AsUnion(parent.Type), "", parent)
			if tc.failure != "" {
				require.NotEmpty(t, errors.Errors)
				require.Contains(t, errors.Error(), tc.failure)
			} else {
				require.Empty(t, errors.Errors, errors.Error())
			}
		})
	}
}
