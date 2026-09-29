package valuecontract

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/testprocess"
)

type (
	formattedReferenceFloat   float64
	formattedReferenceInteger int64
)

func (formattedReferenceFloat) String() string {
	return "2"
}

func (formattedReferenceInteger) String() string {
	return "3"
}

// checkReferenceLiteralOrigins retains the reviewer-discovered same-value,
// different-literal counterexample at the abstract normalization boundary.
// Concrete source admission rejects named host scalars before normalization;
// both that rejection and the explicitly abstract counterexample are retained.
func checkReferenceLiteralOrigins(t *testing.T, executable string) {
	t.Helper()
	raw := []any{float64(1), formattedReferenceFloat(1), int64(1), formattedReferenceInteger(1)}
	origins := []uint64{0, 1, 0, 2}
	attribute := &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.Float64}}}
	context := expr.NewValueContext()
	occurrence, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	result := context.Resolve(occurrence, context.SupplyValue(expr.ValueInput{Raw: raw}), expr.ValueRoleExample)
	require.Equal(t, expr.ValueInvalid, result.Outcome())
	_, present := result.Value()
	require.False(t, present)
	legacy, present := result.LegacyValue()
	require.True(t, present)
	require.Equal(t, raw, legacy)
	// These results describe normalization after abstract admission, not a
	// promise that concrete named Go scalars can enter a Float64 declaration.
	abstractNormalized := []any{float64(1), float64(2), float64(1), float64(3)}
	primitives := []any{"builtinFloat64", referenceConstructor("definedScalar", map[string]any{"kind": "decimal"}), "builtinInt64", referenceConstructor("definedScalar", map[string]any{"kind": "integer"})}
	inputs := make([]any, 0, len(raw))
	codecs := map[string]any{
		"numberReadings": []any{}, "integerReadings": []any{}, "decimalReadings": []any{}, "decimalSpellings": []any{},
		"literalSpellings": []any{}, "literalReadings": []any{},
	}
	for i, input := range raw {
		scalar, rows := referenceNumericSource(t, input, "binary64", origins[i])
		inputs = append(inputs, referenceConstructor("scalar", map[string]any{"value": map[string]any{"value": scalar, "primitive": primitives[i]}}))
		for name, values := range rows {
			for _, value := range values.([]any) {
				encoded, err := json.Marshal(value, json.Deterministic(true))
				require.NoError(t, err)
				duplicate := false
				for _, existing := range codecs[name].([]any) {
					prior, err := json.Marshal(existing, json.Deterministic(true))
					require.NoError(t, err)
					if bytes.Equal(prior, encoded) {
						duplicate = true
					}
				}
				if !duplicate {
					codecs[name] = append(codecs[name].([]any), value)
				}
			}
		}
	}
	root := referenceIdentity{Occurrence: 1, Declaration: 1}
	child := referenceIdentity{Occurrence: 1, Declaration: 2}
	request := map[string]any{
		"declarations": []any{
			map[string]any{
				"identity": root, "expansionRank": 0, "enumeration": nil,
				"contract": referenceConstructor("array", map[string]any{
					"child": child, "length": map[string]any{"minimum": nil, "maximum": nil},
				}),
			},
			map[string]any{
				"identity": child, "expansionRank": 0, "enumeration": nil,
				"contract": referenceConstructor("scalar", map[string]any{
					"kind": "decimal", "rules": map[string]any{
						"sourcePrimitive": "float64", "numericFormat": "binary64", "integerFormat": "mathematical", "enumeration": nil,
						"length":         map[string]any{"minimum": nil, "maximum": nil},
						"numeric":        map[string]any{"minimum": nil, "maximum": nil, "exclusiveMinimum": false, "exclusiveMaximum": false},
						"externalChecks": []any{},
					},
				}),
			},
		},
		"root": root,
		"supplied": map[string]any{
			"source": map[string]any{"occurrence": root, "origin": 1, "role": "authoredExample"},
			"value":  referenceConstructor("array", map[string]any{"items": inputs}),
		},
		"codecs": codecs, "checks": []any{}, "projection": nil,
	}
	command := referenceConstructor("evaluate", map[string]any{"request": request})
	concrete := referenceDecode[map[string]jsontext.Value](t, runReference(t, executable, []any{command})[0])
	concreteResult := referenceDecode[map[string]jsontext.Value](t, concrete["resolved"])
	require.Equal(t, `"invalid"`, string(concreteResult["error"]))
	require.NotContains(t, concreteResult, "ok")
	// Explicitly switch only the model control to abstract kind admission.
	declaration := request["declarations"].([]any)[1].(map[string]any)
	contract := declaration["contract"].(map[string]any)["scalar"].(map[string]any)
	contract["rules"].(map[string]any)["sourcePrimitive"] = nil
	response := referenceDecode[map[string]jsontext.Value](t, runReference(t, executable, []any{command})[0])
	resolved := referenceDecode[map[string]jsontext.Value](t, response["resolved"])
	ok := referenceDecode[map[string]jsontext.Value](t, resolved["ok"])
	array := referenceDecode[map[string]jsontext.Value](t, ok["value"])
	items := referenceDecode[map[string][]jsontext.Value](t, array["array"])["items"]
	require.Len(t, items, len(abstractNormalized))
	for i, item := range items {
		fields := referenceDecode[map[string]jsontext.Value](t, item)
		scalar := referenceDecode[map[string]jsontext.Value](t, fields["scalar"])
		expected, _ := referenceNumericScalar(abstractNormalized[i], 0)
		encoded, err := json.Marshal(expected, json.Deterministic(true))
		require.NoError(t, err)
		require.Equal(t, referenceDecode[map[string]jsontext.Value](t, encoded), referenceDecode[map[string]jsontext.Value](t, scalar["value"]))
	}
	spellings := codecs["literalSpellings"].([]any)
	codecs["literalSpellings"] = append([]any{spellings[0]}, spellings[2:]...)
	missing := referenceDecode[map[string]jsontext.Value](t, runReference(t, executable, []any{command})[0])
	require.Contains(t, missing, "literalSpellingRequests")
	require.NotContains(t, missing, "resolved")
	codecs["literalSpellings"] = append(spellings, spellings[0])
	requireReferenceProtocolRejection(t, executable, command, "duplicate numeric literal spelling")
}

func requireReferenceProtocolRejection(t *testing.T, executable string, command any, diagnostic string) {
	t.Helper()
	request, err := json.Marshal(referenceRequest{Version: 1, Command: command}, json.Deterministic(true))
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	cmd := testprocess.CommandContext(ctx, executable)
	cmd.Stdin = bytes.NewReader(append(request, '\n'))
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	require.Error(t, cmd.Run())
	require.Empty(t, stdout.String(), "protocol rejection must not emit a semantic outcome")
	require.Contains(t, stderr.String(), diagnostic)
}
