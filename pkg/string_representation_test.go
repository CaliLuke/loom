package loom

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type (
	representationAlias       string
	representationCodecString string
	representationToken       struct {
		value string
	}
	representationPointerToken struct {
		value string
	}
	representationOpaque struct {
		value string
	}
	representationCycle *representationCycle
)

var errRepresentationText = errors.New("invalid token")

func TestStringValueRepresentation(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value any
		want  string
		bad   bool
	}{
		{"native", "", "", false},
		{"alias", representationAlias("value"), "value", false},
		{"authored value method", representationToken{"abc"}, "token:abc", false},
		{"pointer method", &representationPointerToken{"abc"}, "pointer:abc", false},
		{"nil pointer", (*representationPointerToken)(nil), "", true},
		{"opaque", representationOpaque{"abc"}, "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value, err := StringValue(tc.value, "result.id")
			if tc.bad {
				require.ErrorContains(t, err, "result.id")
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.want, value)
			}
		})
	}
	value, err := StringValue(representationPointerToken{"abc"}, "result.id")
	require.NoError(t, err)
	require.Equal(t, "pointer:abc", value)
}

func TestStringValuePointerChainPrecedence(t *testing.T) {
	value := representationCodecString("é")
	pointer := &value
	for _, input := range []any{value, pointer, &pointer} {
		text, err := StringValue(input, "result.name")
		require.NoError(t, err)
		require.Equal(t, "codec:é", text)
		require.Equal(t, representationCodecString("é"), value)
	}
	parsed, err := ParseStringValue[**representationCodecString]("é", "default.name")
	require.NoError(t, err)
	text, err := StringValue(parsed, "result.name")
	require.NoError(t, err)
	require.Equal(t, "codec:é", text)
}

func TestParseStringValueRepresentation(t *testing.T) {
	native, err := ParseStringValue[string]("", "default.id")
	require.NoError(t, err)
	require.Empty(t, native)
	alias, err := ParseStringValue[representationAlias]("value", "default.id")
	require.NoError(t, err)
	require.Equal(t, representationAlias("value"), alias)
	token, err := ParseStringValue[representationToken]("token:abc", "default.id")
	require.NoError(t, err)
	require.Equal(t, "abc", token.value)
	pointer, err := ParseStringValue[*representationToken]("token:abc", "default.id")
	require.NoError(t, err)
	require.Equal(t, "abc", pointer.value)
	bad, err := ParseStringValue[representationToken]("bad", "default.id")
	require.ErrorIs(t, err, errRepresentationText)
	require.ErrorContains(t, err, "default.id")
	require.Equal(t, representationToken{}, bad)
	_, err = ParseStringValue[representationOpaque]("abc", "default.id")
	require.ErrorContains(t, err, "default.id")
	_, err = ParseStringValue[representationCycle]("abc", "default.id")
	require.ErrorContains(t, err, "recursive pointer")
}

func (t representationToken) String() string {
	return "token:" + t.value
}

func (t *representationToken) UnmarshalText(value []byte) error {
	if !strings.HasPrefix(string(value), "token:") {
		t.value = "partially mutated"
		return fmt.Errorf("parse: %w", errRepresentationText)
	}
	t.value = strings.TrimPrefix(string(value), "token:")
	return nil
}

func (t *representationPointerToken) String() string {
	return "pointer:" + t.value
}

func (value *representationCodecString) String() string {
	return "codec:" + string(*value)
}
