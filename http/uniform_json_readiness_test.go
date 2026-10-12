package http

import (
	"encoding/json/v2"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUniformJSONReadinessNamedFormCodec(t *testing.T) {
	type address struct {
		Street string `json:"street" form:"street"`
	}
	for _, test := range []struct {
		name  string
		value any
		raw   string
	}{
		{"boolean keys", map[bool]bool{true: false}, `{"true":false}`},
		{"integer keys", map[int][]string{3: {"v"}}, `{"3":["v"]}`},
		{"map", map[string]string{"space key": "a+b&c", "": "empty"}, `{"":"empty","space key":"a+b&c"}`},
		{"object", address{Street: "Main"}, `{"street":"Main"}`},
		{"nested map", map[string]map[string]string{"branch": {"leaf": "value"}}, `{"branch":{"leaf":"value"}}`},
		{"map of arrays", map[string][]string{"k": {"one", "two"}}, `{"k":["one","two"]}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			values := url.Values{}
			seen, err := EncodeFormValue(values, "field", test.value)
			require.NoError(t, err)
			require.True(t, seen)
			require.Equal(t, url.Values{"field": {test.raw}}, values)
		})
	}
}

func TestUniformJSONReadinessNamedFormDecode(t *testing.T) {
	for _, test := range []struct {
		name     string
		raw      string
		accepted bool
	}{
		{"valid", `{"a":"b"}`, true},
		{"truncated", `{"a":`, false},
		{"wrong type", `[]`, false},
		{"null", `null`, false},
		{"spaced null", " \nnull\t", false},
		{"trailing document", `{"a":"b"} {}`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			var target map[string]string
			seen, err := DecodeFormValue(url.Values{"field": {test.raw}}, "field", &target)
			if test.accepted {
				require.NoError(t, err)
				require.True(t, seen)
				require.Equal(t, map[string]string{"a": "b"}, target)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestUniformJSONReadinessNoBracketFallback(t *testing.T) {
	var target map[string]string
	seen, err := DecodeFormValue(url.Values{"field[a]": {"b"}}, "field", &target)
	require.NoError(t, err)
	require.False(t, seen, "bracket keys are not the named JSON field")
	require.Nil(t, target)
}

// Deliberately different tags prove that inner form tags cannot redefine JSON.
type jsonFieldTagged struct {
	Value string `json:"json_name" form:"form_name"`
}
type jsonFieldCustom map[string]string

func (v jsonFieldCustom) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{"custom": v["value"]}, json.Deterministic(true))
}
func (v *jsonFieldCustom) UnmarshalJSON(data []byte) error {
	var encoded map[string]string
	if err := json.Unmarshal(data, &encoded); err != nil {
		return err
	}
	*v = jsonFieldCustom{"value": encoded["custom"]}
	return nil
}
func TestJSONFieldRepresentation(t *testing.T) {
	for _, test := range []struct {
		name          string
		value, target any
		raw           string
	}{
		{"inner tags", jsonFieldTagged{Value: "v"}, &jsonFieldTagged{}, `{"json_name":"v"}`},
		{"custom JSON", jsonFieldCustom{"value": "v"}, new(jsonFieldCustom), `{"custom":"v"}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			values := url.Values{}
			seen, err := EncodeFormValue(values, "outer", test.value)
			require.NoError(t, err)
			require.True(t, seen)
			require.Equal(t, url.Values{"outer": {test.raw}}, values)
			seen, err = DecodeFormValue(values, "outer", test.target)
			require.NoError(t, err)
			require.True(t, seen)
			expected, err := json.Marshal(test.value, json.Deterministic(true))
			require.NoError(t, err)
			actual, err := json.Marshal(test.target, json.Deterministic(true))
			require.NoError(t, err)
			require.Equal(t, string(expected), string(actual))
		})
	}
}
