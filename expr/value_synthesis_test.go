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

func TestValueSynthesisUsesEffectiveNamedBounds(t *testing.T) {
	minimum, maximum := 100.0, 200.0
	base := namedScalar("SynthesisBase", Float64, &ValidationExpr{Minimum: &minimum})
	derived := namedScalar("SynthesisDerived", base, &ValidationExpr{Maximum: &maximum})
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(&AttributeExpr{Type: derived})
	require.NoError(t, err)

	selection := context.SelectExample(occurrence, ExamplePolicy{Reachable: true})
	result := context.Synthesize(selection, NewRandom("effective-named-bounds"))
	require.Equal(t, ValueResolved, result.Outcome(), "%v", result.Diagnostics())
	value, present := result.Value()
	require.True(t, present)
	scalar, present := value.Scalar()
	require.True(t, present)
	number := scalar.(float64)
	require.GreaterOrEqual(t, number, minimum)
	require.LessOrEqual(t, number, maximum)
}

func TestValueSynthesisSuppressesEmptyEffectiveStringPredicates(t *testing.T) {
	for _, test := range []struct {
		name string
		base *ValidationExpr
		last *ValidationExpr
	}{
		{name: "patterns", base: &ValidationExpr{Pattern: "^a$"}, last: &ValidationExpr{Pattern: "^b$"}},
		{name: "formats", base: &ValidationExpr{Format: FormatEmail}, last: &ValidationExpr{Format: FormatHostname}},
	} {
		t.Run(test.name, func(t *testing.T) {
			base := namedScalar("SynthesisBase", String, test.base)
			derived := namedScalar("SynthesisDerived", base, test.last)
			context := NewValueContext()
			occurrence, err := context.NewOccurrence(&AttributeExpr{Type: derived})
			require.NoError(t, err)
			result := context.Synthesize(
				context.SelectExample(occurrence, ExamplePolicy{Reachable: true}),
				NewRandom("empty-effective-string-predicates"),
			)
			require.Equal(t, ValueSuppressed, result.Outcome(), "%v", result.Diagnostics())
			_, present := result.Value()
			require.False(t, present)
		})
	}
}

func TestValueSynthesisFiltersEffectiveEnumByFormat(t *testing.T) {
	const validUUID = "550e8400-e29b-41d4-a716-446655440000"
	for _, test := range []struct {
		name    string
		values  []any
		outcome ValueOutcome
		value   any
	}{
		{name: "empty", values: []any{"not-a-uuid"}, outcome: ValueSuppressed},
		{name: "mixed", values: []any{"not-a-uuid", validUUID}, outcome: ValueResolved, value: validUUID},
	} {
		t.Run(test.name, func(t *testing.T) {
			base := namedScalar("SynthesisFormatBase", String, &ValidationExpr{Values: test.values})
			derived := namedScalar("SynthesisFormatDerived", base, &ValidationExpr{Format: FormatUUID})
			context := NewValueContext()
			occurrence, err := context.NewOccurrence(&AttributeExpr{Type: derived})
			require.NoError(t, err)

			result := context.Synthesize(
				context.SelectExample(occurrence, ExamplePolicy{Reachable: true}),
				NewRandom("effective-enum-format"),
			)
			require.Equal(t, test.outcome, result.Outcome(), "%v", result.Diagnostics())
			if test.outcome == ValueResolved {
				value, present := result.LegacyValue()
				require.True(t, present)
				require.Equal(t, test.value, value)
			}
		})
	}
}

func TestCanonicalizeExampleUsesEffectiveFormatToSelectUnionBranch(t *testing.T) {
	const validUUID = "550e8400-e29b-41d4-a716-446655440000"
	base := namedScalar("CanonicalFormatBase", String, &ValidationExpr{Values: []any{validUUID}})
	derived := namedScalar("CanonicalFormatDerived", base, &ValidationExpr{Format: FormatUUID})
	attribute := &AttributeExpr{Type: &Union{Values: []*NamedAttributeExpr{
		{Name: "Identifier", Attribute: &AttributeExpr{Type: derived}},
		{Name: "Address", Attribute: &AttributeExpr{Type: String, Validation: &ValidationExpr{Format: FormatEmail}}},
	}}}

	require.Equal(t, map[string]any{"type": "Identifier", "value": validUUID},
		CanonicalizeExample(attribute, validUUID))
}

func TestValueSynthesisSuppressesEmptyEffectiveLengthContracts(t *testing.T) {
	shapes := []struct {
		name string
		typ  DataType
	}{
		{name: "string", typ: String},
		{name: "bytes", typ: Bytes},
		{name: "array", typ: &Array{ElemType: &AttributeExpr{Type: String}}},
		{name: "map", typ: &Map{
			KeyType:  &AttributeExpr{Type: String},
			ElemType: &AttributeExpr{Type: String},
		}},
	}
	intervals := []struct {
		name    string
		minimum int
		maximum int
	}{
		{name: "separated", minimum: 5, maximum: 3},
		{name: "zero boundary", minimum: 1, maximum: 0},
	}
	for _, shape := range shapes {
		for _, interval := range intervals {
			t.Run(shape.name+"/"+interval.name, func(t *testing.T) {
				base := namedScalar("SynthesisLengthBase", shape.typ, &ValidationExpr{MinLength: &interval.minimum})
				derived := namedScalar("SynthesisLengthDerived", base, &ValidationExpr{MaxLength: &interval.maximum})
				context := NewValueContext()
				occurrence, err := context.NewOccurrence(&AttributeExpr{Type: derived})
				require.NoError(t, err)
				result := context.Synthesize(
					context.SelectExample(occurrence, ExamplePolicy{Reachable: true}),
					NewRandom("empty-effective-length"),
				)
				require.Equal(t, ValueSuppressed, result.Outcome(), "%v", result.Diagnostics())
				_, present := result.Value()
				require.False(t, present)
			})
		}
	}
}

func TestValueSynthesisSuppressesEqualExclusiveNumericContract(t *testing.T) {
	bound := 5.0
	base := namedScalar("SynthesisNumericBase", Float64, &ValidationExpr{Minimum: &bound})
	derived := namedScalar("SynthesisNumericDerived", base, &ValidationExpr{ExclusiveMaximum: &bound})
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(&AttributeExpr{Type: derived})
	require.NoError(t, err)
	result := context.Synthesize(
		context.SelectExample(occurrence, ExamplePolicy{Reachable: true}),
		NewRandom("empty-effective-number"),
	)
	require.Equal(t, ValueSuppressed, result.Outcome(), "%v", result.Diagnostics())
	_, present := result.Value()
	require.False(t, present)
}
