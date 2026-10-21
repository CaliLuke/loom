package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValueSynthesisScopeIsolation(t *testing.T) {
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(&AttributeExpr{Type: String})
	require.NoError(t, err)
	for _, scope := range []string{"service", "documentation", "service"} {
		selection := context.SelectExample(occurrence, ExamplePolicy{Reachable: true, SynthesisScope: scope})
		result := context.Synthesize(selection, NewRandom(scope))
		raw, present := result.LegacyValue()
		require.True(t, present)
		require.Equal(t, (&AttributeExpr{Type: String}).Example(NewRandom(scope)), raw)
	}
	suppressed := context.SelectExample(occurrence, ExamplePolicy{Reachable: true, SuppressGenerated: true, SynthesisScope: "service"})
	require.Equal(t, ValueSuppressed, context.Synthesize(suppressed, NewRandom("service")).Outcome())
}
