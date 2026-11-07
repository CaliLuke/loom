package codegen

import (
	"testing"

	shared "github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/require"
)

// This uses the actual protobuf wrapper constructor, without DSL or generator
// setup. The source occurrence must be admitted before the physical mutation.
func TestPhysicalProjectionDefaultSemanticOwner(t *testing.T) {
	choice := &expr.UserTypeExpr{TypeName: "Choice", UID: "Choice", AttributeExpr: &expr.AttributeExpr{Type: &expr.Union{TypeName: "Choice", Untagged: true, Values: []*expr.NamedAttributeExpr{
		{Name: "text", Attribute: &expr.AttributeExpr{Type: expr.String}},
		{Name: "number", Attribute: &expr.AttributeExpr{Type: expr.Int}},
	}}}}
	outer := &expr.UserTypeExpr{TypeName: "Outer", UID: "Outer", AttributeExpr: &expr.AttributeExpr{Type: &expr.Union{TypeName: "Outer", Untagged: true, Values: []*expr.NamedAttributeExpr{
		{Name: "nested", Attribute: &expr.AttributeExpr{Type: choice}},
		{Name: "flag", Attribute: &expr.AttributeExpr{Type: expr.Boolean}},
	}}}}
	holder := &expr.UserTypeExpr{TypeName: "NestedHolder", UID: "NestedHolder", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{
		{Name: "outer", Attribute: &expr.AttributeExpr{Type: outer}},
	}}}
	source := &expr.AttributeExpr{Type: holder, DefaultValue: map[string]any{"outer": "nested"}}
	semantic, err := expr.NewValueContext().NewOccurrence(source)
	require.NoError(t, err, "admitted semantic input is a mandatory precondition")
	physical := expr.DupAtt(source)
	branch := expr.AsUnion(expr.AsObject(physical.Type).Attribute("outer").Type).Values[0].Attribute
	wrapUnionBranch(branch, nil) // Named union branch: actual helper does not need ServiceData.
	require.True(t, expr.IsObject(branch.Type), "actual wrapper must change physical topology")
	raw, present := semantic.EffectiveConstraints().Default()
	require.True(t, present)
	require.Equal(t, map[string]any{"outer": "nested"}, raw, "captured semantic default remains intact")
	_, physicalErr := expr.EffectiveConstraintsFor(physical)
	t.Logf("reconstructing a semantic occurrence from physical topology: %v", physicalErr)
	require.NotPanics(t, func() {
		shared.ValidationCode(physical, physical.Type.(expr.UserType), protoBufTypeContext("", shared.NewNameScope(), false), true, false, false, "body")
	}, "physical validation derives predicate facts without re-admitting the authored default")
}

// A globally followed copy origin would incorrectly accept the refined copy.
func TestPhysicalProjectionPreservesOccurrenceRefinement(t *testing.T) {
	base := &expr.AttributeExpr{Type: expr.String, DefaultValue: "short"}
	_, err := expr.EffectiveConstraintsFor(base)
	require.NoError(t, err)
	refined := expr.DupAtt(base)
	minimum := 8
	refined.Validation = &expr.ValidationExpr{MinLength: &minimum}
	_, err = expr.EffectiveConstraintsFor(refined)
	require.ErrorContains(t, err, "violates the effective contract")
}
