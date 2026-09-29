package valuecontract

import (
	"encoding/json/jsontext"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

// checkReferenceSourceCodecClosure checks requests before any semantic result:
// admitted coercions must have evidence, rejected ones must not run callbacks,
// and prepared enum values must not be treated as new authored inputs.
func checkReferenceSourceCodecClosure(t *testing.T, executable string) {
	t.Helper()
	t.Run("rejected Stringer callback", TestProductionRejectedStringerNotInvoked)
	for _, tc := range []struct {
		name         string
		missing      string
		requestField string
	}{
		{"admitted literal spelling", "literalSpellings", "literalSpellingRequests"},
		{"admitted literal reading", "literalReadings", "literalReadingRequests"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := sourceCodecRequest(t, &expr.AttributeExpr{Type: expr.Float64}, int(1))
			codecs := request["codecs"].(map[string]any)
			complete := codecs[tc.missing]
			codecs[tc.missing] = []any{}
			missing := evaluateSourceCodecRequest(t, executable, request)
			require.NotEmpty(t, missing[tc.requestField])
			require.NotContains(t, missing, "resolved")
			codecs[tc.missing] = complete
			accepted := evaluateSourceCodecRequest(t, executable, request)
			require.Contains(t, accepted, "resolved")
			require.Contains(t, referenceDecode[map[string]jsontext.Value](t, accepted["resolved"]), "ok")
		})
	}
	t.Run("admitted map key needs literal evidence", func(t *testing.T) {
		attribute := &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.Float64}, ElemType: &expr.AttributeExpr{Type: expr.String}}}
		request := sourceCodecRequest(t, attribute, map[int]string{1: "value"})
		request["codecs"].(map[string]any)["literalSpellings"] = []any{}
		missing := evaluateSourceCodecRequest(t, executable, request)
		require.NotEmpty(t, missing["literalSpellingRequests"])
		require.NotContains(t, missing, "resolved")
	})
	t.Run("raw Any still needs numeric wire spelling", func(t *testing.T) {
		request := sourceCodecRequest(t, &expr.AttributeExpr{Type: expr.Any}, productionStringByte(104))
		codecs := request["codecs"].(map[string]any)
		complete := codecs["decimalSpellings"]
		codecs["decimalSpellings"] = []any{}
		missing := evaluateSourceCodecRequest(t, executable, request)
		require.NotEmpty(t, missing["decimalRequests"])
		require.NotContains(t, missing, "resolved")
		codecs["decimalSpellings"] = complete
		accepted := evaluateSourceCodecRequest(t, executable, request)
		require.Contains(t, referenceDecode[map[string]jsontext.Value](t, accepted["resolved"]), "ok")
	})
	t.Run("prepared enum needs no literal callbacks", func(t *testing.T) {
		request := sourceCodecRequest(t, &expr.AttributeExpr{Type: expr.Float64, Validation: &expr.ValidationExpr{Values: []any{float64(2)}}}, float64(1))
		codecs := request["codecs"].(map[string]any)
		rows := codecs["literalSpellings"].([]any)
		kept := make([]any, 0, len(rows))
		for _, raw := range rows {
			row := raw.(map[string]any)
			if row["text"] == "1" {
				kept = append(kept, raw)
			}
		}
		codecs["literalSpellings"] = kept
		result := evaluateSourceCodecRequest(t, executable, request)
		require.Contains(t, result, "resolved")
		require.Equal(t, `"invalid"`, string(referenceDecode[map[string]jsontext.Value](t, result["resolved"])["error"]))
	})
}

func sourceCodecRequest(t *testing.T, attribute *expr.AttributeExpr, raw any) map[string]any {
	t.Helper()
	occurrence, err := expr.NewValueContext().NewOccurrence(attribute)
	require.NoError(t, err)
	input := newProductionInput(t)
	graph := newProductionGraph(t, input)
	root := graph.capture(attribute, occurrence)
	graph.ranks()
	supplied := input.encode(raw)
	return map[string]any{
		"declarations": graph.declarations, "root": root,
		"supplied": map[string]any{"source": map[string]any{"occurrence": root, "origin": 1, "role": "authoredExample"}, "value": supplied},
		"codecs":   input.codecs(), "checks": []any{}, "projection": nil,
	}
}

func evaluateSourceCodecRequest(t *testing.T, executable string, request map[string]any) map[string]jsontext.Value {
	t.Helper()
	return referenceDecode[map[string]jsontext.Value](t, runReference(t, executable, []any{referenceConstructor("evaluate", map[string]any{"request": request})})[0])
}
