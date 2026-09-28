package ir

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

type (
	defaultTextKey      int
	defaultAppendKey    int
	defaultJSONKey      int
	defaultJSONToKey    int
	defaultPointerKey   int
	defaultEncodedMap   map[int]int
	defaultEncodedSlice []int
)

// MarshalText provides this key's wire name.
func (defaultTextKey) MarshalText() ([]byte, error) {
	return []byte("text-key"), nil
}

// AppendText appends this key's wire name.
func (defaultAppendKey) AppendText(buffer []byte) ([]byte, error) {
	return append(buffer, "append-key"...), nil
}

// MarshalJSON provides this key's JSON representation.
func (defaultJSONKey) MarshalJSON() ([]byte, error) {
	return []byte(`"json-key"`), nil
}

// MarshalJSONTo writes this key's JSON representation.
func (defaultJSONToKey) MarshalJSONTo(encoder *jsontext.Encoder) error {
	return encoder.WriteToken(jsontext.String("json-to-key"))
}

// MarshalText provides this key's wire name using a pointer receiver.
func (*defaultPointerKey) MarshalText() ([]byte, error) {
	return []byte("pointer-key"), nil
}

// MarshalJSON replaces the underlying map with its wire value.
func (defaultEncodedMap) MarshalJSON() ([]byte, error) {
	return []byte(`"encoded-map"`), nil
}

// MarshalJSON replaces the underlying slice with its wire value.
func (defaultEncodedSlice) MarshalJSON() ([]byte, error) {
	return []byte(`"encoded-slice"`), nil
}

func TestMapDefaultsPreserveCustomEncoders(t *testing.T) {
	for _, tc := range []struct {
		name     string
		value    any
		expected string
	}{
		{"text-key", map[defaultTextKey]string{1: "value"}, `{"text-key":"value"}`},
		{"append-key", map[defaultAppendKey]string{1: "value"}, `{"append-key":"value"}`},
		{"json-key", map[defaultJSONKey]string{1: "value"}, `{"json-key":"value"}`},
		{"json-to-key", map[defaultJSONToKey]string{1: "value"}, `{"json-to-key":"value"}`},
		{"pointer-key", map[defaultPointerKey]string{1: "value"}, `{"pointer-key":"value"}`},
		{"map", defaultEncodedMap{1: 2}, `"encoded-map"`},
		{"nested-map", map[any]any{1: defaultEncodedMap{1: 2}}, `{"1":"encoded-map"}`},
		{"nested-slice", map[any]any{1: defaultEncodedSlice{2}}, `{"1":"encoded-slice"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			attribute := &expr.AttributeExpr{Type: expr.Any, DefaultValue: tc.value}
			schema := NewAnalyzer(expr.NewRandom("custom-defaults"), false).AnalyzeSchema(attribute)
			encoded, err := json.Marshal(schema.DefaultValue, json.Deterministic(true))
			require.NoError(t, err)
			require.Equal(t, tc.expected, string(encoded))
		})
	}
}
