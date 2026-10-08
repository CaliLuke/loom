package ir

import (
	"encoding/json/v2"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"

	"github.com/CaliLuke/loom/expr"
)

func TestMapDefaultScalarKeys(t *testing.T) {
	type namedInt int64
	type namedString string
	for _, tc := range []struct {
		name   string
		key    any
		member string
	}{
		{"string", "key", "key"},
		{"bool", true, "true"},
		{"int", int(-7), "-7"},
		{"int8", int8(-128), "-128"},
		{"int16", int16(-32768), "-32768"},
		{"int32", int32(math.MinInt32), "-2147483648"},
		{"int64", int64(math.MinInt64), "-9223372036854775808"},
		{"uint", uint(7), "7"},
		{"uint8", uint8(255), "255"},
		{"uint16", uint16(65535), "65535"},
		{"uint32", uint32(math.MaxUint32), "4294967295"},
		{"uint64", uint64(math.MaxUint64), "18446744073709551615"},
		{"float32", float32(1.2), "1.2"},
		{"float64", float64(1e21), "1e+21"},
		{"negative-zero", math.Copysign(0, -1), "-0"},
		{"named-int", namedInt(7), "7"},
		{"named-string", namedString("key"), "key"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, nested := range []bool{false, true} {
				var value any = map[any]any{tc.key: "value"}
				var expected any = map[string]any{tc.member: "value"}
				if nested {
					value = map[string]any{"items": []any{value}}
					expected = map[string]any{"items": []any{expected}}
				}
				attr := &expr.AttributeExpr{Type: expr.Any, DefaultValue: value}
				var schema *Schema
				require.NotPanics(t, func() {
					schema = NewAnalyzer(expr.NewRandom("map-defaults"), false).AnalyzeSchema(attr)
				})
				require.Equal(t, expected, schema.DefaultValue)
				actualJSON, err := json.Marshal(schema.DefaultValue, json.Deterministic(true))
				require.NoError(t, err)
				expectedJSON, err := json.Marshal(expected, json.Deterministic(true))
				require.NoError(t, err)
				require.Equal(t, string(expectedJSON), string(actualJSON))
				actualYAML, err := yaml.Marshal(schema.DefaultValue)
				require.NoError(t, err)
				expectedYAML, err := yaml.Marshal(expected)
				require.NoError(t, err)
				require.Equal(t, string(expectedYAML), string(actualYAML))
			}
		})
	}
}

func TestMapDefaultsPreserveNestedTypedValues(t *testing.T) {
	value := map[uint64][]map[int32][]byte{
		math.MaxUint64: {{-3: {0, 255}}},
	}
	attr := &expr.AttributeExpr{Type: expr.Any, DefaultValue: value}
	schema := NewAnalyzer(expr.NewRandom("nested-defaults"), false).AnalyzeSchema(attr)
	expected := map[string]any{
		"18446744073709551615": []any{map[string]any{"-3": []byte{0, 255}}},
	}
	require.Equal(t, expected, schema.DefaultValue)
	encoded, err := json.Marshal(schema.DefaultValue, json.Deterministic(true))
	require.NoError(t, err)
	require.Equal(t, `{"18446744073709551615":[{"-3":"AP8="}]}`, string(encoded))
	require.Equal(t, []byte{0, 255}, value[math.MaxUint64][0][-3])
}
