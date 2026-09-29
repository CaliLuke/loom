package valuecontract

import (
	"encoding/base64"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"math"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestLeanConformance is the mandatory external-reference entry point used by
// the Make/CI gate. Ordinary Go tests retain the adapter and rejection controls.
func TestLeanConformance(t *testing.T) {
	if os.Getenv("LOOM_VALUE_CONFORMANCE") != "1" {
		t.Skip("set LOOM_VALUE_CONFORMANCE=1 and LOOM_LEAN_REFERENCE to run the Lean reference")
	}
	report := newConformanceReport(t)
	executable := referenceExecutable(t)
	report.run(t, "actual-codec-boundaries", "codec-boundary", executable, checkReferenceCodecs)
	report.run(t, "authored-source-presence", "source-selection", executable, checkReferenceSourcePresence)
	report.run(t, "numeric-literal-origins", "codec-boundary", executable, checkReferenceLiteralOrigins)
	report.run(t, "declared-numeric-precision", "codec-boundary", executable, checkReferenceNumericSources)
	report.run(t, "actual-numeric-representations", "codec-boundary", executable, checkReferenceRepresentations)
	report.run(t, "codec-input-completeness", "negative-control", executable, checkReferenceCodecInputs)
	report.run(t, "production-selection", "source-selection", executable, checkProductionSelectionConformance)
	report.run(t, "production-resolution", "resolution", executable, checkProductionSourceConformance)
	report.run(t, "production-synthesis", "resolution", executable, checkProductionSynthesisConformance)
	report.run(t, "production-projection", "projection", executable, checkProductionProjectionConformance)
	report.run(t, "production-projection-corruption", "negative-control", executable, checkProductionProjectionCorruption)
	report.run(t, "production-numeric-decoding", "codec-boundary", executable, checkProductionTargetNumericConformance)
	report.run(t, "production-policy-rejection", "negative-control", executable, checkProductionTargetPolicyRejection)
	report.run(t, "byte-alias-lengths", "projection", executable, checkByteAliasLengthConformance)
	report.run(t, "byte-schema-bounds", "projection", executable, checkByteSchemaConformance)
}

func checkReferenceCodecInputs(t *testing.T, executable string) {
	t.Helper()
	checkReferenceSourceCodecClosure(t, executable)
	data, err := os.ReadFile("testdata/reference_codec_boundaries.json")
	require.NoError(t, err)
	var cases []struct {
		Name         string         `json:"name"`
		Request      jsontext.Value `json:"request"`
		Codecs       jsontext.Value `json:"codecs"`
		RequestField string         `json:"requestField"`
	}
	require.NoError(t, json.Unmarshal(data, &cases))
	require.Len(t, cases, 2)
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			request := referenceDecode[map[string]jsontext.Value](t, tc.Request)
			missing := runReference(t, executable, []any{request["command"]})[0]
			fields := referenceDecode[map[string]jsontext.Value](t, missing)
			require.Contains(t, fields, tc.RequestField)
			require.NotEmpty(t, referenceDecode[[]jsontext.Value](t, fields[tc.RequestField]))
			require.NotContains(t, fields, "projected", "missing codec input must not become a semantic conclusion")
			if tc.RequestField == "decimalRequests" {
				require.NotContains(t, fields, "resolved", "key spelling is needed before source collision checking")
			}
			command := referenceDecode[map[string]jsontext.Value](t, request["command"])
			evaluation := referenceDecode[map[string]jsontext.Value](t, command["evaluate"])
			body := referenceDecode[map[string]jsontext.Value](t, evaluation["request"])
			body["codecs"] = tc.Codecs
			completeCommand := referenceConstructor("evaluate", map[string]any{"request": body})
			complete := referenceDecode[map[string]jsontext.Value](t, runReference(t, executable, []any{completeCommand})[0])
			require.NotContains(t, complete, "decimalRequests")
			require.NotContains(t, complete, "numericRequests")
			if tc.RequestField == "decimalRequests" {
				resolved := referenceDecode[map[string]jsontext.Value](t, complete["resolved"])
				require.Equal(t, `"invalid"`, string(resolved["error"]), "Float32 and string keys collide under the actual codec")
			} else {
				projected := referenceDecode[map[string]jsontext.Value](t, complete["projected"])
				require.Contains(t, projected, "emitted", "overflowing competing integer-key decoder must reject")
			}
		})
	}
}

func checkReferenceCodecs(t *testing.T, executable string) {
	t.Helper()
	byteInputs := make([][]byte, 0, 261)
	byteInputs = append(byteInputs, nil, []byte{}, []byte("hi"), []byte{0, 0xff, 0x80}, []byte("plain text"))
	for value := range 256 {
		byteInputs = append(byteInputs, []byte{byte(value)})
	}
	bytesAsNumbers := make([][]uint16, len(byteInputs))
	for i, input := range byteInputs {
		bytesAsNumbers[i] = make([]uint16, len(input))
		for j, value := range input {
			bytesAsNumbers[i][j] = uint16(value)
		}
	}
	texts := []string{"", "aGk=", "aGl=", "/w==", "//==", "aGk=\r\n", "aG\nk=", "aGk", "aGk===", "aGk= ", "aGk=\x00", "aGk=aGk=", "1", "1.0", "1e0", "+1", "01", "-0", " 1", "1 ", "true", "1e-7"}
	decimals := []referenceDecimal{{Coefficient: jsontext.Value("0")}, {Coefficient: jsontext.Value("1")}, {Coefficient: jsontext.Value("-1")}, {Coefficient: jsontext.Value("100"), Exponent: -2}, {Coefficient: jsontext.Value("314"), Exponent: -2}, {Coefficient: jsontext.Value("1"), Exponent: -6}, {Coefficient: jsontext.Value("1"), Exponent: -7}, {Coefficient: jsontext.Value("1"), Exponent: 20}, {Coefficient: jsontext.Value("1"), Exponent: 21}}
	result := runReference(t, executable, []any{referenceConstructor("codecs", map[string]any{
		"bytes": bytesAsNumbers, "texts": texts, "decimals": decimals,
	})})[0]
	var decoded struct {
		EncodedBytes    []string         `json:"encodedBytes"`
		DecodedBytes    []*[]uint16      `json:"decodedBytes"`
		SchemaNumbers   []jsontext.Value `json:"schemaNumbers"`
		IntegerNumbers  []jsontext.Value `json:"integerNumbers"`
		EncodedDecimals []string         `json:"encodedDecimals"`
	}
	require.NoError(t, json.Unmarshal(result, &decoded, json.RejectUnknownMembers(true)))
	require.Len(t, decoded.EncodedBytes, len(byteInputs))
	for i, input := range byteInputs {
		require.Equal(t, base64.StdEncoding.EncodeToString(input), decoded.EncodedBytes[i])
	}
	require.Len(t, decoded.DecodedBytes, len(texts))
	require.Len(t, decoded.IntegerNumbers, len(texts))
	for i, text := range texts {
		actual, err := base64.StdEncoding.DecodeString(text)
		if err != nil {
			require.Nil(t, decoded.DecodedBytes[i], "decoder must reject %q", text)
		} else {
			require.NotNil(t, decoded.DecodedBytes[i], "decoder must accept %q", text)
			expected := make([]uint16, len(actual))
			for j, value := range actual {
				expected[j] = uint16(value)
			}
			require.Equal(t, expected, *decoded.DecodedBytes[i], "decoded bytes for %q", text)
		}
		// Model token parsing excludes surrounding whitespace; JSON document
		// whitespace is handled by the separate protocol/materialization layer.
		if text == " 1" || text == "1 " {
			continue
		}
		var integer int64
		err = json.Unmarshal([]byte(text), &integer)
		if err != nil {
			require.Equal(t, "null", string(decoded.IntegerNumbers[i]), "integer decoder for %q", text)
		} else {
			require.Equal(t, integer, referenceDecode[int64](t, decoded.IntegerNumbers[i]), "integer decoder for %q", text)
		}
	}
	require.Len(t, decoded.EncodedDecimals, len(decimals))
	for i, decimal := range decimals {
		number := referenceDecode[float64](t, decimal.Coefficient) * math.Pow10(int(decimal.Exponent))
		text, err := json.Marshal(number)
		require.NoError(t, err)
		require.Equal(t, string(text), decoded.EncodedDecimals[i])
	}
}

func checkReferenceSourcePresence(t *testing.T, executable string) {
	t.Helper()
	source := map[string]any{"occurrence": referenceIdentity{Occurrence: 5, Declaration: 7}, "origin": 11, "role": "defaultValue"}
	commands := []any{
		referenceConstructor("selectContract", map[string]any{"reachable": true, "supplied": nil}),
		referenceConstructor("selectContract", map[string]any{"reachable": true, "supplied": map[string]any{"source": source, "value": "null"}}),
		referenceConstructor("selectContract", map[string]any{"reachable": false, "supplied": map[string]any{"source": source, "value": "null"}}),
	}
	results := runReference(t, executable, commands)
	require.Equal(t, `"absent"`, string(results[0]))
	require.Equal(t, `"excluded"`, string(results[2]))
	selected := referenceDecode[map[string]jsontext.Value](t, results[1])
	require.Contains(t, selected, "selected")
	fields := referenceDecode[map[string]jsontext.Value](t, selected["selected"])
	supplied := referenceDecode[map[string]jsontext.Value](t, fields["supplied"])
	require.Equal(t, `"null"`, string(supplied["value"]))
	provenance := referenceDecode[map[string]jsontext.Value](t, supplied["source"])
	require.Equal(t, `"defaultValue"`, string(provenance["role"]))
	require.Equal(t, uint64(11), referenceDecode[uint64](t, provenance["origin"]))
}
