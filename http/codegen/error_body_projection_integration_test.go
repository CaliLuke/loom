package codegen

const errorBodyProjectionHarness = `package projection_test

import (
	"context"
	"encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	service "example.com/projection/gen/projection"
	client "example.com/projection/gen/http/projection/client"
	server "example.com/projection/gen/http/projection/server"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
)

func TestErrorBodyProjection(t *testing.T) {
	const nested = ` + "`" + `{"field":["Invalid value"],"nested":{"integer":9007199254740993}}` + "`" + `
	text := "render failed"
	var nestedMap map[string]loom.JSONValue
	require.NoError(t, json.Unmarshal([]byte(nested), &nestedMap))
	for _, tc := range []struct {
		name, method, contentType, wire string
		value error
	}{
		{"html", "html", "text/html; charset=utf-8", "<html>render failed</html>", &service.HTMLError{Kind: "html", Body: "<html>render failed</html>"}},
		{"plain", "plain", "text/plain", text, &service.PlainError{Kind: "plain", Body: text}},
		{"empty string", "plain", "text/plain", "", &service.PlainError{Kind: "plain", Body: ""}},
		{"any object", "any", "application/json", nested+"\n", &service.AnyError{Kind: "any", Body: loom.JSONValue(nested)}},
		{"any null", "any", "application/json", "null\n", &service.AnyError{Kind: "any", Body: loom.JSONValue("null")}},
		{"any integer", "any", "application/json", "9007199254740993\n", &service.AnyError{Kind: "any", Body: loom.JSONValue("9007199254740993")}},
		{"map", "map", "application/json", nested+"\n", &service.MapError{Kind: "map", Body: nestedMap}},
		{"object", "object", "application/json", "{\"message\":\"render failed\"}\n", &service.ObjectError{Kind: "object", Body: &service.ErrorDetails{Message: text}}},
		{"explicit", "explicit", "application/json", "{\"body\":\"render failed\"}\n", &service.ExplicitError{Kind: "explicit", Body: text}},
		{"renamed", "renamed", "text/plain", text, &service.RenamedError{Kind: "renamed", Content: text}},
		{"optional", "optional", "text/plain", text, &service.OptionalError{Kind: "optional", Body: &text}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			endpoint := func(context.Context, any) (any, error) {
				return nil, tc.value
			}
			endpoints := &service.Endpoints{
				HTML: endpoint, Plain: endpoint, Any: endpoint, Map: endpoint,
				Object: endpoint, Explicit: endpoint, Renamed: endpoint, Optional: endpoint,
			}
			mux := loomhttp.NewMuxer()
			generated := server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, func(_ context.Context, _ http.ResponseWriter, err error) {
				t.Errorf("unexpected encoding failure: %v", err)
			}, nil)
			server.Mount(mux, generated)
			httpServer := httptest.NewServer(mux)
			t.Cleanup(httpServer.Close)

			response, err := httpServer.Client().Get(httpServer.URL + "/" + tc.method)
			require.NoError(t, err)
			body, err := io.ReadAll(response.Body)
			require.NoError(t, err)
			require.NoError(t, response.Body.Close())
			require.Equal(t, http.StatusInternalServerError, response.StatusCode)
			require.Equal(t, tc.contentType, response.Header.Get("Content-Type"))
			require.Equal(t, tc.method, response.Header.Get("loom-error"))
			require.Equal(t, tc.wire, string(body))

			serverURL, err := url.Parse(httpServer.URL)
			require.NoError(t, err)
			cli := client.NewClient(serverURL.Scheme, serverURL.Host, httpServer.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
			call := map[string]loom.Endpoint{
				"html": cli.HTML(), "plain": cli.Plain(), "any": cli.Any(), "map": cli.Map(),
				"object": cli.Object(), "explicit": cli.Explicit(), "renamed": cli.Renamed(), "optional": cli.Optional(),
			}[tc.method]
			result, err := call(t.Context(), nil)
			require.Nil(t, result)
			require.Equal(t, tc.value, err)
		})
	}
}
`
