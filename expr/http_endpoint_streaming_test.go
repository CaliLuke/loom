package expr_test

import (
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/expr/testdata"
)

func TestHTTPStreamingSessionSecurityRequestBodyInference(t *testing.T) {
	cases := map[string]struct {
		DSL                func()
		ExpectedRoute      string
		ExpectedParamField string
		ExpectStreamingIn  bool
	}{
		"server stream path param": {
			DSL:                websocketSessionCookiePathStreamDSL,
			ExpectedRoute:      "/ws/projects/{project_id}",
			ExpectedParamField: "project_id",
		},
		"bidirectional path param": {
			DSL:                websocketSessionCookiePathBidirectionalDSL,
			ExpectedRoute:      "/ws/projects/{project_id}",
			ExpectedParamField: "project_id",
			ExpectStreamingIn:  true,
		},
		"server stream query param": {
			DSL:                websocketSessionCookieQueryStreamDSL,
			ExpectedRoute:      "/ws/projects",
			ExpectedParamField: "project_id",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			root := expr.RunDSL(t, tc.DSL)
			e := root.API.HTTP.Services[0].HTTPEndpoints[0]

			if e.Body == nil || e.Body.Type != expr.Empty {
				t.Fatalf("expected websocket handshake request body to be empty, got %#v", e.Body)
			}
			if e.Cookies.Find("browser_session_cookie") == nil {
				t.Fatalf("expected inferred browser_session_cookie cookie mapping")
			}
			if e.Params.Find(tc.ExpectedParamField) == nil {
				t.Fatalf("expected %q handshake param mapping", tc.ExpectedParamField)
			}
			if len(e.Routes) != 1 || e.Routes[0].Path != tc.ExpectedRoute {
				t.Fatalf("got route %#v, expected %q", e.Routes, tc.ExpectedRoute)
			}
			if tc.ExpectStreamingIn && (e.StreamingBody == nil || e.StreamingBody.Type == expr.Empty) {
				t.Fatalf("expected bidirectional websocket endpoint to retain streaming body")
			}
		})
	}
}

func TestHTTPMixedTransportGraphFinalization(t *testing.T) {
	root := expr.RunDSL(t, mixedTransportIRDSL)

	requireHTTP := func() *expr.HTTPEndpointExpr {
		t.Helper()
		for _, svc := range root.API.HTTP.Services {
			if svc.Name() == "PlainHTTP" {
				return svc.HTTPEndpoints[0]
			}
		}
		t.Fatal("plain HTTP endpoint not found")
		return nil
	}
	requireJSONRPC := func(service string) *expr.HTTPEndpointExpr {
		t.Helper()
		for _, svc := range root.API.JSONRPC.Services {
			if svc.Name() == service {
				return svc.HTTPEndpoints[0]
			}
		}
		t.Fatalf("jsonrpc endpoint for service %q not found", service)
		return nil
	}

	plain := requireHTTP()
	if plain.Body == nil || plain.Body.Type == expr.Empty {
		t.Fatalf("expected plain HTTP endpoint to retain request body")
	}
	if len(plain.Routes) != 1 || plain.Routes[0].Method != "POST" || plain.Routes[0].Path != "/plain/{id}" {
		t.Fatalf("unexpected plain HTTP routes: %#v", plain.Routes)
	}

	post := requireJSONRPC("RPCPost")
	if post.PayloadIDAttribute != "id" || post.ResultIDAttribute != "id" {
		t.Fatalf("expected JSON-RPC POST endpoint to project request/result ids, got payload=%q result=%q", post.PayloadIDAttribute, post.ResultIDAttribute)
	}
	if post.MethodExpr.IsStreaming() {
		t.Fatalf("expected JSON-RPC POST endpoint to remain unary")
	}

	sse := requireJSONRPC("RPCSSE")
	if !sse.MethodExpr.HasMixedResults() {
		t.Fatalf("expected JSON-RPC SSE endpoint to retain mixed results")
	}
	if sse.SSE == nil {
		t.Fatalf("expected JSON-RPC SSE endpoint to finalize SSE transport")
	}
	if len(sse.Routes) != 1 || sse.Routes[0].Method != "POST" || sse.Routes[0].Path != "/events" {
		t.Fatalf("unexpected JSON-RPC SSE routes: %#v", sse.Routes)
	}

	ws := requireJSONRPC("RPCWebSocket")
	if ws.Body == nil || ws.Body.Type == expr.Empty {
		t.Fatalf("expected JSON-RPC websocket endpoint to retain migrated request body")
	}
	if ws.StreamingBody == nil || ws.StreamingBody.Type == expr.Empty {
		t.Fatalf("expected JSON-RPC websocket endpoint to keep streaming payload body")
	}
	if ws.Body != ws.StreamingBody {
		t.Fatalf("expected JSON-RPC websocket endpoint to reuse streaming body for migrated payload")
	}
	if len(ws.Routes) != 1 || ws.Routes[0].Method != "GET" || ws.Routes[0].Path != "/ws" {
		t.Fatalf("unexpected JSON-RPC websocket routes: %#v", ws.Routes)
	}
}

func TestHTTPAuthorizationMapping(t *testing.T) {
	cases := []struct {
		Name           string
		DSL            func()
		ExpectedHeader string
	}{{
		Name:           "explicit",
		DSL:            testdata.ExplicitAuthHeaderDSL,
		ExpectedHeader: "token",
	}, {
		Name:           "implicit",
		DSL:            testdata.ImplicitAuthHeaderDSL,
		ExpectedHeader: "Authorization",
	},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			root := expr.RunDSL(t, tc.DSL)
			e := root.API.HTTP.Services[0].HTTPEndpoints[0]
			if e.Headers == nil {
				t.Errorf("got endpoint without header, expected endpoint with HTTP header")
				return
			}
			if len(*expr.AsObject(e.Headers.Type)) != 1 {
				t.Errorf("got %d, expected 1 attribute in endpoint headers", len(*expr.AsObject(e.Headers.Type)))
				return
			}
			n := e.Headers.ElemName("token")
			if n != tc.ExpectedHeader {
				t.Errorf("got %q, expected %q attribute in endpoint headers", n, tc.ExpectedHeader)
			}
		})
	}
}

func websocketSessionCookieAuth() any {
	browserSession := APIKeySecurity("browser_session_cookie", func() {
		Description("Browser session cookie")
	})
	return SessionAuth("app_session", func() {
		CookieTransport(browserSession, "", func() {
			CookieName("__Host-ak_session")
		})
	})
}

var inconsistentRouteParamsDSL = func() {
	Service("RouteMismatch", func() {
		Method("Show", func() {
			Payload(func() {
				Attribute("id", String)
				Attribute("slug", String)
			})
			Result(String)
			HTTP(func() {
				GET("/{id}")
				GET("/{slug}")
			})
		})
	})
}

var mixedTransportIRDSL = func() {
	API("mixed-transport-ir", func() {
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
				Required("filter")
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
				Required("message")
			})
			StreamingResult(func() {
				Attribute("event", String)
			})
			JSONRPC(func() {})
		})
	})
}

var websocketSessionCookiePathStreamDSL = websocketSessionCookieStreamingDSL(
	"WebSocketSessionCookiePathStream",
	"/ws/projects/{project_id}",
	false,
)

var websocketSessionCookiePathBidirectionalDSL = websocketSessionCookieStreamingDSL(
	"WebSocketSessionCookiePathBidirectional",
	"/ws/projects/{project_id}",
	true,
)

var websocketSessionCookieQueryStreamDSL = websocketSessionCookieStreamingDSL(
	"WebSocketSessionCookieQueryStream",
	"/ws/projects",
	false,
)

func websocketSessionCookieStreamingDSL(serviceName string, route string, bidirectional bool) func() {
	return func() {
		appSession := websocketSessionCookieAuth()
		Service(serviceName, func() {
			Method("connect", func() {
				SessionSecurity(appSession)
				Payload(func() {
					Attribute("project_id", String)
					Required("project_id")
				})
				if bidirectional {
					StreamingPayload(String)
				}
				StreamingResult(func() {
					Attribute("event", String)
					Required("event")
				})
				HTTP(func() {
					GET(route)
					Param("project_id")
					Response(StatusOK)
				})
			})
		})
	}
}

var mixedResultsWithoutSSEDsl = func() {
	Service("MixedResults", func() {
		Method("Watch", func() {
			Result(func() {
				Attribute("done", Boolean)
			})
			StreamingResult(func() {
				Attribute("event", String)
			})
			HTTP(func() {
				GET("/")
			})
		})
	})
}

var sseWithClientStreamDsl = func() {
	Service("ClientStreamSSE", func() {
		Method("Watch", func() {
			StreamingPayload(func() {
				Attribute("value", String)
			})
			Result(func() {
				Attribute("done", Boolean)
			})
			HTTP(func() {
				GET("/")
				ServerSentEvents()
			})
		})
	})
}

var sseWithBidirectionalStreamDsl = func() {
	Service("BidirectionalStreamSSE", func() {
		Method("Watch", func() {
			StreamingPayload(func() {
				Attribute("value", String)
			})
			StreamingResult(func() {
				Attribute("event", String)
			})
			HTTP(func() {
				GET("/")
				ServerSentEvents()
			})
		})
	})
}

var serviceLevelSSEInheritanceDSL = func() {
	Service("ServiceLevelSSE", func() {
		HTTP(func() {
			ServerSentEvents("event")
		})
		Method("watch", func() {
			StreamingResult(func() {
				Attribute("event", String)
			})
			HTTP(func() {
				GET("/watch")
			})
		})
	})
}

var apiLevelSSEInheritanceDSL = func() {
	API("APISSE", func() {
		HTTP(func() {
			ServerSentEvents("event")
		})
	})
	Service("APILevelSSE", func() {
		Method("watch", func() {
			StreamingResult(func() {
				Attribute("event", String)
			})
			HTTP(func() {
				GET("/watch")
			})
		})
	})
}

var serviceLevelHTTPErrorsDSL = func() {
	Service("ServiceLevelErrors", func() {
		Error("bad_request")
		HTTP(func() {
			Response("bad_request", StatusBadRequest)
		})
		Method("show", func() {
			Error("bad_request")
			Result(String)
			HTTP(func() {
				GET("/")
			})
		})
	})
}

var apiLevelHTTPErrorsDSL = func() {
	API("APILevelErrors", func() {
		Error("bad_request")
		HTTP(func() {
			Response("bad_request", StatusBadRequest)
		})
	})
	Service("APIInheritedErrors", func() {
		Method("show", func() {
			Error("bad_request")
			Result(String)
			HTTP(func() {
				GET("/")
			})
		})
	})
}

var websocketRouteMethodCoercionDSL = func() {
	Service("WebSocketRouteMethodCoercion", func() {
		Method("stream", func() {
			StreamingResult(String)
			HTTP(func() {
				POST("/stream")
			})
		})
	})
}

var defaultNoContentResponseDSL = func() {
	Service("DefaultNoContent", func() {
		Method("create", func() {
			HTTP(func() {
				POST("/")
			})
		})
	})
}

var defaultRedirectResponseDSL = func() {
	Service("DefaultRedirect", func() {
		Method("show", func() {
			HTTP(func() {
				GET("/")
				Redirect("/dest", StatusMovedPermanently)
			})
		})
	})
}

var jsonrpcWebSocketPayloadMigrationDSL = func() {
	Service("JSONRPCPayloadMigration", func() {
		Method("stream", func() {
			Payload(func() {
				ID("id", String)
				Attribute("message", String)
				Required("id")
			})
			StreamingResult(func() {
				Attribute("event", String)
			})
			JSONRPC(func() {})
		})
		JSONRPC(func() {
			GET("/rpc")
		})
	})
}

var sessionCookieMappingDSL = func() {
	var browserSession = APIKeySecurity("browser_session_key")
	var appSession = SessionAuth("app_session", func() {
		CookieTransport(browserSession, "", func() {
			CookieName("__Host-browser_session")
		})
	})
	Service("SessionCookieMapping", func() {
		Method("show", func() {
			SessionSecurity(appSession)
			Payload(func() {
				Attribute("message", String)
			})
			Result(String)
			HTTP(func() {
				GET("/")
			})
		})
	})
}

var jsonrpcEndpointIDProjectionDSL = func() {
	Service("JSONRPCEndpointIDProjection", func() {
		Method("show", func() {
			Payload(func() {
				ID("id", String)
				Attribute("query", String)
				Required("id")
			})
			Result(func() {
				ID("id", String)
				Attribute("value", String)
				Required("id")
			})
			JSONRPC(func() {})
		})
		JSONRPC(func() {
			Path("/rpc")
		})
	})
}

var allTaggedResponsesDSL = func() {
	Service("AllTaggedResponses", func() {
		Method("show", func() {
			Result(func() {
				Attribute("kind", String)
			})
			HTTP(func() {
				GET("/")
				Response(StatusOK, func() {
					Tag("kind", "ok")
				})
				Response(StatusAccepted, func() {
					Tag("kind", "accepted")
				})
			})
		})
	})
}

var taggedPrimitiveResultDSL = func() {
	Service("TaggedPrimitiveResult", func() {
		Method("show", func() {
			Result(String)
			HTTP(func() {
				GET("/")
				Response(StatusOK)
				Response(StatusAccepted, func() {
					Tag("kind", "accepted")
				})
			})
		})
	})
}

var nonStreamingSSEDsl = func() {
	Service("NonStreamingSSE", func() {
		Method("show", func() {
			Result(func() {
				Attribute("event", String)
			})
			HTTP(func() {
				GET("/")
				ServerSentEvents("event")
			})
		})
	})
}

var jsonrpcResultIDWithoutRequestIDDSL = func() {
	Service("JSONRPCIDValidation", func() {
		Method("show", func() {
			Payload(func() {
				Attribute("query", String)
			})
			Result(func() {
				ID("id", String)
				Attribute("value", String)
				Required("id")
			})
			JSONRPC(func() {})
		})
		JSONRPC(func() {
			Path("/rpc")
		})
	})
}

var redirectWithMismatchedResponseDSL = func() {
	Service("RedirectMismatch", func() {
		Method("show", func() {
			HTTP(func() {
				GET("/")
				Redirect("/dest", StatusMovedPermanently)
				Response(StatusOK)
			})
		})
	})
}
