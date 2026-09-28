package jsonkey

import (
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

type (
	namedBoolean  bool
	textBoolean   bool
	jsonBoolean   bool
	skipBoolean   bool
	appendBoolean bool
	streamBoolean bool
)

// MarshalText gives textBoolean an authored object member name.
func (value *textBoolean) MarshalText() ([]byte, error) {
	return []byte("authored"), nil
}

// UnmarshalText restores textBoolean from its authored member name.
func (value *textBoolean) UnmarshalText(data []byte) error {
	if string(data) != "authored" {
		return errors.New("expected authored key")
	}
	*value = true
	return nil
}

// MarshalJSON gives jsonBoolean an authored object member name.
func (value jsonBoolean) MarshalJSON() ([]byte, error) {
	return []byte(`"authored"`), nil
}

// UnmarshalJSON restores jsonBoolean from its authored member name.
func (value *jsonBoolean) UnmarshalJSON(data []byte) error {
	if string(data) != `"authored"` {
		return errors.New("expected authored key")
	}
	*value = true
	return nil
}

// MarshalJSON leaves the standard JSON fallback in control.
func (value skipBoolean) MarshalJSON() ([]byte, error) {
	return nil, errors.ErrUnsupported
}

// UnmarshalJSON leaves the standard JSON fallback in control.
func (value *skipBoolean) UnmarshalJSON([]byte) error {
	return errors.ErrUnsupported
}

// AppendText supplies the authored object member name.
func (value appendBoolean) AppendText(dst []byte) ([]byte, error) {
	return append(dst, "authored"...), nil
}

// UnmarshalText restores the authored key.
func (value *appendBoolean) UnmarshalText(data []byte) error {
	if string(data) != "authored" {
		return errors.New("expected authored key")
	}
	*value = true
	return nil
}

// MarshalJSONTo supplies the authored object member name.
func (value *streamBoolean) MarshalJSONTo(encoder *jsontext.Encoder) error {
	return encoder.WriteToken(jsontext.String("authored"))
}

// UnmarshalJSONFrom restores the authored key.
func (value *streamBoolean) UnmarshalJSONFrom(decoder *jsontext.Decoder) error {
	token, err := decoder.ReadToken()
	if err != nil {
		return err
	}
	if token.Kind() != jsontext.KindString || token.String() != "authored" {
		return errors.New("expected authored key")
	}
	*value = true
	return nil
}

func TestBooleanKeysRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value any
		wire  string
	}{
		{"boolean-values", map[bool]bool{false: true, true: false}, `{"false":true,"true":false}`},
		{"named-keys", map[namedBoolean]int{true: 7}, `{"true":7}`},
		{"nested-array", []map[bool][]bool{{true: {false, true}}}, `[{"true":[false,true]}]`},
		{"nested-map", map[string]map[bool]int{"team": {false: 9}}, `{"team":{"false":9}}`},
		{"pointer-text-methods", map[textBoolean]bool{true: false}, `{"authored":false}`},
		{"json-methods", map[jsonBoolean]bool{true: false}, `{"authored":false}`},
		{"text-appender", map[appendBoolean]bool{true: false}, `{"authored":false}`},
		{"stream-methods", map[streamBoolean]bool{true: false}, `{"authored":false}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			encoded, err := json.Marshal(tc.value, BooleanKeys, json.Deterministic(true))
			require.NoError(t, err)
			require.Equal(t, tc.wire, string(encoded))
			decoded := reflect.New(reflect.TypeOf(tc.value))
			require.NoError(t, json.Unmarshal(encoded, decoded.Interface(), BooleanKeys))
			require.Equal(t, tc.value, decoded.Elem().Interface())
		})
	}
}

func TestBooleanKeysLeaveOrdinaryValuesUnchanged(t *testing.T) {
	for _, value := range []any{true, false, []bool{false, true}, map[string]bool{"true": false}} {
		t.Run(reflect.TypeOf(value).String(), func(t *testing.T) {
			original, err := json.Marshal(value, json.Deterministic(true))
			require.NoError(t, err)
			extended, err := json.Marshal(value, BooleanKeys, json.Deterministic(true))
			require.NoError(t, err)
			require.Equal(t, original, extended)
			decoded := reflect.New(reflect.TypeOf(value))
			require.NoError(t, json.Unmarshal(extended, decoded.Interface(), BooleanKeys))
			require.Equal(t, value, decoded.Elem().Interface())
		})
	}
	var array []bool
	require.Error(t, json.Unmarshal([]byte(`["true"]`), &array, BooleanKeys))
}

func TestBooleanKeysRejectMalformedInput(t *testing.T) {
	for _, input := range []string{
		`{"TRUE":false}`, `{"1":false}`, `{"false":"true"}`,
		`{"true":false,"true":true}`, `{`, `{"true":`, `not-json`,
	} {
		t.Run(input, func(t *testing.T) {
			var decoded map[bool]bool
			require.Error(t, json.Unmarshal([]byte(input), &decoded, BooleanKeys))
		})
	}
	_, err := json.Marshal(map[any]any{true: false, "true": true}, BooleanKeys, json.Deterministic(true))
	require.Error(t, err, "distinct Go keys must not silently collapse to one JSON member")
}

func TestBooleanKeysPreserveCustomFallback(t *testing.T) {
	value := map[skipBoolean]bool{true: false}
	_, original := json.Marshal(value)
	_, extended := json.Marshal(value, BooleanKeys)
	require.Error(t, original)
	require.EqualError(t, extended, original.Error())
	var decoded map[skipBoolean]bool
	original = json.Unmarshal([]byte(`{"true":false}`), &decoded)
	extended = json.Unmarshal([]byte(`{"true":false}`), &decoded, BooleanKeys)
	require.Error(t, original)
	require.EqualError(t, extended, original.Error())
}
