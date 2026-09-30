package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValueCopyAncestryDoesNotAliasOccurrences(t *testing.T) {
	minimum := 1
	original := &AttributeExpr{Type: String, Validation: &ValidationExpr{MinLength: &minimum}}
	first := DupAtt(original)
	second := DupAtt(original)
	otherMinimum := 8
	second.Validation.MinLength = &otherMinimum
	require.Same(t, valueAttributeOrigin(original), valueAttributeOrigin(first))
	require.Same(t, valueAttributeOrigin(first), valueAttributeOrigin(second))
	context := NewValueContext()
	left, err := context.NewOccurrence(first)
	require.NoError(t, err)
	right, err := context.NewOccurrence(second)
	require.NoError(t, err)
	require.False(t, left.ID() == right.ID())
	require.Equal(t, 1, *left.node.attribute.Validation.MinLength)
	require.Equal(t, 8, *right.node.attribute.Validation.MinLength)
}

func TestValuePlanOwnsValidationClauseSlices(t *testing.T) {
	source := &AttributeExpr{Type: String, Validation: &ValidationExpr{
		PatternClauses: []string{"^a"},
		FormatClauses:  []ValidationFormat{FormatIP},
	}}
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(source)
	require.NoError(t, err)
	plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{
		Target: source,
		Codec:  ValueCodecJSON,
		Use:    ValuePlanSchema,
	})
	require.NoError(t, err)

	source.Validation.PatternClauses[0] = "^b"
	source.Validation.FormatClauses[0] = FormatEmail
	require.Equal(t, []string{"^a"}, occurrence.node.attribute.Validation.PatternClauses)
	require.Equal(t, []ValidationFormat{FormatIP}, occurrence.node.attribute.Validation.FormatClauses)
	actual := plan.Root().Attribute().Validation
	require.Equal(t, []string{"^a"}, actual.PatternClauses)
	require.Equal(t, []ValidationFormat{FormatIP}, actual.FormatClauses)
}

func TestValueViewAncestry(t *testing.T) {
	source := resultType("data", Bytes, view(DefaultView, "data", Bytes))
	projected, err := Project(source, DefaultView)
	require.NoError(t, err)
	require.Same(t, valueAttributeOrigin(source.AttributeExpr), valueAttributeOrigin(projected.AttributeExpr))
	require.Same(t, valueAttributeOrigin(AsObject(source.Type).Attribute("data")), valueAttributeOrigin(AsObject(projected.Type).Attribute("data")))
}

func TestValueResponseBodyPreservesSourceAndMapsMembers(t *testing.T) {
	source := &AttributeExpr{
		Type: &Object{
			{Name: "data", Attribute: &AttributeExpr{Type: String}},
			{Name: "header", Attribute: &AttributeExpr{Type: String}},
		},
		UserExamples: []*ExampleExpr{{Value: map[string]any{"data": "body", "header": "keep"}}},
	}
	response := &HTTPResponseExpr{Headers: NewMappedAttributeExpr(&AttributeExpr{Type: &Object{{Name: "header", Attribute: &AttributeExpr{Type: String}}}})}
	body := buildHTTPResponseBody("show", source, response, &HTTPServiceExpr{ServiceExpr: &ServiceExpr{Name: "service"}})
	require.Equal(t, map[string]any{"data": "body", "header": "keep"}, source.UserExamples[0].Value)
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(source)
	require.NoError(t, err)
	plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: body, Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
	require.NoError(t, err)
	require.Len(t, plan.root.alias.members, 1)
	require.Equal(t, occurrence.node.declaration.members[0].id, plan.root.alias.members[0].source)
}

func TestValueCopiedExampleUsesAuthoredSourceWhenCapturedFirst(t *testing.T) {
	authored := &ExampleExpr{Value: map[string]any{"data": "body", "header": "authored"}}
	service := &AttributeExpr{Type: Any, UserExamples: []*ExampleExpr{authored}}
	transport := DupAtt(service)
	delete(transport.UserExamples[0].Value.(map[string]any), "header")
	context := NewValueContext()
	targetOccurrence, err := context.NewOccurrence(transport)
	require.NoError(t, err)
	targetSource, found := context.SelectExample(targetOccurrence, ExamplePolicy{Reachable: true}).Source()
	require.True(t, found)
	serviceOccurrence, err := context.NewOccurrence(service)
	require.NoError(t, err)
	serviceSource, found := context.SelectExample(serviceOccurrence, ExamplePolicy{Reachable: true}).Source()
	require.True(t, found)
	require.True(t, targetSource.ID() == serviceSource.ID())
	value, present := context.Resolve(serviceOccurrence, serviceSource, ValueRoleExample).LegacyValue()
	require.True(t, present)
	require.Equal(t, map[string]any{"data": "body", "header": "authored"}, value)
	require.Equal(t, map[string]any{"data": "body"}, transport.UserExamples[0].Value)
}

func TestValueCopyOwnsCyclicBuiltinGraphsAndBorrowsCustomValues(t *testing.T) {
	cyclic := map[string]any{}
	cyclic["self"] = cyclic
	calls := 0
	custom := &snapshotCustom{calls: &calls}
	source := &AttributeExpr{Type: Any, UserExamples: []*ExampleExpr{{Value: cyclic}, {Value: custom}}}
	copy := DupAtt(source)
	copy.UserExamples[0].Value.(map[string]any)["changed"] = true
	require.NotContains(t, cyclic, "changed")
	require.Contains(t, copy.UserExamples[0].Value.(map[string]any)["self"], "changed")
	require.Same(t, custom, copy.UserExamples[1].Value)
	require.Zero(t, calls)
}
