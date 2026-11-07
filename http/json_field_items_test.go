package http

import (
	"encoding/json/jsontext"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSONFieldItemsRoundTrip(t *testing.T) {
	values := url.Values{}
	present, err := EncodeJSONField(values, "value", []string{"a,b", "", "λ"}, FieldItems)
	require.NoError(t, err)
	require.True(t, present)
	require.Equal(t, []string{`"a,b"`, `""`, `"λ"`}, values["value"])
	var decoded []string
	present, err = DecodeJSONField(values, "value", &decoded, false, FieldItems)
	require.NoError(t, err)
	require.True(t, present)
	require.Equal(t, []string{"a,b", "", "λ"}, decoded)
	pairs, err := ParseRawPairs(values.Encode())
	require.NoError(t, err)
	projected, matched, err := FormJSONValue(pairs, "value", FieldItems)
	require.NoError(t, err)
	require.True(t, matched)
	require.JSONEq(t, `["a,b","","λ"]`, string(projected))
}

func TestJSONFieldItemDocuments(t *testing.T) {
	for _, tc := range []struct {
		name   string
		values []string
		want   string
		valid  bool
	}{
		{"empty JSON", []string{""}, "", false},
		{"whitespace", []string{" \t\n"}, "", false},
		{"malformed", []string{"}"}, "", false},
		{"truncated", []string{"["}, "", false},
		{"trailing", []string{"1 2"}, "", false},
		{"invalid later item", []string{"1", "{"}, "", false},
		{"duplicate keys", []string{`{"x":1,"x":2}`}, "", false},
		{"null item", []string{"null"}, `[null]`, true},
		{"empty string item", []string{`""`}, `[""]`, true},
		{"nested arrays once", []string{`[1,2]`, `[]`}, `[[1,2],[]]`, true},
		{"exact number", []string{"123456789012345678901234567890"}, `[123456789012345678901234567890]`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			values := url.Values{"value": tc.values}
			target := jsontext.Value(`[7]`)
			present, err := DecodeJSONField(values, "value", &target, false, FieldItems)
			require.True(t, present)
			if tc.valid {
				require.NoError(t, err)
				require.JSONEq(t, tc.want, string(target))
			} else {
				require.Error(t, err)
				require.Equal(t, `[7]`, string(target), "validate every JSON item before typed decoding")
			}
			pairs, err := ParseRawPairs(values.Encode())
			require.NoError(t, err)
			raw, matched, err := FormJSONValue(pairs, "value", FieldItems)
			require.NoError(t, err)
			require.Equal(t, tc.valid, matched)
			if tc.valid {
				require.JSONEq(t, tc.want, string(raw))
			} else {
				require.Nil(t, raw)
			}
		})
	}
}

func TestJSONFieldApplicationPresence(t *testing.T) {
	for _, application := range []FieldApplication{FieldValue, FieldItems} {
		target := jsontext.Value(`[7]`)
		present, err := DecodeJSONField(url.Values{}, "value", &target, false, application)
		require.NoError(t, err)
		require.False(t, present)
		require.Equal(t, `[7]`, string(target))
		pairs, err := ParseRawPairs("other=1")
		require.NoError(t, err)
		value, matched, err := FormJSONValue(pairs, "value", application)
		require.NoError(t, err)
		require.True(t, matched)
		require.Nil(t, value)
	}
	values := url.Values{}
	present, err := EncodeJSONField(values, "value", []int{}, FieldItems)
	require.NoError(t, err)
	require.False(t, present)
	require.NotContains(t, values, "value")
	present, err = EncodeJSONField(values, "value", [][]int{{}}, FieldItems)
	require.NoError(t, err)
	require.True(t, present)
	require.Equal(t, []string{"[]"}, values["value"])
	for _, invalid := range []any{3, map[string]string{}, jsontext.Value("null")} {
		before := values.Encode()
		_, err := EncodeJSONField(values, "value", invalid, FieldItems)
		require.Error(t, err)
		require.Equal(t, before, values.Encode())
	}
}

func TestJSONFieldItemsDeterministicMaps(t *testing.T) {
	values := url.Values{}
	_, err := EncodeJSONField(values, "value", []map[string]map[string]int{{"z": {"b": 2, "a": 1}, "a": {"y": 2, "x": 1}}}, FieldItems)
	require.NoError(t, err)
	require.Equal(t, []string{`{"a":{"x":1,"y":2},"z":{"a":1,"b":2}}`}, values["value"])
}
