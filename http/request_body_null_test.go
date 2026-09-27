package http

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	loom "github.com/CaliLuke/loom/pkg"
)

// TestNonNullableJSONRequestBody checks root null rejection independently of
// empty, malformed, truncated, and valid JSON decoding.
func TestNonNullableJSONRequestBody(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want error
	}{
		{"empty", "", io.EOF},
		{"whitespace", " \t\r\n", io.EOF},
		{"null", "null", errNullRequestBody},
		{"padded null", " \nnull\t", errNullRequestBody},
		{"valid", `{"value":"ok"}`, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/", strings.NewReader(tc.body))
			var got struct {
				Value string `json:"value"`
			}
			err := RequestDecoder(WithNonNullableBody(r)).Decode(&got)
			require.ErrorIs(t, err, tc.want)
			if tc.want == nil {
				require.Equal(t, "ok", got.Value)
			} else {
				require.Empty(t, got.Value)
			}
		})
	}
	for _, body := range []string{`{x}`, `{"value":`, "null true", "\u00a0null"} {
		t.Run(body, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/", strings.NewReader(body))
			var value any
			err := RequestDecoder(WithNonNullableBody(r)).Decode(&value)
			require.Error(t, err)
			require.NotErrorIs(t, err, io.EOF)
			require.NotErrorIs(t, err, errNullRequestBody)
		})
	}
}

// TestNullableJSONRequestBody checks that unmarked requests preserve explicit
// null in nullable and unconstrained values.
func TestNullableJSONRequestBody(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader("null"))
	var got loom.Nullable[string]
	require.NoError(t, RequestDecoder(r).Decode(&got))
	require.True(t, got.IsNull())
	r = httptest.NewRequest("POST", "/", strings.NewReader("null"))
	var raw loom.JSONValue
	require.NoError(t, RequestDecoder(r).Decode(&raw))
	require.Equal(t, "null", string(raw))
}

// TestNonNullableBodyPreservesOtherCodecs checks that the JSON null policy does
// not reinterpret text or XML values.
func TestNonNullableBodyPreservesOtherCodecs(t *testing.T) {
	for _, tc := range []struct {
		contentType string
		body        string
	}{
		{"text/plain", "null"},
		{"application/xml", "<string>null</string>"},
	} {
		t.Run(tc.contentType, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/", strings.NewReader(tc.body))
			r.Header.Set("Content-Type", tc.contentType)
			var got string
			require.NoError(t, RequestDecoder(WithNonNullableBody(r)).Decode(&got))
			require.Equal(t, "null", got)
		})
	}
}
