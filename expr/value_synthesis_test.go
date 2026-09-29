package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValueSynthesisRetainsOneChoiceAndLegacyValue(t *testing.T) {
	attribute := &AttributeExpr{Type: &Union{Values: []*NamedAttributeExpr{
		{Name: "Left", Attribute: &AttributeExpr{Type: String}},
		{Name: "Right", Attribute: &AttributeExpr{Type: String}},
	}}}
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	selection := context.SelectExample(occurrence, ExamplePolicy{Reachable: true})
	random := NewRandom("retained-choice")
	result := context.Synthesize(selection, random)
	require.Equal(t, ValueResolved, result.Outcome(), "%v", result.Diagnostics())
	value, ok := result.Value()
	require.True(t, ok)
	_, branch, payload, selected := value.Union()
	require.True(t, selected)
	require.False(t, branch == ValueIdentity{})
	legacy, present := result.LegacyValue()
	require.True(t, present)
	scalar, present := payload.Scalar()
	require.True(t, present)
	require.Equal(t, scalar, legacy)
	// The legacy algorithm consumes the same one branch choice and string draw.
	require.Equal(t, attribute.Example(NewRandom("retained-choice")), legacy)
	again := context.Synthesize(selection, random)
	againValue, ok := again.Value()
	require.True(t, ok)
	require.True(t, againValue.ID() == value.ID())
}

func TestValueSynthesisCannotReplaceAuthoredFailure(t *testing.T) {
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(&AttributeExpr{Type: String, UserExamples: []*ExampleExpr{{Value: 17}}})
	require.NoError(t, err)
	selection := context.SelectExample(occurrence, ExamplePolicy{Reachable: true})
	result := context.Synthesize(selection, nil)
	require.Equal(t, ValueInvalid, result.Outcome())
	_, present := result.LegacyValue()
	require.False(t, present)
}

func TestValueSynthesisNamedUnionRetainsChoice(t *testing.T) {
	union := &UserTypeExpr{TypeName: "NamedChoice", AttributeExpr: &AttributeExpr{Type: &Union{Values: []*NamedAttributeExpr{
		{Name: "Left", Attribute: &AttributeExpr{Type: String}},
		{Name: "Right", Attribute: &AttributeExpr{Type: String}},
	}}}}
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(&AttributeExpr{Type: union})
	require.NoError(t, err)
	selection := context.SelectExample(occurrence, ExamplePolicy{Reachable: true})
	result := context.Synthesize(selection, NewRandom("named-retained-choice"))
	require.Equal(t, ValueResolved, result.Outcome(), "%v", result.Diagnostics())
	value, present := result.Value()
	require.True(t, present)
	identity, branch, payload, selected := value.Union()
	require.True(t, selected)
	require.True(t, identity == occurrence.Underlying().ID())
	require.False(t, branch == (ValueIdentity{}))
	legacy, present := result.LegacyValue()
	require.True(t, present)
	scalar, present := payload.Scalar()
	require.True(t, present)
	require.Equal(t, scalar, legacy)
}
