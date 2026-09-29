package valuecontract

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	loom "github.com/CaliLuke/loom/pkg"
)

// checkReferenceRepresentations tests the actual JSONValue boundary against the
// model while retaining exact values, representation width and signed zero.
func checkReferenceRepresentations(t *testing.T, executable string) {
	t.Helper()
	narrow := float32(0.1)
	values := []any{narrow, float64(narrow), math.Copysign(0, -1), float64(0)}
	formats := []string{"binary32", "binary64", "binary64", "binary64"}
	numbers := []referenceDecimal{
		referenceBinaryDecimal(float64(narrow)), referenceBinaryDecimal(float64(narrow)),
		referenceBinaryDecimal(0), referenceBinaryDecimal(0),
	}
	schemas := []referenceDecimal{
		{Coefficient: jsontext.Value("1"), Exponent: -1},
		{Coefficient: jsontext.Value("10000000149011612"), Exponent: -17},
		{Coefficient: jsontext.Value("0")}, {Coefficient: jsontext.Value("0")},
	}
	inputs := make([]any, 0, len(values))
	spellings := make([]any, 0, len(values))
	readings := make([]any, 0, len(values))
	expected := make([]string, 0, len(values))
	for i, value := range values {
		wire, err := json.Marshal(value)
		require.NoError(t, err)
		text := string(wire)
		expected = append(expected, text)
		negativeZero := i == 2
		identity := map[string]any{"value": numbers[i], "format": formats[i], "negativeZero": negativeZero}
		spellings = append(spellings, map[string]any{"number": identity, "text": text})
		scalar := referenceConstructor("decimal", map[string]any{
			"coefficient": numbers[i].Coefficient, "exponent": numbers[i].Exponent,
			"format": formats[i], "negativeZero": negativeZero, "literalOrigin": 0,
		})
		primitive := "builtinFloat64"
		if i == 0 {
			primitive = "builtinFloat32"
		}
		inputs = append(inputs, referenceConstructor("scalar", map[string]any{"value": map[string]any{"value": scalar, "primitive": primitive}}))
		readings = append(readings, map[string]any{"text": text, "schema": schemas[i]})
	}
	identity := referenceIdentity{Occurrence: 1, Declaration: 1}
	codecs := map[string]any{"decimalSpellings": spellings, "numberReadings": readings, "integerReadings": []any{}, "decimalReadings": []any{}, "literalSpellings": []any{}, "literalReadings": []any{}}
	request := map[string]any{
		"declarations": []any{map[string]any{
			"identity": identity, "expansionRank": 0, "contract": "any", "enumeration": nil,
		}},
		"root": identity,
		"supplied": map[string]any{
			"source": map[string]any{"occurrence": identity, "origin": 1, "role": "authoredExample"},
			"value":  referenceConstructor("array", map[string]any{"items": inputs}),
		},
		"codecs": codecs, "checks": []any{},
		"projection": map[string]any{
			"root": identity, "use": "runtime",
			"targets": []any{map[string]any{
				"identity": identity, "expansionRank": 0, "target": "any",
				"enumeration": nil, "schemaEnumeration": nil,
				"schemaAllowsUnknown": true, "decoderRejectsUnknown": false,
			}},
		},
	}
	command := referenceConstructor("evaluate", map[string]any{"request": request})
	result := referenceDecode[map[string]jsontext.Value](t, runReference(t, executable, []any{command})[0])
	require.NotContains(t, result, "decimalRequests")
	require.NotContains(t, result, "numericRequests")
	var projected struct {
		Emitted struct {
			Value struct {
				Array struct {
					Items []struct {
						Number struct {
							Text string `json:"text"`
						} `json:"number"`
					} `json:"items"`
				} `json:"array"`
			} `json:"value"`
		} `json:"emitted"`
	}
	require.NoError(t, json.Unmarshal(result["projected"], &projected, json.RejectUnknownMembers(true)))
	actual := make([]string, 0, len(projected.Emitted.Value.Array.Items))
	for _, item := range projected.Emitted.Value.Array.Items {
		actual = append(actual, item.Number.Text)
	}
	require.Equal(t, expected, actual)
	wire, err := loom.JSONValueFrom(values)
	require.NoError(t, err)
	var rawValues []jsontext.Value
	require.NoError(t, json.Unmarshal(wire, &rawValues))
	require.Len(t, rawValues, len(actual))
	for i, raw := range rawValues {
		require.Equal(t, string(raw), actual[i])
	}
	// One width's row cannot satisfy the other width's codec obligation, even
	// when both inputs have exactly the same coefficient and exponent.
	codecs["decimalSpellings"] = spellings[1:]
	missing := referenceDecode[map[string]jsontext.Value](t, runReference(t, executable, []any{command})[0])
	require.Contains(t, missing, "decimalRequests")
	require.NotContains(t, missing, "resolved")
	require.NotContains(t, missing, "projected")
	requests := referenceDecode[[]struct {
		Format string `json:"format"`
	}](t, missing["decimalRequests"])
	require.Len(t, requests, 1)
	require.Equal(t, "binary32", requests[0].Format)
}
