package loom

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPresenceDecodeCallerOptions(t *testing.T) {
	type item struct {
		Known string `json:"known"`
	}
	for _, tc := range []struct {
		name     string
		newValue func() any
	}{
		{"plain", func() any {
			return new([]*item)
		}},
		{"optional", func() any {
			return new([]Optional[*item])
		}},
		{"nullable", func() any {
			return new([]Nullable[*item])
		}},
		{"nested", func() any {
			return new([]Optional[Nullable[*item]])
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := json.Unmarshal([]byte(`[{"known":"ok","unknown":true}]`), tc.newValue(), json.RejectUnknownMembers(true))
			require.Error(t, err)
			var semantic *json.SemanticError
			require.ErrorAs(t, err, &semantic)
			require.Equal(t, jsontext.Pointer("/0/unknown"), semantic.JSONPointer)
			require.NoError(t, json.Unmarshal([]byte(`[{"known":"ok","unknown":true}]`), tc.newValue()))
			require.NoError(t, json.Unmarshal([]byte(`[{"KNOWN":"ok"}]`), tc.newValue(), json.RejectUnknownMembers(true), json.MatchCaseInsensitiveNames(true)))
		})
	}
}

func TestPresenceDecodeCustomCodecAndBooleanKeys(t *testing.T) {
	type label string
	codec := json.WithUnmarshalers(json.UnmarshalFromFunc(func(dec *jsontext.Decoder, out *label) error {
		token, err := dec.ReadToken()
		if err != nil {
			return err
		}
		*out = label(strings.ToUpper(token.String()))
		return nil
	}))
	var value Optional[Nullable[map[bool]label]]
	require.NoError(t, json.Unmarshal([]byte(`{"true":"hello"}`), &value, codec))
	nullable, ok := value.Value()
	require.True(t, ok)
	actual, ok := nullable.Value()
	require.True(t, ok)
	require.Equal(t, map[bool]label{true: "HELLO"}, actual)
}

func TestPresenceDecodeStringifiedNumbers(t *testing.T) {
	var optional Optional[int]
	require.NoError(t, json.Unmarshal([]byte(`"42"`), &optional, json.StringifyNumbers(true)))
	value, ok := optional.Value()
	require.True(t, ok)
	require.Equal(t, 42, value)
	var nullable Nullable[int]
	require.NoError(t, json.Unmarshal([]byte(`"42"`), &nullable, json.StringifyNumbers(true)))
	value, ok = nullable.Value()
	require.True(t, ok)
	require.Equal(t, 42, value)
}

func TestPresenceDecodeInputContract(t *testing.T) {
	for _, wire := range []string{"", " \t\n", "!", `{"known":`, `{"known":1}`, `{"known":"a","known":"b"}`} {
		t.Run(wire, func(t *testing.T) {
			type item struct {
				Known string `json:"known"`
			}
			var optional Optional[*item]
			require.Error(t, json.Unmarshal([]byte(wire), &optional))
			require.False(t, optional.Present())
			var nullable Nullable[*item]
			require.Error(t, json.Unmarshal([]byte(wire), &nullable))
		})
	}
	optional := OptionalValue(42)
	require.ErrorIs(t, json.Unmarshal([]byte(" \nnull\t"), &optional, json.RejectUnknownMembers(true)), ErrNullOptional)
	require.False(t, optional.Present())
	nullable := NullableValue(42)
	require.NoError(t, json.Unmarshal([]byte(" \nnull\t"), &nullable, json.RejectUnknownMembers(true)))
	require.True(t, nullable.IsNull())
}

func TestPresenceDecodeConsumesOneValue(t *testing.T) {
	dec := jsontext.NewDecoder(strings.NewReader(`42 null 7`))
	var optional Optional[int]
	require.NoError(t, json.UnmarshalDecode(dec, &optional))
	value, ok := optional.Value()
	require.True(t, ok)
	require.Equal(t, 42, value)
	var nullable Nullable[int]
	require.NoError(t, json.UnmarshalDecode(dec, &nullable))
	require.True(t, nullable.IsNull())
	var tail int
	require.NoError(t, json.UnmarshalDecode(dec, &tail))
	require.Equal(t, 7, tail)
}

func TestPresenceDecodeCallerKeyCodecPrecedence(t *testing.T) {
	codec := json.WithUnmarshalers(json.UnmarshalFromFunc(func(dec *jsontext.Decoder, out *bool) error {
		token, err := dec.ReadToken()
		if err != nil {
			return err
		}
		*out = token.String() == "yes"
		return nil
	}))
	var optional Optional[Nullable[map[bool]int]]
	require.NoError(t, json.Unmarshal([]byte(`{"yes":42}`), &optional, codec))
	nullable, ok := optional.Value()
	require.True(t, ok)
	value, ok := nullable.Value()
	require.True(t, ok)
	require.Equal(t, map[bool]int{true: 42}, value)
}
