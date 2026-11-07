package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRawFormRetainsBodyPresence(t *testing.T) {
	for _, tc := range []struct {
		name, raw string
		present   bool
		keys      int
	}{
		{"empty", "", false, 0},
		{"separators", "&&", true, 0},
		{"empty field", "value=", true, 1},
		{"value", "value=hello", true, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc.raw))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			request.ContentLength = -1 // Chunked input cannot use declared length as presence.
			raw, err := ParseRawFormWithLimit(request, 1024)
			require.NoError(t, err)
			require.Equal(t, tc.present, raw.HasInput())
			require.Len(t, request.PostForm, tc.keys)
		})
	}
}
