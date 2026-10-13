package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAddRequestCookie checks exact values and validation before header mutation.
func TestAddRequestCookie(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value string
		valid bool
	}{
		{"token", "plain", true},
		{"token", "", true},
		{"token", "a b", true},
		{"token", "a,b", true},
		{"token", "a;b", false},
		{"token", "a\"b", false},
		{"token", "a\\b", false},
		{"token", "a\x00b", false},
		{"token", "a\nb", false},
		{"token", "a\rb", false},
		{"token", "café", false},
		{"bad name", "value", false},
		{"", "value", false},
		{"__Host-token", "value", true},
	} {
		t.Run(tc.name+"/"+tc.value, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Add("Cookie", "existing=kept")
			if !tc.valid {
				req.Header.Add("Cookie", "second=kept")
			}
			req.Header.Add("X-Control", "unchanged")
			before := req.Header.Clone()
			err := AddRequestCookie(req, tc.name, tc.value)
			if !tc.valid {
				require.Error(t, err)
				require.Equal(t, before, req.Header, "invalid input must not mutate any header")
				return
			}
			require.NoError(t, err)
			cookies := req.Cookies()
			require.Len(t, cookies, 2)
			require.Equal(t, "existing", cookies[0].Name)
			require.Equal(t, "kept", cookies[0].Value)
			require.Equal(t, tc.name, cookies[1].Name)
			require.Equal(t, tc.value, cookies[1].Value)
			require.Equal(t, before.Values("X-Control"), req.Header.Values("X-Control"))
		})
	}
}
