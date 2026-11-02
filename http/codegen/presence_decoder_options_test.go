package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

func TestPresenceDecoderOptionsGeneratedHTTP(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		item := Type("Item", func() {
			Attribute("known", String)
			Required("known")
		})
		body := Type("Document", func() {
			Attribute("items", ArrayOf(item))
			Required("items")
		})
		Service("presence", func() {
			for _, name := range []string{"required", "optional"} {
				Method(name, func() {
					Payload(func() {
						Attribute("body", body)
						if name == "required" {
							Required("body")
						}
					})
					HTTP(func() {
						POST("/" + name)
						Body("body")
						if name == "optional" {
							OptionalRequestBody()
						}
					})
				})
			}
		})
	})
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/presence", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "presence_test.go"), []byte(presenceDecoderOptionsHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-race", "-count=1", "./...")
}

const presenceDecoderOptionsHarness = `package presence_test

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	service "example.com/presence/gen/presence"
	server "example.com/presence/gen/http/presence/server"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
)

func TestRequestOptions(t *testing.T) {
	for _, method := range []string{"required", "optional"} {
		for _, tc := range []struct {
			name, wire string
			code string
			want []*service.Item
		}{
			{"empty", "", "missing_payload", nil},
			{"whitespace", " \n\t", "missing_payload", nil},
			{"malformed", "!", "decode_payload", nil},
			{"truncated", "{\"items\":[{", "decode_payload", nil},
			{"unknown", "{\"items\":[{\"known\":\"ok\",\"unknown\":true}]}", "decode_payload", nil},
			{"null element", "{\"items\":[null]}", "decode_payload", nil},
			{"valid", "{\"items\":[{\"known\":\"ok\"}]}", "", []*service.Item{{Known: "ok"}}},
			{"empty array", "{\"items\":[]}", "", []*service.Item{}},
		} {
			t.Run(method+"/"+tc.name, func(t *testing.T) {
				var received []*service.Item
				calls := 0
				endpoint := func(_ context.Context, payload any) (any, error) {
					calls++
					switch p := payload.(type) {
					case *service.RequiredPayload:
						received = p.Body.Items
					case *service.OptionalPayload:
						if p.Body != nil {
							received = p.Body.Items
						}
					default:
						t.Errorf("unexpected payload type %T", payload)
					}
					return nil, nil
				}
				decoder := func(r *http.Request) loomhttp.Decoder {
					return loomhttp.EncodingFunc(func(out any) error {
						return json.UnmarshalDecode(jsontext.NewDecoder(r.Body), out, loom.JSONOptions(), json.RejectUnknownMembers(true))
					})
				}
				mux := loomhttp.NewMuxer()
				server.Mount(mux, server.New(&service.Endpoints{Required: endpoint, Optional: endpoint}, mux,
					decoder, loomhttp.ResponseEncoder, nil, nil))
				request := httptest.NewRequest(http.MethodPost, "/"+method, strings.NewReader(tc.wire))
				request.Header.Set("Content-Type", "application/json")
				response := httptest.NewRecorder()
				mux.ServeHTTP(response, request)
				code := tc.code
				if method == "optional" && code == "missing_payload" {
					code = ""
				}
				if code == "" {
					require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())
					require.Equal(t, 1, calls)
					require.Equal(t, tc.want, received)
					return
				}
				require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
				require.Zero(t, calls)
				require.Nil(t, received)
				var problem loomhttp.ProblemResponse
				require.NoError(t, json.Unmarshal(response.Body.Bytes(), &problem))
				require.Equal(t, code, problem.Code)
				detail := "invalid request body"
				if code == "missing_payload" {
					detail = "validation error"
				}
				require.Equal(t, detail, problem.Detail)
			})
		}
	}
}
`
