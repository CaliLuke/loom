package expr

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAttributeExampleUsesEffectiveNamedConstraints(t *testing.T) {
	const validUUID = "550e8400-e29b-41d4-a716-446655440000"
	t.Run("enum and format", func(t *testing.T) {
		base := namedScalar("ExampleFormatBase", String, &ValidationExpr{
			Values: []any{"not-a-uuid", validUUID},
		})
		derived := namedScalar("ExampleFormatDerived", base, &ValidationExpr{Format: FormatUUID})

		require.Equal(t, validUUID, (&AttributeExpr{Type: derived}).Example(NewRandom("effective-format")))
	})
	t.Run("present empty enum", func(t *testing.T) {
		base := namedScalar("ExampleEmptyBase", String, &ValidationExpr{Values: []any{"not-a-uuid"}})
		derived := namedScalar("ExampleEmptyDerived", base, &ValidationExpr{Format: FormatUUID})

		require.Nil(t, (&AttributeExpr{Type: derived}).Example(NewRandom("effective-empty")))
	})
	t.Run("numeric bounds", func(t *testing.T) {
		minimum, maximum := 100.0, 200.0
		base := namedScalar("ExampleNumberBase", Float64, &ValidationExpr{Minimum: &minimum})
		derived := namedScalar("ExampleNumberDerived", base, &ValidationExpr{Maximum: &maximum})

		example, ok := (&AttributeExpr{Type: derived}).Example(NewRandom("reviewer")).(float64)
		require.True(t, ok)
		require.GreaterOrEqual(t, example, minimum)
		require.LessOrEqual(t, example, maximum)
	})
	t.Run("patterns", func(t *testing.T) {
		base := namedScalar("ExamplePatternBase", String, &ValidationExpr{Pattern: "^a"})
		derived := namedScalar("ExamplePatternDerived", base, &ValidationExpr{Pattern: "b$"})

		require.Nil(t, (&AttributeExpr{Type: derived}).Example(NewRandom("effective-pattern")))
	})
	t.Run("object enum and required", func(t *testing.T) {
		attribute := requiredObjectEnumAttribute("AttributeExample", false)

		require.Equal(t, map[string]any{"required": "ok"}, attribute.Example(NewRandom("effective-required")))
	})
	t.Run("physical enum clause and required", func(t *testing.T) {
		attribute := &AttributeExpr{
			Type: &Object{{Name: "required", Attribute: &AttributeExpr{Type: String}}},
			Validation: &ValidationExpr{
				EnumClauses: [][]any{{map[string]any{}, map[string]any{"required": "ok"}}},
				Required:    []string{"required"},
			},
		}

		require.Equal(t, map[string]any{"required": "ok"}, attribute.Example(NewRandom("physical-required")))
	})
}

func TestValueSynthesisUsesFilteredEffectiveEnumCandidates(t *testing.T) {
	for _, test := range []struct {
		name      string
		attribute *AttributeExpr
		outcome   ValueOutcome
		value     any
	}{
		{
			name:      "required object member",
			attribute: requiredObjectEnumAttribute("SynthesisExample", false),
			outcome:   ValueResolved,
			value:     map[string]any{"required": "ok"},
		},
		{
			name:      "present empty object enum",
			attribute: requiredObjectEnumAttribute("EmptySynthesisExample", true),
			outcome:   ValueSuppressed,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			context := NewValueContext()
			occurrence, err := context.NewOccurrence(test.attribute)
			require.NoError(t, err)
			result := context.Synthesize(
				context.SelectExample(occurrence, ExamplePolicy{Reachable: true}),
				NewRandom("effective-object-enum"),
			)
			require.Equal(t, test.outcome, result.Outcome(), "%v", result.Diagnostics())
			if test.outcome == ValueSuppressed {
				_, present := result.Value()
				require.False(t, present)
				return
			}
			legacy, present := result.LegacyValue()
			require.True(t, present)
			require.Equal(t, test.value, legacy)
		})
	}
}

func TestNumericAliasExamplesRetainPrimitiveRepresentation(t *testing.T) {
	for _, primitive := range []Primitive{Int, Int32, Int64, UInt, UInt32, UInt64, Float32, Float64} {
		for _, tc := range []struct {
			name                      string
			base, derived, occurrence *ValidationExpr
			minimum, maximum          float64
		}{
			{"minimum", &ValidationExpr{Minimum: new(1.0)}, nil, nil, 1, 1e30},
			{"inherited bounds", &ValidationExpr{Minimum: new(1.0)}, &ValidationExpr{Maximum: new(8.0)}, &ValidationExpr{Minimum: new(3.0), Maximum: new(6.0)}, 3, 6},
			{"exact", &ValidationExpr{Minimum: new(4.0)}, &ValidationExpr{Maximum: new(4.0)}, nil, 4, 4},
			{"exclusive", &ValidationExpr{ExclusiveMinimum: new(1.0)}, &ValidationExpr{ExclusiveMaximum: new(3.0)}, nil, 1, 3},
		} {
			t.Run(primitive.Name()+"/"+tc.name, func(t *testing.T) {
				base := namedScalar("NumericBase", primitive, tc.base)
				derived := namedScalar("NumericDerived", base, tc.derived)
				attribute := &AttributeExpr{Type: derived, Validation: tc.occurrence}
				example := attribute.Example(NewRandom("numeric-alias"))
				require.IsType(t, primitive.Example(NewRandom("type")), example)
				value := reflect.ValueOf(example).Convert(reflect.TypeFor[float64]()).Float()
				if tc.name == "exclusive" {
					require.Greater(t, value, tc.minimum)
					require.Less(t, value, tc.maximum)
				} else {
					require.GreaterOrEqual(t, value, tc.minimum)
					require.LessOrEqual(t, value, tc.maximum)
				}
				require.Equal(t, example, attribute.Example(NewRandom("numeric-alias")))
				require.Same(t, derived, attribute.Type, "synthesis must retain the authored alias")
				require.Same(t, base, derived.Type)
			})
		}
	}
}

func TestNumericAliasArrayExample(t *testing.T) {
	count := namedScalar("Count", Int, &ValidationExpr{Minimum: new(1.0)})
	counts := namedScalar("Counts", &Array{ElemType: &AttributeExpr{Type: count}}, nil)
	attribute := &AttributeExpr{Type: counts}
	generator := &ExampleGenerator{Randomizer: NewDeterministicRandomizer()}
	require.Equal(t, []int{2}, attribute.Example(generator))
}

func requiredObjectEnumAttribute(prefix string, empty bool) *AttributeExpr {
	values := []any{map[string]any{}, map[string]any{"required": "ok"}}
	if empty {
		values = values[:1]
	}
	base := &UserTypeExpr{
		TypeName: prefix + "Base",
		UID:      prefix + "Base",
		AttributeExpr: &AttributeExpr{
			Type: &Object{{Name: "required", Attribute: &AttributeExpr{Type: String}}},
			Validation: &ValidationExpr{
				Values: values,
			},
		},
	}
	derived := &UserTypeExpr{
		TypeName: prefix + "Derived",
		UID:      prefix + "Derived",
		AttributeExpr: &AttributeExpr{
			Type:       base,
			Validation: &ValidationExpr{Required: []string{"required"}},
		},
	}
	return &AttributeExpr{Type: derived}
}
