package headerlist

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFraming(t *testing.T) {
	for _, tc := range []struct {
		name    string
		lines   []string
		present bool
		want    []string
	}{
		{"absent", nil, false, nil},
		{"empty", []string{""}, true, []string{}},
		{"combined", []string{"a, b"}, true, []string{"a", "b"}},
		{"repeated", []string{"a", "b, c"}, true, []string{"a", "b", "c"}},
		{"empty-members", []string{",a,\t, b,,"}, true, []string{"a", "b"}},
		{"quotes-literal", []string{`"a,b"`}, true, []string{`"a`, `b"`}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, Decode(tc.lines, tc.present))
		})
	}
	for _, item := range []string{"", "secret,comma", " secret", "secret\t"} {
		_, err := Encode([]string{item})
		require.Error(t, err)
		require.NotContains(t, err.Error(), "secret")
	}
	wire, err := Encode([]string{"a b", "c"})
	require.NoError(t, err)
	require.Equal(t, "a b, c", wire)
}
