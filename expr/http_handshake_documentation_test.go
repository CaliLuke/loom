package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestSSEHandshakeIndependentDocumentation(t *testing.T) {
	for _, mapped := range []bool{false, true} {
		name := "plain names"
		if mapped {
			name = "mapped names"
		}
		t.Run(name, func(t *testing.T) {
			key := func(name string) string {
				if mapped {
					return name + ":event_" + name
				}
				return name
			}
			var event, document expr.UserType
			root := expr.RunDSL(t, func() {
				event = Type("Event", func() {
					for _, name := range []string{"status", "trace", "session"} {
						Attribute(key(name), String, "stream "+name)
						Required(key(name))
					}
				})
				document = Type("DocumentedEvent", func() {
					Description("Documentation event")
					for _, name := range []string{"status", "trace", "session"} {
						Attribute(key(name), String, "documented "+name, func() {
							Example("documented-" + name)
						})
						Required(key(name))
					}
				})
				handshake := Type("Handshake", func() {
					Attribute(key("status"), Int)
					Attribute(key("trace"), String)
					Attribute(key("session"), String)
					Required(key("status"), key("trace"), key("session"))
				})
				Service("events", func() {
					Method("watch", func() {
						StreamingResult(event)
						HTTP(func() {
							GET("/")
							ServerSentEvents()
							Response(Status2XX, handshake, func() {
								StatusField("status")
								Header("trace:X-Trace")
								Cookie("session:subscription")
								OpenAPIBody(document, func() {
									Description("Authored event documentation")
								})
							})
						})
					})
				})
			})
			e := root.API.HTTP.Service("events").Endpoint("watch")
			r := e.Responses[0]
			require.Equal(t, expr.Empty, r.Body.Type)
			require.NotSame(t, r.Handshake, r.OpenAPIBody)
			require.NotSame(t, e.MethodExpr.EffectiveStreamingResult(), r.OpenAPIBody)
			_, status := r.Handshake.FindAttribute("status")
			require.Equal(t, expr.Int, status.Type)
			require.ElementsMatch(t, []string{"status", "trace", "session"}, r.RequiredFields(""))
			require.True(t, r.Headers.IsRequired("trace"))
			require.Equal(t, "X-Trace", r.Headers.ElemName("trace"))
			require.True(t, r.Cookies[0].IsRequired("session"))
			require.Equal(t, "subscription", r.Cookies[0].HTTPName())
			require.Equal(t, "Authored event documentation", r.OpenAPIBody.Description)
			require.Equal(t, "Documentation event", document.Attribute().Description)
			for _, source := range []*expr.AttributeExpr{r.OpenAPIBody, document.Attribute(), e.MethodExpr.EffectiveStreamingResult(), event.Attribute()} {
				for _, name := range []string{"status", "trace", "session"} {
					field := source.Find(key(name))
					require.NotNil(t, field, name)
					require.True(t, source.IsRequired(key(name)), name)
					require.Equal(t, expr.String, field.Type)
					if source == r.OpenAPIBody || source == document.Attribute() {
						require.Equal(t, "documented "+name, field.Description)
						require.Len(t, field.UserExamples, 1)
						require.Equal(t, "documented-"+name, field.UserExamples[0].Value)
					} else {
						require.Equal(t, "stream "+name, field.Description)
					}
				}
			}
		})
	}
}

func TestStatusDocumentationBindingCompetition(t *testing.T) {
	for _, tc := range []struct {
		name      string
		handshake bool
		binding   string
		want      string
	}{
		{"handshake header", true, "header", "explicit header binding"},
		{"handshake cookie", true, "cookie", "explicit cookie binding"},
		{"ordinary object documentation", false, "object", "explicit body binding"},
		{"ordinary selected documentation", false, "selected", "explicit body binding"},
		{"ordinary separate documentation", false, "separate", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			design := func() {
				value := Type("Value", func() {
					Attribute("status", Int)
					Attribute("message", String)
					Required("status")
				})
				Service("bindings", func() {
					Method("read", func() {
						if tc.handshake {
							StreamingResult(String)
						} else {
							Result(value)
						}
						HTTP(func() {
							GET("/")
							mapping := func() {
								StatusField("status")
								switch tc.binding {
								case "header":
									Header("status:X-Status")
								case "cookie":
									Cookie("status:status_code")
								case "object":
									OpenAPIBody(value)
								case "selected":
									OpenAPIBody("status")
								case "separate":
									OpenAPIBody("message")
								}
							}
							if tc.handshake {
								ServerSentEvents()
								Response(Status2XX, value, mapping)
							} else {
								Response(StatusOK, mapping)
							}
						})
					})
				})
			}
			if tc.want != "" {
				require.ErrorContains(t, expr.RunInvalidDSL(t, design), tc.want)
				return
			}
			root := expr.RunDSL(t, design)
			r := root.API.HTTP.Service("bindings").Endpoint("read").Responses[0]
			require.Equal(t, expr.String, r.OpenAPIBody.Type)
			require.Nil(t, r.Body.Find("status"))
		})
	}
}
