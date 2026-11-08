package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestHTTPStatusBodyPresenceAdmission(t *testing.T) {
	cases := []struct {
		name     string
		selector any
		code     int
		body     string
		wantBody bool
		wantErr  string
	}{
		{"explicit empty 204", Status2XX, 204, "explicit", true, "constraints exclude every owned status"},
		{"explicit empty 304", Status3XX, 304, "explicit", true, "constraints exclude every owned status"},
		{"additional properties 204", Status2XX, 204, "additional", true, "constraints exclude every owned status"},
		{"additional properties 304", Status3XX, 304, "additional", true, "constraints exclude every owned status"},
		{"explicit empty allowed", Status2XX, 201, "explicit", true, ""},
		{"additional properties allowed", Status2XX, 201, "additional", true, ""},
		{"implicit no body 204", Status2XX, 204, "implicit", false, ""},
		{"implicit no body 304", Status3XX, 304, "implicit", false, ""},
		{"explicit Empty 204", Status2XX, 204, "sentinel", false, ""},
		{"explicit Empty 304", Status3XX, 304, "sentinel", false, ""},
		{"exact explicit empty remains rejected", StatusNoContent, 204, "explicit", true, "does not allow response body"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			design := func() {
				empty := Type("EmptyRecord", func() {
					Meta("openapi:additionalProperties", "false")
				})
				value := Type("Value", func() {
					Attribute("status", Int, func() {
						Enum(tc.code)
					})
					Required("status")
					if tc.body == "additional" {
						AdditionalAttributes(String)
					}
				})
				Service("bodies", func() {
					Method("read", func() {
						Result(value)
						HTTP(func() {
							GET("/")
							Response(tc.selector, func() {
								StatusField("status")
								switch tc.body {
								case "explicit":
									Body(empty)
								case "sentinel":
									Body(Empty)
								}
							})
						})
					})
				})
			}
			if tc.wantErr != "" {
				require.ErrorContains(t, expr.RunInvalidDSL(t, design), tc.wantErr)
				return
			}
			root := expr.RunDSL(t, design)
			body := root.API.HTTP.Service("bodies").Endpoint("read").Responses[0].Body
			require.Equal(t, tc.wantBody, body.Type != expr.Empty)
			if tc.body == "additional" {
				require.NotNil(t, body.TypedObjectDeclaration())
			}
		})
	}
}

func TestSSECookieHandshakeAdmission(t *testing.T) {
	for _, mode := range []string{"untyped cookie", "typed cookie", "no metadata"} {
		t.Run(mode, func(t *testing.T) {
			design := func() {
				event := Type("Event", func() {
					Attribute("session", String)
					Attribute("message", String)
					Required("session", "message")
				})
				metadata := Type("Metadata", func() {
					Attribute("session", String)
					Required("session")
				})
				Service("events", func() {
					Method("watch", func() {
						StreamingResult(event)
						HTTP(func() {
							GET("/")
							ServerSentEvents()
							switch mode {
							case "untyped cookie":
								Response(StatusOK, func() {
									Cookie("session")
								})
							case "typed cookie":
								Response(StatusOK, metadata, func() {
									Cookie("session")
								})
							case "no metadata":
								Response(StatusOK)
							}
						})
					})
				})
			}
			if mode == "untyped cookie" {
				require.ErrorContains(t, expr.RunInvalidDSL(t, design), "separate typed handshake")
				return
			}
			root := expr.RunDSL(t, design)
			e := root.API.HTTP.Service("events").Endpoint("watch")
			r := e.Responses[0]
			require.NotNil(t, e.MethodExpr.EffectiveStreamingResult().Find("session"))
			require.True(t, e.MethodExpr.EffectiveStreamingResult().IsRequired("session"))
			if mode == "typed cookie" {
				require.NotNil(t, r.Handshake)
				require.Equal(t, expr.Empty, r.Body.Type)
				require.True(t, r.Cookies[0].IsRequired("session"))
				require.ElementsMatch(t, []string{"session"}, r.RequiredFields(""))
			} else {
				require.Nil(t, r.Handshake)
				require.Empty(t, r.Cookies)
			}
		})
	}
}
