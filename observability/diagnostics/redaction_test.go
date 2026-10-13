package diagnostics

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPolicyBodyProjection(t *testing.T) {
	const secret = "sensitive-value"
	policy := New(Fields{JSON: []string{"permit"}, Form: []string{"proof"}})
	for _, tc := range []struct {
		name, contentType, input, want string
		complete                       bool
	}{
		{"nested JSON", "application/json", `{"nested":[{"permit":"` + secret + `","ordinary":"visible"}]}`, `{"nested":[{"ordinary":"visible","permit":"[REDACTED]"}]}`, true},
		{"form wire name", "application/x-www-form-urlencoded", "proof=" + secret + "&ordinary=visible", "ordinary=visible&proof=%5BREDACTED%5D", true},
		{"repeated form", "application/x-www-form-urlencoded", "proof=" + secret + "&proof=" + secret, "proof=%5BREDACTED%5D", true},
		{"invalid JSON", "application/json", `{"permit":"` + secret, "[invalid JSON body omitted]", true},
		{"incomplete valid JSON", "application/json", `{"permit":"` + secret + `"}`, "[incomplete structured body omitted]", false},
		{"incomplete form", "application/x-www-form-urlencoded", "proof=" + secret, "[incomplete structured body omitted]", false},
		{"invalid form", "application/x-www-form-urlencoded", "proof=%gg" + secret, "[invalid form body omitted]", true},
		{"opaque credential", "text/plain", secret, "[credential body with unsupported content type omitted]", true},
		{"absent content type", "", secret, "[credential body with unsupported content type omitted]", true},
		{"invalid content type", "not a media type", secret, "[credential body with unsupported content type omitted]", true},
		{"empty", "application/json", "", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := []byte(tc.input)
			result := policy.Body(tc.contentType, input, false, tc.complete)
			require.Equal(t, tc.want, string(result))
			require.Equal(t, tc.input, string(input))
			require.NotContains(t, string(result), secret)
		})
	}
	require.Equal(t, "ordinary", string(New().Body("text/plain", []byte("ordinary"), false, false)))
	require.Equal(t, "[credential body omitted]", string(New(Fields{Body: true}).Body("application/json", []byte(`"`+secret+`"`), false, true)))
	require.Equal(t, "[body omitted after 65536 bytes]", string(policy.Body("application/json", []byte(secret), true, false)))
}

func TestPolicyImmutableInputsAndOutputs(t *testing.T) {
	fields := Fields{Headers: []string{"X-Permit"}, Query: []string{"ticket"}, JSON: []string{"permit"}, Form: []string{"proof"}, Path: true}
	policy := New(fields)
	fields.Headers[0], fields.Query[0], fields.JSON[0], fields.Form[0] = "ordinary", "ordinary", "ordinary", "ordinary"
	fields.Path = false
	headers := http.Header{"X-Permit": []string{"sensitive-value"}, "Ordinary": []string{"visible"}}
	masked := policy.Headers(headers)
	require.Equal(t, redactedValue, masked.Get("X-Permit"))
	require.Equal(t, "visible", masked.Get("Ordinary"))
	masked["X-Permit"][0] = "changed"
	require.Equal(t, "sensitive-value", headers.Get("X-Permit"))
	target, err := url.Parse("https://user:password@example.test/sensitive%2Fvalue?ticket=sensitive-value&ticket=second&ordinary=visible")
	require.NoError(t, err)
	original := target.String()
	maskedURL := policy.URL(target)
	require.NotContains(t, maskedURL.String(), "sensitive")
	require.NotContains(t, maskedURL.String(), "password")
	require.Empty(t, maskedURL.RawPath)
	require.Equal(t, "visible", maskedURL.Query().Get("ordinary"))
	require.Equal(t, original, target.String())
	vars := map[string]string{"ticket": "sensitive-value"}
	require.Equal(t, map[string]string{"ticket": redactedValue}, policy.PathValues(vars))
	require.Equal(t, "sensitive-value", vars["ticket"])
	require.Contains(t, string(policy.Body("application/json", []byte(`{"permit":"sensitive-value","ordinary":"visible"}`), false, true)), `"ordinary":"visible"`)
	require.NotContains(t, string(policy.Body("application/json", []byte(`{"permit":"sensitive-value"}`), false, true)), "sensitive-value")
	require.NotContains(t, string(policy.Body("application/x-www-form-urlencoded", []byte("proof=sensitive-value"), false, true)), "sensitive-value")
}

func TestPolicyConventionalNames(t *testing.T) {
	for _, name := range []string{"apiKey", "api_key", "API-KEY", "api key", "accessToken", "refreshToken", "sessionToken", "clientSecret", "key", "token", "password", "Cookie", "Set-Cookie"} {
		t.Run(name, func(t *testing.T) {
			require.True(t, sensitiveName(name))
		})
	}
	for _, name := range []string{"ordinary", "monkey", "keyboard", "value", "tokenizer"} {
		require.False(t, sensitiveName(name), name)
	}
}

func TestPolicyMalformedQuery(t *testing.T) {
	original := &url.URL{Scheme: "https", Host: "example.test", Path: "/resource", RawQuery: "ticket=secret&broken=%xx"}
	result := New(Fields{Query: []string{"ticket"}}).URL(original)
	require.NotContains(t, result.String(), "secret")
	require.Contains(t, result.String(), "invalid query omitted")
	require.True(t, strings.Contains(original.RawQuery, "secret"))
}

func TestPolicyOpaqueCredentialPath(t *testing.T) {
	original := &url.URL{Scheme: "http", Opaque: "//example.test/secret-value"}
	result := New(Fields{Path: true}).URL(original)
	require.NotContains(t, result.String(), "secret-value")
	require.Equal(t, "//example.test/secret-value", original.Opaque)
}
