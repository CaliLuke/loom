package security

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodeBearer(t *testing.T) {
	for _, tc := range []struct {
		input, token string
		ok           bool
	}{
		{"Bearer abc.def_+/=", "abc.def_+/=", true},
		{"bEaReR  token", "token", true},
		{"", "", false},
		{"token", "", false},
		{"Basic token", "", false},
		{"Bearer ", "", false},
		{"Bearer\ttoken", "", false},
		{"Bearer token ", "", false},
		{"Bearer two tokens", "", false},
		{"Bearer token\r\n", "", false},
		{"Bearer token\v", "", false},
		{"Bearer token\u00a0", "", false},
	} {
		t.Run(tc.input, func(t *testing.T) {
			token, ok := DecodeBearer(tc.input)
			require.Equal(t, tc.ok, ok)
			require.Equal(t, tc.token, token)
		})
	}
}

func TestOtherAuthorizationScheme(t *testing.T) {
	for _, tc := range []struct {
		input string
		other bool
	}{
		{"Basic dTpw", true}, {"bAsIc dTpw", true}, {"Digest a=b", true},
		{"Custom+Scheme value", true}, {"Bearer good", false}, {"bearer bad value", false},
		{"Bearer", false}, {"Bearer ", false}, {"Basic", false}, {"Basic ", false},
		{"Basic\tvalue", false}, {" Basic value", false}, {"Básic value", false},
		{"Basic: value", false}, {"raw-token", false}, {"", false},
	} {
		t.Run(tc.input, func(t *testing.T) {
			require.Equal(t, tc.other, IsOtherAuthorizationScheme(tc.input, "Bearer"))
		})
	}
}
