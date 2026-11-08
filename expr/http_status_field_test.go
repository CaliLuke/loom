package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestStatusFieldWireBinding(t *testing.T) {
	t.Run("implicit body and request isolation", func(t *testing.T) {
		root := expr.RunDSL(t, func() {
			value := Type("Value", func() {
				Attribute("status", Int)
				Attribute("message", String)
				Required("status", "message")
			})
			Service("binding", func() {
				Method("call", func() {
					Payload(value)
					Result(value)
					HTTP(func() {
						POST("/")
						Response(StatusOK, func() {
							StatusField("status")
						})
					})
				})
			})
		})
		e := root.API.HTTP.Service("binding").Endpoint("call")
		require.NotNil(t, e.Body.Find("status"))
		require.Nil(t, e.Responses[0].Body.Find("status"))
		require.NotNil(t, e.Responses[0].Body.Find("message"))
		require.True(t, e.MethodExpr.Payload.IsRequired("status"))
		require.True(t, e.MethodExpr.Result.IsRequired("status"))
	})
	t.Run("explicit selected native error body", func(t *testing.T) {
		root := expr.RunDSL(t, func() {
			failure := Type("Failure", func() {
				Attribute("status", Int)
				Attribute("message", String)
				Required("status", "message")
			})
			Service("binding", func() {
				Method("call", func() {
					Error("failure", failure)
					HTTP(func() {
						POST("/")
						Response("failure", StatusBadRequest, func() {
							StatusField("status")
							Body("message")
						})
					})
				})
			})
		})
		e := root.API.HTTP.Service("binding").Endpoint("call").HTTPErrors[0]
		require.Equal(t, expr.String, e.Response.Body.Type)
		require.NotNil(t, e.AttributeExpr.Find("status"))
	})
	for _, reverse := range []bool{false, true} {
		name := "branch requirements"
		if reverse {
			name += " reversed"
		}
		t.Run(name, func(t *testing.T) {
			var declared expr.UserType
			root := expr.RunDSL(t, func() {
				declared = Type("BranchValue", func() {
					Attribute("status", Int)
					Attribute("left", String)
					Attribute("right", String)
					Attribute("trace", String)
					Attribute("session", String)
					Required("status", "left", "right", "trace", "session")
				})
				Service("branches", func() {
					Method("call", func() {
						Payload(declared)
						Result(declared)
						HTTP(func() {
							POST("/")
							left := func() {
								Response(StatusOK, func() {
									StatusField("status")
									Header("trace:X-Trace")
									Body("left")
								})
							}
							right := func() {
								Response(StatusCreated, func() {
									StatusField("status")
									Cookie("session")
									Body("right")
								})
							}
							if reverse {
								right()
								left()
							} else {
								left()
								right()
							}
						})
					})
				})
			})
			e := root.API.HTTP.Service("branches").Endpoint("call")
			for _, name := range []string{"left", "right", "trace", "session"} {
				require.False(t, e.MethodExpr.Result.IsRequired(name), name)
				require.True(t, e.MethodExpr.Payload.IsRequired(name), name)
				require.True(t, declared.Attribute().IsRequired(name), name)
			}
			require.True(t, e.MethodExpr.Result.IsRequired("status"))
			for _, r := range e.Responses {
				want := []string{"status", "right", "session"}
				if r.Status.Code == 200 {
					want = []string{"status", "left", "trace"}
					require.True(t, r.Headers.IsRequired("trace"))
				} else {
					require.True(t, r.Cookies[0].IsRequired("session"))
				}
				require.ElementsMatch(t, want, r.RequiredFields(""))
				copy := r.RequiredFields("")
				copy[0] = "mutated"
				require.ElementsMatch(t, want, r.RequiredFields(""))
			}
		})
	}

	t.Run("wire-local requirements", func(t *testing.T) {
		root := expr.RunDSL(t, func() {
			value := Type("WireRequired", func() {
				Attribute("status", Int)
				Attribute("message", String)
				Attribute("trace", String)
				Required("status")
			})
			Service("wire", func() {
				Method("call", func() {
					Result(value)
					HTTP(func() {
						GET("/")
						Response(StatusOK, func() {
							StatusField("status")
							Headers(func() {
								Header("trace:X-Trace")
								Required("trace")
							})
							Body(func() {
								Attribute("message")
								Required("message")
							})
						})
					})
				})
			})
		})
		e := root.API.HTTP.Service("wire").Endpoint("call")
		require.ElementsMatch(t, []string{"status", "trace", "message"}, e.Responses[0].RequiredFields(""))
		require.False(t, e.MethodExpr.Result.IsRequired("trace"))
		require.False(t, e.MethodExpr.Result.IsRequired("message"))
	})

	t.Run("view survival", func(t *testing.T) {
		for _, missing := range []bool{false, true} {
			design := func() {
				viewed := ResultType("application/vnd.status+json", func() {
					Attributes(func() {
						Attribute("status", Int)
						Attribute("message", String)
						Required("status", "message")
					})
					View("default", func() {
						Attribute("status")
						Attribute("message")
					})
					View("summary", func() {
						if !missing {
							Attribute("status")
						}
						Attribute("message")
					})
				})
				Service("views", func() {
					Method("call", func() {
						Result(viewed)
						HTTP(func() {
							GET("/")
							Response(StatusOK, func() {
								StatusField("status")
							})
						})
					})
				})
			}
			if missing {
				err := expr.RunInvalidDSL(t, design)
				require.ErrorContains(t, err, "StatusField")
				require.ErrorContains(t, err, "summary")
				continue
			}
			root := expr.RunDSL(t, design)
			r := root.API.HTTP.Service("views").Endpoint("call").Responses[0]
			body := r.Body.Type.(*expr.ResultTypeExpr)
			for _, view := range []string{"default", "summary"} {
				require.Nil(t, body.View(view).Find("status"))
				require.ElementsMatch(t, []string{"status", "message"}, r.RequiredFields(view))
			}
		}
	})
	t.Run("media and distinct native values", func(t *testing.T) {
		root := expr.RunDSL(t, func() {
			doc := Type("Doc", func() {
				Attribute("text", String)
				Required("text")
			})
			carrier := Type("Carrier", func() {
				Attribute("status", Int)
				OneOf("data", func() {
					Attribute("doc", doc)
					Attribute("raw", Bytes)
				})
				Required("status", "data")
			})
			hints := Type("Hints", func() {
				Attribute("status", Int)
				Attribute("link", String)
				Required("status", "link")
			})
			meta := Type("Meta", func() {
				Attribute("status", Int)
				Attribute("trace", String)
				Required("status", "trace")
			})
			Service("occurrences", func() {
				Method("call", func() {
					Result(carrier)
					HTTP(func() {
						GET("/")
						Response(StatusOK, func() {
							StatusField("status")
							Body("data")
							MediaType("application/json", "doc")
							MediaType("application/octet-stream", "raw")
						})
						Informational("hints", StatusEarlyHints, hints, func() {
							StatusField("status")
							Header("link:Link")
						})
					})
				})
				Method("stream", func() {
					StreamingResult(String)
					HTTP(func() {
						GET("/stream")
						ServerSentEvents()
						Response(Status2XX, meta, func() {
							StatusField("status")
							Header("trace:X-Trace")
						})
					})
				})
			})
		})
		call := root.API.HTTP.Service("occurrences").Endpoint("call")
		require.ElementsMatch(t, []string{"status", "data"}, call.Responses[0].RequiredFields(""))
		require.Len(t, call.Responses[0].MediaTypes, 2)
		require.NotNil(t, expr.HTTPMediaBranch(call.Responses[0].Body, "doc"))
		require.NotNil(t, expr.HTTPMediaBranch(call.Responses[0].Body, "raw"))
		require.ElementsMatch(t, []string{"status", "link"}, call.Informational[0].RequiredFields(""))
		require.True(t, call.Informational[0].Headers.IsRequired("link"))
		stream := root.API.HTTP.Service("occurrences").Endpoint("stream")
		require.Equal(t, expr.String, stream.MethodExpr.Result.Type)
		require.Equal(t, expr.Empty, stream.Responses[0].Body.Type)
		require.ElementsMatch(t, []string{"status", "trace"}, stream.Responses[0].RequiredFields(""))
		require.True(t, stream.Responses[0].Headers.IsRequired("trace"))
	})

	t.Run("handshake mappings do not consume event fields", func(t *testing.T) {
		var event expr.UserType
		root := expr.RunDSL(t, func() {
			event = Type("Event", func() {
				Attribute("status", String)
				Attribute("trace", String)
				Attribute("session", String)
				Required("status", "trace", "session")
			})
			meta := Type("Handshake", func() {
				Attribute("status", Int)
				Attribute("trace", String)
				Attribute("session", String)
				Required("status", "trace", "session")
			})
			Service("events", func() {
				Method("watch", func() {
					StreamingResult(event)
					HTTP(func() {
						GET("/")
						ServerSentEvents()
						Response(Status2XX, meta, func() {
							StatusField("status")
							Header("trace:X-Trace")
							Cookie("session")
						})
					})
				})
			})
		})
		e := root.API.HTTP.Service("events").Endpoint("watch")
		r := e.Responses[0]
		require.Equal(t, expr.Empty, r.Body.Type)
		require.Equal(t, expr.Int, r.Handshake.Find("status").Type)
		require.ElementsMatch(t, []string{"status", "trace", "session"}, r.RequiredFields(""))
		require.True(t, r.Headers.IsRequired("trace"))
		require.True(t, r.Cookies[0].IsRequired("session"))
		for _, source := range []*expr.AttributeExpr{e.MethodExpr.EffectiveStreamingResult(), event.Attribute()} {
			for _, name := range []string{"status", "trace", "session"} {
				require.NotNil(t, source.Find(name), name)
				require.True(t, source.IsRequired(name), name)
			}
			require.Equal(t, expr.String, source.Find("status").Type)
		}
	})

	t.Run("SSE status requires separate handshake", func(t *testing.T) {
		err := expr.RunInvalidDSL(t, func() {
			Service("events", func() {
				Method("watch", func() {
					StreamingResult(func() {
						Attribute("status", Int)
						Required("status")
					})
					HTTP(func() {
						GET("/")
						ServerSentEvents()
						Response(Status2XX, func() {
							StatusField("status")
						})
					})
				})
			})
		})
		require.ErrorContains(t, err, "handshake")
	})
	t.Run("per-key error identity", func(t *testing.T) {
		root := expr.RunDSL(t, func() {
			failure := Type("FailureIdentity", func() {
				ErrorName("name", String)
				Attribute("status", Int)
				Attribute("message", String)
				Required("name", "status", "message")
			})
			Service("errors", func() {
				Method("call", func() {
					Error("client", failure)
					Error("other", failure)
					HTTP(func() {
						GET("/")
						NoSuccessResponse()
						Response("client", Status4XX, func() {
							StatusField("status")
							ErrorNameFromStatus()
						})
						Response("other", StatusFallback, func() {
							StatusField("status")
							ErrorNameFromStatus()
						})
					})
				})
			})
		})
		for _, er := range root.API.HTTP.Service("errors").Endpoint("call").HTTPErrors {
			require.Equal(t, "name", er.Response.StatusErrorName)
			require.Nil(t, er.Response.Body.Find("name"))
			require.Nil(t, er.Response.Body.Find("status"))
			require.NotNil(t, er.Response.Body.Find("message"))
		}
	})

	t.Run("explicit body competes", func(t *testing.T) {
		err := expr.RunInvalidDSL(t, func() {
			Service("binding", func() {
				Method("call", func() {
					Result(func() {
						Attribute("status", Int)
						Required("status")
					})
					HTTP(func() {
						GET("/")
						Response(StatusOK, func() {
							StatusField("status")
							Body("status")
						})
					})
				})
			})
		})
		require.ErrorContains(t, err, "StatusField")
		require.ErrorContains(t, err, "body")
	})
}
