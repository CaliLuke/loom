package valuecontract

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

// checkReferenceNumericSources compares the production resolver against the
// proved source evaluator. Codec rows come from standalone standard-library
// operations, never from the resolver's result or chosen branch.
func checkReferenceNumericSources(t *testing.T, executable string) {
	t.Helper()
	for _, role := range []struct {
		name   string
		goRole expr.ValueRole
	}{
		{name: "authoredExample", goRole: expr.ValueRoleExample},
		{name: "enumMember", goRole: expr.ValueRoleEnum},
		{name: "defaultValue", goRole: expr.ValueRoleDefault},
	} {
		for _, constrained := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/constrained=%t", role.name, constrained), func(t *testing.T) {
				raw := float32(0.1)
				attribute := &expr.AttributeExpr{Type: expr.Float64}
				minimum := 0.1000000005
				if constrained {
					attribute.Validation = &expr.ValidationExpr{Minimum: &minimum}
				}
				context := expr.NewValueContext()
				occurrence, err := context.NewOccurrence(attribute)
				require.NoError(t, err)
				result := context.Resolve(occurrence, context.SupplyValue(expr.ValueInput{Raw: raw}), role.goRole)
				scalar, codecs := referenceNumericSource(t, raw, "binary64", 0)
				bounds := map[string]any{"minimum": nil, "maximum": nil, "exclusiveMinimum": false, "exclusiveMaximum": false}
				if constrained {
					// The evaluated DSL bound is itself a float64. Preserve its
					// exact value rather than replacing it with decimal source text.
					bounds["minimum"] = referenceBinaryDecimal(minimum)
				}
				identity := referenceIdentity{Occurrence: 1, Declaration: 1}
				request := map[string]any{
					"declarations": []any{map[string]any{
						"identity": identity, "expansionRank": 0, "enumeration": nil,
						"contract": referenceConstructor("scalar", map[string]any{
							"kind": "decimal", "rules": map[string]any{
								"sourcePrimitive": "float64", "numericFormat": "binary64", "integerFormat": "mathematical", "enumeration": nil,
								"length":  map[string]any{"minimum": nil, "maximum": nil},
								"numeric": bounds, "externalChecks": []any{},
							},
						}),
					}},
					"root": identity,
					"supplied": map[string]any{
						"source": map[string]any{"occurrence": identity, "origin": 1, "role": role.name},
						"value":  referenceConstructor("scalar", map[string]any{"value": map[string]any{"value": scalar, "primitive": "builtinFloat32"}}),
					},
					"codecs": codecs, "checks": []any{}, "projection": nil,
				}
				command := referenceConstructor("evaluate", map[string]any{"request": request})
				actual := referenceDecode[map[string]jsontext.Value](t, runReference(t, executable, []any{command})[0])
				for _, field := range []string{"literalSpellingRequests", "literalReadingRequests", "decimalRequests"} {
					require.NotContains(t, actual, field)
				}
				resolved := referenceDecode[map[string]jsontext.Value](t, actual["resolved"])
				legacy, present := result.LegacyValue()
				require.True(t, present)
				require.Equal(t, raw, legacy, "semantic normalization must not rewrite the supplied legacy snapshot")
				if constrained {
					require.Equal(t, expr.ValueInvalid, result.Outcome())
					require.Equal(t, `"invalid"`, string(resolved["error"]))
					return
				}
				require.Equal(t, expr.ValueResolved, result.Outcome())
				value, present := result.Value()
				require.True(t, present)
				normalized, present := value.Scalar()
				require.True(t, present)
				require.Equal(t, float64(0.1), normalized)
				require.NotEqual(t, float64(raw), normalized)
				ok := referenceDecode[map[string]jsontext.Value](t, resolved["ok"])
				wireValue := referenceDecode[map[string]jsontext.Value](t, ok["value"])
				scalarValue := referenceDecode[map[string]jsontext.Value](t, wireValue["scalar"])
				expected, _ := referenceNumericScalar(normalized, 0)
				expectedJSON, err := json.Marshal(expected, json.Deterministic(true))
				require.NoError(t, err)
				// Compare the exact integer coefficient through typed JSON; do
				// not use a float-decoding JSON equality helper for this assertion.
				require.Equal(t, referenceDecode[map[string]jsontext.Value](t, expectedJSON), referenceDecode[map[string]jsontext.Value](t, scalarValue["value"]))
				codecs["literalReadings"] = []any{}
				missing := referenceDecode[map[string]jsontext.Value](t, runReference(t, executable, []any{command})[0])
				require.Contains(t, missing, "literalReadingRequests")
				require.NotContains(t, missing, "resolved")
			})
		}
	}
}

func referenceNumericScalar(value any, literalOrigin uint64) (map[string]any, map[string]any) {
	var number float64
	var format string
	raw := reflect.ValueOf(value)
	switch raw.Kind() {
	case reflect.Float32:
		number, format = raw.Float(), "binary32"
	case reflect.Float64:
		number, format = raw.Float(), "binary64"
	case reflect.Int64:
		integer := jsontext.Value(strconv.FormatInt(raw.Int(), 10))
		return referenceConstructor("integer", map[string]any{"value": integer, "literalOrigin": literalOrigin}),
			map[string]any{"value": referenceDecimal{Coefficient: integer}, "format": "exact", "negativeZero": false}
	default:
		panic("referenceNumericScalar requires a supported numeric value")
	}
	decimal := referenceBinaryDecimal(number)
	negativeZero := number == 0 && math.Signbit(number)
	return referenceConstructor("decimal", map[string]any{
		"coefficient": decimal.Coefficient, "exponent": decimal.Exponent,
		"format": format, "negativeZero": negativeZero, "literalOrigin": literalOrigin,
	}), map[string]any{"value": decimal, "format": format, "negativeZero": negativeZero}
}

func referenceNumericSource(t *testing.T, raw any, format string, literalOrigin uint64) (map[string]any, map[string]any) {
	t.Helper()
	bits := 64
	if format == "binary32" {
		bits = 32
	}
	literal := fmt.Sprint(raw)
	parsed, err := strconv.ParseFloat(literal, bits)
	require.NoError(t, err)
	var normalized any = parsed
	if bits == 32 {
		normalized = float32(parsed)
	}
	scalar, identity := referenceNumericScalar(raw, literalOrigin)
	_, normalizedIdentity := referenceNumericScalar(normalized, 0)
	rawWire, err := json.Marshal(raw)
	require.NoError(t, err)
	normalizedWire, err := json.Marshal(normalized)
	require.NoError(t, err)
	spellings := []any{map[string]any{"number": identity, "text": string(rawWire)}}
	if !reflect.DeepEqual(identity, normalizedIdentity) {
		spellings = append(spellings, map[string]any{"number": normalizedIdentity, "text": string(normalizedWire)})
	}
	return scalar, map[string]any{
		"numberReadings": []any{}, "integerReadings": []any{}, "decimalReadings": []any{}, "decimalSpellings": spellings,
		"literalSpellings": []any{map[string]any{"input": scalar, "text": literal}},
		"literalReadings": []any{map[string]any{
			"format": format, "text": literal,
			"result": map[string]any{"value": referenceBinaryDecimal(parsed), "negativeZero": parsed == 0 && math.Signbit(parsed)},
		}},
	}
}
