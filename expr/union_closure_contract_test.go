package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnionClosureContractComparison(t *testing.T) {
	record := &UserTypeExpr{TypeName: "Record", AttributeExpr: &AttributeExpr{
		Type: &Object{{Name: "x", Attribute: &AttributeExpr{Type: Int}}},
	}}
	choice := &UserTypeExpr{TypeName: "Choice", AttributeExpr: &AttributeExpr{Type: &Union{
		TypeName: "Choice", Untagged: true, Values: []*NamedAttributeExpr{
			{Name: "Record", Attribute: &AttributeExpr{Type: record}},
			{Name: "Text", Attribute: &AttributeExpr{Type: String}},
		},
	}}}
	AsObject(record).Set("next", &AttributeExpr{Type: choice})
	for _, tc := range []struct {
		name   string
		change func(*UserTypeExpr)
	}{
		{"discriminator", func(copy *UserTypeExpr) {
			AsUnion(copy).TypeKey = "kind"
		}},
		{"nullable", func(copy *UserTypeExpr) {
			AsUnion(copy).Values[1].Attribute.Nullable = true
		}},
		{"constraint", func(copy *UserTypeExpr) {
			AsUnion(copy).Values[1].Attribute.Validation = &ValidationExpr{Pattern: "^a"}
		}},
		{"field metadata", func(copy *UserTypeExpr) {
			AsObject(AsUnion(copy).Values[0].Attribute.Type).Attribute("x").Meta = MetaExpr{"struct:tag:json:name": {"number"}}
		}},
		{"additional schema", func(copy *UserTypeExpr) {
			AsUnion(copy).Values[0].Attribute.Type.(UserType).Attribute().AdditionalAttributes = &AttributeExpr{Type: Int}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			copy := Dup(choice).(*UserTypeExpr)
			require.True(t, sameUnionClosureContract(choice, copy), "recursive identical copy")
			tc.change(copy)
			require.False(t, sameUnionClosureContract(choice, copy), "localized contract must remain separate")
		})
	}
}
