package http

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormValuesRootMapEmptyKey(t *testing.T) {
	for _, c := range []struct {
		name   string
		input  any
		target any
		wire   string
	}{
		{"string", &map[string]string{"": "empty", "normal": "value"}, &map[string]string{}, "=empty&normal=value"},
		{"empty string", &map[string]string{"": ""}, &map[string]string{}, "="},
		{"integer", &map[string]int{"": 0}, &map[string]int{}, "=0"},
		{"boolean", &map[string]bool{"": false}, &map[string]bool{}, "=false"},
		{"bytes", &map[string][]byte{"": []byte("data")}, &map[string][]byte{}, "=data"},
	} {
		t.Run(c.name, func(t *testing.T) {
			values, err := EncodeFormValues(c.input)
			require.NoError(t, err)
			require.Equal(t, c.wire, values.Encode())
			parsed, err := url.ParseQuery(c.wire)
			require.NoError(t, err)
			target := c.target
			require.NoError(t, DecodeFormValues(parsed, target))
			require.Equal(t, c.input, target)
		})
	}
}

func TestFormValuesUnnamedScalarRootsRemainRejected(t *testing.T) {
	for _, value := range []any{"text", 1, false, []byte("data")} {
		_, err := EncodeFormValues(value)
		require.ErrorContains(t, err, "requires a field name")
	}
}
