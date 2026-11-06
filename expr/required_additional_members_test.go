package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestRequiredAdditionalMembersStayInMap(t *testing.T) {
	for _, key := range []string{"a:b", "a,b", "", "-", "quote\"slash\\", "line\nend", "日本語", "AdditionalAttributes"} {
		t.Run(key, func(t *testing.T) {
			expr.RunDSL(t, func() {
				Type("Record", func() {
					AdditionalAttributes(Int)
					Required(key)
				})
			})
			attribute := expr.Root.Types[0].Attribute()
			require.Empty(t, attribute.PhysicalMembers(), "required extra keys are map entries, not Go fields")
			require.Empty(t, attribute.PhysicalWireNames(), "required extra keys remain available to the map")
		})
	}
}

func TestRequiredAdditionalMemberIdentityAcrossMappings(t *testing.T) {
	base := &expr.UserTypeExpr{TypeName: "Record", AttributeExpr: &expr.AttributeExpr{
		Type:                 &expr.Object{{Name: "a", Attribute: &expr.AttributeExpr{Type: expr.Int}}},
		AdditionalAttributes: &expr.AttributeExpr{Type: expr.Int},
		Validation:           &expr.ValidationExpr{Required: []string{"a:b"}},
	}}
	occurrence := &expr.AttributeExpr{Type: base, Validation: &expr.ValidationExpr{Required: []string{"a,b", "a:b"}}}
	for _, attribute := range []*expr.AttributeExpr{occurrence, expr.NewMappedAttributeExpr(occurrence).AttributeExpr} {
		require.Equal(t, []string{"a,b", "a:b"}, attribute.RequiredAdditionalMembers())
		require.False(t, expr.NewObjectFields(attribute).FieldFacts("a").Required)
		constraints, err := expr.EffectiveConstraintsFor(attribute)
		require.NoError(t, err)
		names := make([]string, 0, len(constraints.Required()))
		for _, required := range constraints.Required() {
			names = append(names, required.WireName)
		}
		require.Equal(t, []string{"a,b", "a:b"}, names)
	}
}

func TestMappedRequiredDeclaredAndAdditionalMembers(t *testing.T) {
	base := &expr.UserTypeExpr{TypeName: "Mapped", AttributeExpr: &expr.AttributeExpr{
		Type:                 &expr.Object{{Name: "a:x", Attribute: &expr.AttributeExpr{Type: expr.Int}}},
		AdditionalAttributes: &expr.AttributeExpr{Type: expr.Int},
		Validation:           &expr.ValidationExpr{Required: []string{"a:x", "a:b"}},
	}}
	mapped := expr.NewMappedAttributeExpr(&expr.AttributeExpr{Type: base})
	require.Equal(t, []string{"a", "a:b"}, mapped.AllRequired())
	require.Equal(t, []string{"a:b"}, mapped.RequiredAdditionalMembers())
	constraints, err := expr.EffectiveConstraintsFor(mapped.AttributeExpr)
	require.NoError(t, err)
	for _, req := range constraints.Required() {
		require.NotEqual(t, "a:x", req.Name)
	}
}

func TestRequiredAdditionalMemberHashIdentity(t *testing.T) {
	record := func(required ...string) *expr.Array {
		return &expr.Array{ElemType: &expr.AttributeExpr{
			Type:                 &expr.Object{},
			AdditionalAttributes: &expr.AttributeExpr{Type: expr.Int},
			Validation:           &expr.ValidationExpr{Required: required},
		}}
	}
	left, right := record("a-b"), record("a", "b")
	for _, ignoreFields := range []bool{true, false} {
		require.NotEqual(t, expr.Hash(left, ignoreFields, false, false), expr.Hash(right, ignoreFields, false, false))
	}
	require.False(t, expr.Equal(left, right))
}
