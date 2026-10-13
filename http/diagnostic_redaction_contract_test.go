package http

import (
	"bytes"
	"io"
	stdhttp "net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/observability/diagnostics"
)

func TestDebugCredentialConventionalNames(t *testing.T) {
	for _, name := range []string{"apiKey", "accessToken", "refreshToken", "clientSecret", "sessionToken", "key"} {
		for _, location := range []string{"query", "header", "JSON", "form", "response JSON"} {
			t.Run(name+"/"+location, func(t *testing.T) {
				const secret = "diagnostic-secret-value"
				target := "https://example.test/resource?ordinary=visible"
				body := `{"ordinary":"visible"}`
				contentType := "application/json"
				responseBody := `{"ordinary":"visible"}`
				switch location {
				case "query":
					target += "&" + url.QueryEscape(name) + "=" + secret
				case "JSON":
					body = `{"ordinary":"visible","` + name + `":"` + secret + `"}`
				case "form":
					body = "ordinary=visible&" + url.QueryEscape(name) + "=" + secret
					contentType = "application/x-www-form-urlencoded"
				case "response JSON":
					responseBody = `{"ordinary":"visible","` + name + `":"` + secret + `"}`
				}
				req, err := stdhttp.NewRequest(stdhttp.MethodPost, target, strings.NewReader(body))
				require.NoError(t, err)
				req.Header.Set("Content-Type", contentType)
				if location == "header" {
					req.Header.Set(name, secret)
				}
				doer := &debugCaptureDoer{t: t, responseBody: responseBody, responseHeader: stdhttp.Header{"Content-Type": []string{"application/json"}}}
				debug := NewDebugDoer(doer).(*debugDoer)
				var automatic bytes.Buffer
				debug.output = &automatic
				response, err := debug.Do(req)
				require.NoError(t, err)
				wireBody, readErr := io.ReadAll(response.Body)
				require.NoError(t, readErr)
				require.NoError(t, response.Body.Close())
				require.Equal(t, target, req.URL.String())
				require.Equal(t, body, doer.requestBody)
				require.Equal(t, responseBody, string(wireBody))
				var snapshot bytes.Buffer
				debug.Fprint(&snapshot)
				require.Equal(t, automatic.String(), snapshot.String())
				require.Contains(t, snapshot.String(), "visible")
				require.NotContains(t, snapshot.String(), secret)
			})
		}
	}
}

func TestDebugCredentialPolicySurvivesDelayedCompletion(t *testing.T) {
	const secret = "diagnostic-secret-value"
	fields := diagnostics.Fields{Headers: []string{"X-Permit"}, Query: []string{"ticket"}, JSON: []string{"permit"}}
	doer := &debugCaptureDoer{t: t, responseBody: `{"nested":[{"permit":"` + secret + `","ordinary":"visible"}]}`, responseHeader: stdhttp.Header{"Content-Type": []string{"application/json"}, "X-Permit": []string{secret}}}
	debug := NewDebugDoer(doer, fields).(*debugDoer)
	var automatic bytes.Buffer
	debug.output = &automatic
	req, err := stdhttp.NewRequest(stdhttp.MethodGet, "http://example.test/?ticket="+secret, nil)
	require.NoError(t, err)
	req.Body = stdhttp.NoBody
	response, err := debug.Do(req)
	require.NoError(t, err)
	require.Empty(t, automatic.String())
	fields.Headers[0], fields.Query[0], fields.JSON[0] = "ordinary", "ordinary", "ordinary"
	response.Header.Set("Content-Type", "text/plain")
	data, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	require.Contains(t, string(data), secret)
	require.NotContains(t, automatic.String(), secret)
	require.Contains(t, automatic.String(), "visible")
	var snapshot bytes.Buffer
	debug.Fprint(&snapshot)
	require.Equal(t, automatic.String(), snapshot.String())
}
