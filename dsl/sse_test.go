package dsl_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestServerSentEventsDSL(t *testing.T) {
	root := expr.RunDSL(t, func() {
		notification := Type("Notification", func() {
			Attribute("message", String)
			Attribute("timestamp", String)
			Attribute("kind", String)
			Attribute("retry", Int)
		})

		Service("events", func() {
			Method("stream", func() {
				Payload(func() {
					Attribute("id", String)
				})
				StreamingResult(notification)
				HTTP(func() {
					POST("/events")
					ServerSentEvents(func() {
						SSERequestID("id")
						SSEEventData("message")
						SSEEventID("timestamp")
						SSEEventType("kind")
						SSEEventRetry("retry")
					})
				})
			})
			Method("raw", func() {
				StreamingResult(notification)
				HTTP(func() {
					POST("/raw")
					ServerSentEvents()
				})
			})
			Method("named", func() {
				StreamingResult(notification)
				HTTP(func() {
					POST("/named")
					ServerSentEvents("message")
				})
			})
		})
	})

	httpSvc := root.API.HTTP.Service("events")
	require.NotNil(t, httpSvc)

	stream := httpSvc.Endpoint("stream")
	require.NotNil(t, stream)
	require.NotNil(t, stream.SSE)
	require.Equal(t, "id", stream.SSE.RequestIDField)
	require.Equal(t, "message", stream.SSE.DataField)
	require.Equal(t, "timestamp", stream.SSE.IDField)
	require.Equal(t, "kind", stream.SSE.EventField)
	require.Equal(t, "retry", stream.SSE.RetryField)

	raw := httpSvc.Endpoint("raw")
	require.NotNil(t, raw)
	require.NotNil(t, raw.SSE)
	require.Empty(t, raw.SSE.DataField)
	require.Empty(t, raw.SSE.IDField)
	require.Empty(t, raw.SSE.EventField)
	require.Empty(t, raw.SSE.RetryField)

	named := httpSvc.Endpoint("named")
	require.NotNil(t, named)
	require.NotNil(t, named.SSE)
	require.Equal(t, "message", named.SSE.DataField)
}

func TestServerSentEventsDefaultsDSL(t *testing.T) {
	root := expr.RunDSL(t, func() {
		notification := Type("Notification", func() {
			Attribute("message", String)
		})

		API("sse_api", func() {
			HTTP(func() {
				ServerSentEvents()
			})
		})

		Service("svclevel", func() {
			HTTP(func() {
				ServerSentEvents()
			})
			Method("watch", func() {
				StreamingResult(notification)
				HTTP(func() {
					POST("/watch")
				})
			})
		})

		Service("apilevel", func() {
			Method("follow", func() {
				StreamingResult(notification)
				HTTP(func() {
					POST("/follow")
				})
			})
		})
	})

	require.NotNil(t, root.API.HTTP.SSE)

	svc := root.API.HTTP.Service("svclevel")
	require.NotNil(t, svc)
	require.NotNil(t, svc.SSE)
	watch := svc.Endpoint("watch")
	require.NotNil(t, watch)
	require.Same(t, svc.SSE, watch.SSE)

	follow := root.API.HTTP.Service("apilevel").Endpoint("follow")
	require.NotNil(t, follow)
	require.Same(t, root.API.HTTP.SSE, follow.SSE)
}

func TestServerSentEventsJSONRPCNotificationMethodDSL(t *testing.T) {
	root := expr.RunDSL(t, func() {
		Service("clock", func() {
			JSONRPC(func() {
				POST("/rpc")
			})
			Method("tick", func() {
				Payload(func() {
					ID("id", String)
				})
				StreamingResult(func() {
					Attribute("value", String)
				})
				JSONRPC(func() {
					ServerSentEvents(func() {
						SSENotificationMethod("clock.tick")
					})
				})
			})
		})
	})

	endpoint := root.API.JSONRPC.Service("clock").Endpoint("tick")
	require.NotNil(t, endpoint)
	require.NotNil(t, endpoint.SSE)
	require.Equal(t, "clock.tick", endpoint.SSE.NotificationMethod)
}

func TestServerSentEventsDSLErrors(t *testing.T) {
	cases := []struct {
		name    string
		dsl     func()
		wantErr string
	}{
		{
			name: "sse without streaming result",
			dsl: func() {
				Service("svc", func() {
					Method("show", func() {
						Result(func() {
							Attribute("value", String)
						})
						HTTP(func() {
							GET("/show")
							ServerSentEvents()
						})
					})
				})
			},
			wantErr: "Server-Sent Events can only be used with endpoints that have a streaming result or mixed results",
		},
		{
			name: "sse with bidirectional streaming",
			dsl: func() {
				Service("svc", func() {
					Method("chat", func() {
						StreamingPayload(func() {
							Attribute("message", String)
						})
						StreamingResult(func() {
							Attribute("reply", String)
						})
						HTTP(func() {
							POST("/chat")
							ServerSentEvents()
						})
					})
				})
			},
			wantErr: "Server-Sent Events cannot be used with bidirectional streaming endpoints",
		},
		{
			name: "sse with client streaming",
			dsl: func() {
				Service("svc", func() {
					Method("upload", func() {
						StreamingPayload(func() {
							Attribute("chunk", Bytes)
						})
						HTTP(func() {
							POST("/upload")
							ServerSentEvents()
						})
					})
				})
			},
			wantErr: "Server-Sent Events cannot be used with client-to-server streaming endpoints",
		},
		{
			name: "event id attribute is not a string",
			dsl: func() {
				Service("svc", func() {
					Method("watch", func() {
						StreamingResult(func() {
							Attribute("seq", Int)
						})
						HTTP(func() {
							POST("/watch")
							ServerSentEvents(func() {
								SSEEventID("seq")
							})
						})
					})
				})
			},
			wantErr: `cannot use "seq" for SSE event id field: attribute type must be one of`,
		},
		{
			name: "event retry attribute is not an integer",
			dsl: func() {
				Service("svc", func() {
					Method("watch", func() {
						StreamingResult(func() {
							Attribute("message", String)
						})
						HTTP(func() {
							POST("/watch")
							ServerSentEvents(func() {
								SSEEventRetry("message")
							})
						})
					})
				})
			},
			wantErr: `cannot use "message" for SSE event retry field: attribute type must be one of`,
		},
		{
			name: "event data attribute not found",
			dsl: func() {
				Service("svc", func() {
					Method("watch", func() {
						StreamingResult(func() {
							Attribute("message", String)
						})
						HTTP(func() {
							POST("/watch")
							ServerSentEvents(func() {
								SSEEventData("missing")
							})
						})
					})
				})
			},
			wantErr: `cannot use "missing" for SSE event data field: attribute not found in result type`,
		},
		{
			name: "request id attribute is not a string",
			dsl: func() {
				Service("svc", func() {
					Method("watch", func() {
						Payload(func() {
							Attribute("id", Int)
						})
						StreamingResult(func() {
							Attribute("message", String)
						})
						HTTP(func() {
							POST("/watch")
							ServerSentEvents(func() {
								SSERequestID("id")
							})
						})
					})
				})
			},
			wantErr: `cannot use "id" for SSE request ID field: attribute type must be one of`,
		},
		{
			name: "empty event data field name",
			dsl: func() {
				Service("svc", func() {
					Method("watch", func() {
						StreamingResult(func() {
							Attribute("message", String)
						})
						HTTP(func() {
							POST("/watch")
							ServerSentEvents(func() {
								SSEEventData("")
							})
						})
					})
				})
			},
			wantErr: "data field name cannot be empty",
		},
		{
			name: "event id outside server sent events",
			dsl: func() {
				Service("svc", func() {
					Method("watch", func() {
						StreamingResult(func() {
							Attribute("message", String)
						})
						HTTP(func() {
							POST("/watch")
							SSEEventID("message")
						})
					})
				})
			},
			wantErr: "invalid use of SSEEventID",
		},
		{
			name: "invalid server sent events argument",
			dsl: func() {
				Service("svc", func() {
					Method("watch", func() {
						StreamingResult(func() {
							Attribute("message", String)
						})
						HTTP(func() {
							POST("/watch")
							ServerSentEvents(42)
						})
					})
				})
			},
			wantErr: "cannot use 42 (type int) as type function or string",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, tc.dsl)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.wantErr)
		})
	}
}
