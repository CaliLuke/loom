package valuecontract

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func productionPredicateKind(t *testing.T, kind expr.EffectiveValidationClauseKind) string {
	t.Helper()
	switch kind {
	case expr.EffectivePatternClause:
		return "pattern"
	case expr.EffectiveFormatClause:
		return "format"
	default:
		t.Fatalf("unknown production predicate kind %d", kind)
		return ""
	}
}

func checkLoweredValidationRoundTrip(
	t *testing.T,
	extractor *independentAliasExtractor,
	validation expr.EffectiveValidation,
	want []predicateObservation,
) {
	t.Helper()
	var patterns []string
	var formats []expr.ValidationFormat
	for _, clause := range want {
		switch clause.kind {
		case "pattern":
			patterns = append(patterns, clause.value)
		case "format":
			formats = append(formats, expr.ValidationFormat(clause.value))
		default:
			t.Fatalf("unknown reference predicate kind %q", clause.kind)
		}
	}
	lowered := validation.Lowered()
	require.Nil(t, lowered.Values, "lowering must not re-author an effective enum")
	require.Empty(t, lowered.Pattern)
	require.Empty(t, lowered.Format)
	require.Equal(t, patterns, lowered.PatternClauses)
	require.Equal(t, formats, lowered.FormatClauses)

	roundTrip, err := expr.EffectiveConstraintsFor(&expr.AttributeExpr{
		Type: extractor.attribute.Type, Nullable: extractor.attribute.Nullable, Validation: lowered,
	})
	require.NoError(t, err)
	wantEnums := semanticEnumClauses(extractor, lowered.Enums())
	gotEnums := semanticEnumClauses(extractor, roundTrip.Validation().Lowered().Enums())
	require.Equal(t, wantEnums, gotEnums)
	var gotPatterns []string
	var gotFormats []expr.ValidationFormat
	for _, clause := range roundTrip.Validation().Clauses() {
		require.Equal(t, extractor.attribute.Type.Name(), clause.Provenance.Declaration,
			"lowering creates one synthetic physical declaration")
		switch productionPredicateKind(t, clause.Kind) {
		case "pattern":
			gotPatterns = append(gotPatterns, clause.Value)
		case "format":
			gotFormats = append(gotFormats, expr.ValidationFormat(clause.Value))
		}
	}
	require.Equal(t, patterns, gotPatterns)
	require.Equal(t, formats, gotFormats)
}

func TestLoweredValidationPreservesDetachedRawEnumHostShapes(t *testing.T) {
	authoredBytes := []byte("bytes")
	attribute := &expr.AttributeExpr{
		Type: expr.Bytes,
		Validation: &expr.ValidationExpr{
			Values: []any{"text", authoredBytes},
		},
	}
	constraints, err := expr.EffectiveConstraintsFor(attribute)
	require.NoError(t, err)

	authoredBytes[0] = 'X'
	lowered := constraints.Validation().Lowered()
	require.Nil(t, lowered.Values, "lowering must not re-author the effective enum")
	require.Equal(t, [][]any{{"text", []byte("bytes")}}, lowered.EnumClauses)
	require.IsType(t, "", lowered.EnumClauses[0][0])
	require.IsType(t, []byte(nil), lowered.EnumClauses[0][1])

	lowered.EnumClauses[0][1].([]byte)[0] = 'Y'
	loweredAgain := constraints.Validation().Lowered()
	require.Equal(t, [][]any{{"text", []byte("bytes")}}, loweredAgain.EnumClauses,
		"built-in raw enum carriers must be detached from caller mutation")
}

func semanticEnumClauses(extractor *independentAliasExtractor, clauses [][]any) [][]uint64 {
	result := make([][]uint64, len(clauses))
	for clauseIndex, clause := range clauses {
		result[clauseIndex] = make([]uint64, len(clause))
		for valueIndex, value := range clause {
			result[clauseIndex][valueIndex] = extractor.semantic(value).Semantic
		}
	}
	return result
}
