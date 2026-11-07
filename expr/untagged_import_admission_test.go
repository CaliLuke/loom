package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateUntaggedUnionDetachedGraph(t *testing.T) {
	for _, tc := range []struct {
		name  string
		root  DataType
		valid bool
	}{
		{"scalar", String, true},
		{"any", Any, true},
		{"string map", &Map{KeyType: &AttributeExpr{Type: String}, ElemType: &AttributeExpr{Type: Any}}, true},
		{"integer map", &Map{KeyType: &AttributeExpr{Type: Int}, ElemType: &AttributeExpr{Type: String}}, false},
		{"anonymous object", &Object{}, false},
		{"named object", &UserTypeExpr{TypeName: "Record", AttributeExpr: &AttributeExpr{Type: &Object{}}}, true},
		{"tagged union", &Union{TypeName: "Tagged", Values: []*NamedAttributeExpr{{Name: "A", Attribute: &AttributeExpr{Type: String}}}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			union := &Union{TypeName: "Detached", Untagged: true, Values: []*NamedAttributeExpr{{Name: "Value", Attribute: &AttributeExpr{Type: tc.root}}}}
			errors := ValidateUntaggedUnion(union)
			if tc.valid {
				require.Empty(t, errors.Errors)
			} else {
				require.NotEmpty(t, errors.Errors)
				require.Contains(t, errors.Error(), "Value")
			}
			union.Values[0].Attribute.Type = String
			require.Empty(t, ValidateUntaggedUnion(union).Errors, "validation must not reuse global visited state")
		})
	}
}

func TestValidateUntaggedUnionConsumption(t *testing.T) {
	union := &Union{TypeName: "Recursive", Untagged: true}
	alias := &UserTypeExpr{TypeName: "Node", AttributeExpr: &AttributeExpr{Type: union}}
	union.Values = []*NamedAttributeExpr{{Name: "Self", Attribute: &AttributeExpr{Type: alias}}}
	require.Contains(t, ValidateUntaggedUnion(union).Error(), "unconsumed")
	object := Object{&NamedAttributeExpr{Name: "child", Attribute: &AttributeExpr{Type: alias}}}
	union.Values[0].Attribute.Type = &UserTypeExpr{TypeName: "Record", AttributeExpr: &AttributeExpr{Type: &object}}
	require.Empty(t, ValidateUntaggedUnion(union).Errors)
}
