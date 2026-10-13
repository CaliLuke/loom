package codegen

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestNamedValidationEligibilityUsesDeclaredLayout(t *testing.T) {
	for _, pointer := range []bool{false, true} {
		for _, shape := range []string{"leaf", "union", "array", "map", "recursive"} {
			t.Run(fmt.Sprintf("%s/pointer=%t", shape, pointer), func(t *testing.T) {
				leaf, root := namedValidationLayoutGraph(shape)
				ctx := NewAttributeContext(!pointer, false, false, "", NewNameScope())
				ctx.PresencePointerTypes = map[string]bool{leaf.Hash(): pointer}
				require.Equal(t, pointer || shape == "union", hasValidations(ctx, root), "eligibility must follow the named declaration's fields, independently of caller storage")
				require.Equal(t, !pointer, ctx.Pointer, "eligibility must not mutate caller mode")
			})
		}
	}
}

func TestNestedUnionValidationRetainsRequiredNamedLeaf(t *testing.T) {
	leaf, root := namedValidationLayoutGraph("union")
	ctx := NewAttributeContext(true, false, false, "", NewNameScope())
	ctx.PresencePointerTypes = map[string]bool{leaf.Hash(): true}
	code := ValidationCode(root.Attribute(), root, ctx, true, false, false, "body")
	require.Contains(t, code, "ValidateLeaf(")
	require.Contains(t, code, `case "nested":`)
	leafFunction := NamedValidationFunction(&expr.AttributeExpr{Type: leaf}, ctx, "body")
	leafCode := ValidationCode(leaf.Attribute(), leaf, leafFunction.Context(ctx), true, false, false, "body")
	require.Contains(t, leafCode, "body.Name == nil")
}

func namedValidationLayoutGraph(shape string) (leaf, root expr.UserType) {
	leaf = &expr.UserTypeExpr{TypeName: "Leaf", UID: "Leaf", AttributeExpr: &expr.AttributeExpr{
		Type:       &expr.Object{{Name: "name", Attribute: &expr.AttributeExpr{Type: expr.String}}},
		Validation: &expr.ValidationExpr{Required: []string{"name"}},
	}}
	if shape == "leaf" {
		return leaf, leaf
	}
	inner := &expr.Union{Values: []*expr.NamedAttributeExpr{{Name: "leaf", Attribute: &expr.AttributeExpr{Type: leaf}}}}
	rootAtt := &expr.AttributeExpr{Type: &expr.Union{Values: []*expr.NamedAttributeExpr{{Name: "nested", Attribute: &expr.AttributeExpr{Type: inner}}}}}
	switch shape {
	case "array":
		rootAtt.Type = &expr.Array{ElemType: &expr.AttributeExpr{Type: leaf}}
	case "map":
		rootAtt.Type = &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: leaf}}
	case "recursive":
		rootAtt.Type = &expr.Object{{Name: "leaf", Attribute: &expr.AttributeExpr{Type: leaf}}}
	}
	root = &expr.UserTypeExpr{TypeName: "Root", UID: "Root", AttributeExpr: rootAtt}
	if shape == "recursive" {
		object := rootAtt.Type.(*expr.Object)
		*object = append(*object, &expr.NamedAttributeExpr{Name: "next", Attribute: &expr.AttributeExpr{Type: root}})
	}
	return leaf, root
}

func TestNamedValidationEligibilityIgnoresInlineUnionValueRequiredness(t *testing.T) {
	inline := &expr.AttributeExpr{
		Type:       &expr.Object{{Name: "name", Attribute: &expr.AttributeExpr{Type: expr.String}}},
		Validation: &expr.ValidationExpr{Required: []string{"name"}},
	}
	root := &expr.UserTypeExpr{TypeName: "Holder", UID: "Holder", AttributeExpr: &expr.AttributeExpr{
		Type: &expr.Object{{Name: "choice", Attribute: &expr.AttributeExpr{Type: &expr.Union{Values: []*expr.NamedAttributeExpr{{Name: "inline", Attribute: inline}}}}}},
	}}
	ctx := NewAttributeContext(true, false, false, "", NewNameScope())
	require.Empty(t, ValidationCode(root.Attribute(), root, ctx, true, false, false, "body"))
	require.False(t, hasValidations(ctx, root), "inline union object primitives use values, so their requiredness cannot require a helper")
}

func TestValidationEligibilityRevisitsDistinctLayouts(t *testing.T) {
	for _, unionFirst := range []bool{false, true} {
		t.Run(fmt.Sprintf("union-first=%t", unionFirst), func(t *testing.T) {
			shared := &expr.AttributeExpr{
				Type:       &expr.Object{{Name: "name", Attribute: &expr.AttributeExpr{Type: expr.String}}},
				Validation: &expr.ValidationExpr{Required: []string{"name"}},
			}
			fields := &expr.Object{
				{Name: "plain", Attribute: shared},
				{Name: "choice", Attribute: &expr.AttributeExpr{Type: &expr.Union{Values: []*expr.NamedAttributeExpr{{Name: "inline", Attribute: shared}}}}},
			}
			if unionFirst {
				(*fields)[0], (*fields)[1] = (*fields)[1], (*fields)[0]
			}
			root := &expr.UserTypeExpr{TypeName: "Holder", UID: "Holder", AttributeExpr: &expr.AttributeExpr{Type: fields}}
			ctx := NewAttributeContext(true, false, false, "", NewNameScope())
			require.True(t, hasValidations(ctx, root), "the value-layout visit must not suppress a later pointer-layout visit")
		})
	}
}

func TestNamedValidationEligibilityIncludesUnionPointerPresence(t *testing.T) {
	leaf := &expr.UserTypeExpr{TypeName: "Empty", UID: "empty", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{}}}
	choice := &expr.AttributeExpr{Type: &expr.Union{Values: []*expr.NamedAttributeExpr{{Name: "empty", Attribute: &expr.AttributeExpr{Type: leaf}}}}}
	root := &expr.UserTypeExpr{TypeName: "Holder", UID: "holder", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{{Name: "choice", Attribute: choice}}}}
	ctx := NewAttributeContext(false, false, false, "", NewNameScope())
	require.Contains(t, ValidationCode(root.Attribute(), root, ctx, true, false, false, "body"), `loom.MissingFieldError("value"`)
	require.True(t, hasValidations(ctx, root), "named eligibility must retain the physical pointer presence check emitted by its body")
}
