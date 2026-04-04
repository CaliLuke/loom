package expr_test

import (
	"strings"
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestHTTPServiceValidate(t *testing.T) {
	cases := []struct {
		Name          string
		DSL           func()
		Error         string
		ContainsError string
	}{
		{"valid jsonrpc websocket", validJSONRPCWebSocketDSL, "", ""},
		{"jsonrpc websocket with headers", jsonrpcWebSocketWithHeadersDSL, "", `JSON-RPC endpoint "method" using WebSocket cannot have header mappings`},
		{"jsonrpc websocket with cookies", jsonrpcWebSocketWithCookiesDSL, "", `JSON-RPC endpoint "method" using WebSocket cannot have cookie mappings`},
		{"jsonrpc websocket with params", jsonrpcWebSocketWithParamsDSL, "", `JSON-RPC endpoint "method" using WebSocket cannot have parameter mappings`},
		{"jsonrpc websocket with all mappings", jsonrpcWebSocketWithAllMappingsDSL, "", `JSON-RPC endpoint "method" using WebSocket cannot have header mappings`},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			if tc.Error == "" && tc.ContainsError == "" {
				expr.RunDSL(t, tc.DSL)
			} else {
				err := expr.RunInvalidDSL(t, tc.DSL)
				if tc.Error != "" {
					if err.Error() != tc.Error {
						t.Errorf("got error %q, expected %q", err.Error(), tc.Error)
					}
				} else if tc.ContainsError != "" {
					if !strings.Contains(err.Error(), tc.ContainsError) {
						t.Errorf("error %q does not contain expected substring %q", err.Error(), tc.ContainsError)
					}
				}
			}
		})
	}
}

func TestHTTPServiceMixedTransportGraph(t *testing.T) {
	root := expr.RunDSL(t, mixedTransportServiceDSL)

	if len(root.API.HTTP.Services) != 1 {
		t.Fatalf("got %d plain HTTP services, expected 1", len(root.API.HTTP.Services))
	}
	if len(root.API.JSONRPC.Services) != 3 {
		t.Fatalf("got %d JSON-RPC services, expected 3", len(root.API.JSONRPC.Services))
	}

	plain := root.API.HTTP.Services[0].HTTPEndpoints[0]
	if len(plain.Routes) != 1 || plain.Routes[0].Method != "POST" || plain.Routes[0].Path != "/plain/{id}" {
		t.Fatalf("unexpected plain HTTP route synthesis: %#v", plain.Routes)
	}

	var sse, ws *expr.HTTPEndpointExpr
	for _, svc := range root.API.JSONRPC.Services {
		switch svc.Name() {
		case "RPCSSE":
			sse = svc.HTTPEndpoints[0]
		case "RPCWebSocket":
			ws = svc.HTTPEndpoints[0]
		}
	}
	if sse == nil || sse.SSE == nil || len(sse.Routes) != 1 || sse.Routes[0].Method != "POST" {
		t.Fatalf("expected JSON-RPC SSE endpoint with synthesized POST route, got %#v", sse)
	}
	if ws == nil || len(ws.Routes) != 1 || ws.Routes[0].Method != "GET" {
		t.Fatalf("expected JSON-RPC websocket endpoint with synthesized GET route, got %#v", ws)
	}
}

// Test DSL functions

var validJSONRPCWebSocketDSL = func() {
	Service("calc", func() {
		JSONRPC(func() {
			GET("/ws")
		})
		Method("method", func() {
			StreamingPayload(func() {
				ID("request_id", String)
				Attribute("data", String)
				Required("request_id")
			})
			StreamingResult(func() {
				ID("response_id", String)
				Attribute("value", String)
				Required("response_id")
			})
			JSONRPC(func() {})
		})
	})
}

var jsonrpcWebSocketWithHeadersDSL = func() {
	Service("calc", func() {
		JSONRPC(func() {
			GET("/ws")
		})
		Method("method", func() {
			StreamingPayload(func() {
				ID("request_id", String)
				Attribute("data", String)
				Required("request_id")
			})
			StreamingResult(func() {
				ID("response_id", String)
				Attribute("value", String)
				Required("response_id")
			})
			JSONRPC(func() {
				Headers(func() {
					Header("X-API-Version", String)
				})
			})
		})
	})
}

var jsonrpcWebSocketWithCookiesDSL = func() {
	Service("calc", func() {
		JSONRPC(func() {
			GET("/ws")
		})
		Method("method", func() {
			StreamingPayload(func() {
				ID("request_id", String)
				Attribute("data", String)
				Required("request_id")
			})
			StreamingResult(func() {
				ID("response_id", String)
				Attribute("value", String)
				Required("response_id")
			})
			JSONRPC(func() {
				Cookie("session", String)
			})
		})
	})
}

var jsonrpcWebSocketWithParamsDSL = func() {
	Service("calc", func() {
		JSONRPC(func() {
			GET("/ws")
		})
		Method("method", func() {
			StreamingPayload(func() {
				ID("request_id", String)
				Attribute("data", String)
				Required("request_id")
			})
			StreamingResult(func() {
				ID("response_id", String)
				Attribute("value", String)
				Required("response_id")
			})
			JSONRPC(func() {
				Params(func() {
					Param("id", String)
				})
			})
		})
	})
}

var jsonrpcWebSocketWithAllMappingsDSL = func() {
	Service("calc", func() {
		JSONRPC(func() {
			GET("/ws")
		})
		Method("method", func() {
			StreamingPayload(func() {
				ID("request_id", String)
				Attribute("data", String)
				Required("request_id")
			})
			StreamingResult(func() {
				ID("response_id", String)
				Attribute("value", String)
				Required("response_id")
			})
			JSONRPC(func() {
				Headers(func() {
					Header("X-API-Version", String)
				})
				Cookie("session", String)
				Params(func() {
					Param("id", String)
				})
			})
		})
	})
}

var mixedTransportServiceDSL = func() {
	API("mixed-transport-service", func() {
		JSONRPC(func() {})
	})

	Service("PlainHTTP", func() {
		Method("create", func() {
			Payload(func() {
				Attribute("id", String)
				Attribute("name", String)
				Required("id", "name")
			})
			Result(String)
			HTTP(func() {
				POST("/plain/{id}")
				Body("name")
			})
		})
	})

	Service("RPCPost", func() {
		JSONRPC(func() {
			POST("/rpc")
		})
		Method("ping", func() {
			Payload(func() {
				ID("id", String)
			})
			Result(func() {
				ID("id", String)
				Attribute("value", String)
			})
			JSONRPC(func() {})
		})
	})

	Service("RPCSSE", func() {
		JSONRPC(func() {
			POST("/events")
		})
		Method("events/stream", func() {
			Payload(func() {
				ID("id", String)
				Attribute("filter", String)
			})
			Result(func() {
				Attribute("accepted", Boolean)
			})
			StreamingResult(func() {
				Attribute("event", String)
			})
			JSONRPC(func() {
				ServerSentEvents()
			})
		})
	})

	Service("RPCWebSocket", func() {
		JSONRPC(func() {
			GET("/ws")
		})
		Method("stream", func() {
			StreamingPayload(func() {
				ID("id", String)
				Attribute("message", String)
			})
			StreamingResult(func() {
				Attribute("event", String)
			})
			JSONRPC(func() {})
		})
	})
}
