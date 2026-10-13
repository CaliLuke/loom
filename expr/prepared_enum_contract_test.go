package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPreparedEnumDefaultsAndOwnedCopies(t *testing.T) {
	for _, n := range []int{1, 2, 3} {
		source := preparedEnumSource(t, false)
		source.DefaultValue = map[string]any{"n": n}
		mapped := NewMappedAttributeExpr(source)
		_, sourceErr := NewValueContext().NewOccurrence(source)
		_, mappedErr := NewValueContext().NewOccurrence(mapped.AttributeExpr)
		if n == 2 {
			require.NoError(t, sourceErr)
			require.NoError(t, mappedErr)
		} else {
			require.ErrorContains(t, sourceErr, "default value")
			require.ErrorContains(t, mappedErr, "default value")
		}
	}
	source := preparedEnumSource(t, false)
	first, second := NewMappedAttributeExpr(source), NewMappedAttributeExpr(source)
	first.Find("n").Validation.Minimum = boundForTest(9)
	require.Equal(t, "2", source.Find("n").Validation.Minimum.String())
	require.Equal(t, "2", second.Find("n").Validation.Minimum.String())
	assertPreparedEnumContract(t, second.AttributeExpr)
}

func TestPreparedMappingAllowsIncompleteDSLFields(t *testing.T) {
	mapped := NewEmptyMappedAttributeExpr()
	AsObject(mapped.Type).Set("n:wire_n", &AttributeExpr{})
	mapped.Remap()
	mapped.Find("n").Type = Int
	mapped.Validation = &ValidationExpr{Required: []string{"n"}}
	mapped.Remap()
	require.True(t, mapped.IsRequired("n"))
	constraints, err := EffectiveConstraintsFor(mapped.Attribute())
	require.NoError(t, err)
	require.Len(t, constraints.Required(), 1)
	require.Equal(t, "n:wire_n", constraints.Required()[0].Name)
}

func TestPreparedEnumLogicalMemberBindingGuards(t *testing.T) {
	source := &AttributeExpr{Type: &Object{{Name: "n", Attribute: &AttributeExpr{Type: Int}}}}
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(source)
	require.NoError(t, err)
	target := DupAtt(source)
	target.Type = &Object{{Name: "n:wire_n", Attribute: &AttributeExpr{Type: Int}}}
	_, err = context.NewValuePlan(occurrence, ValuePlanRequest{Target: target, Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
	require.ErrorContains(t, err, "unrelated member", "logical names must not bypass ordinary ancestry checks")

	ambiguous := &AttributeExpr{Type: &Object{
		{Name: "n:first", Attribute: &AttributeExpr{Type: Int}},
		{Name: "n:second", Attribute: &AttributeExpr{Type: Int}},
	}}
	ancestor, err := context.NewOccurrence(ambiguous)
	require.NoError(t, err)
	physical, err := context.NewOccurrence(target)
	require.NoError(t, err)
	builder := &valuePlanBuilder{enumConstant: true, context: context, source: ancestor,
		request: ValuePlanRequest{Codec: ValueCodecJSON, Use: ValuePlanDocumentation},
		nodes:   make(map[valuePlanKey]*valuePlanNode), assocs: make(map[*valueOccurrenceNode][]*valuePlanAssociation)}
	_, err = builder.node(ancestor.node, physical.node, ValueCodecJSON)
	require.ErrorContains(t, err, "ambiguous member ancestry")
}
