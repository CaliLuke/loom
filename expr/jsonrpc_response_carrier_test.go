package expr_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestJSONRPCResponseCarrierAdmission(t *testing.T) {
	for _, carrier := range []string{"http", "sse", "websocket"} {
		for _, scope := range []string{"success", "method error", "service error", "API error"} {
			for _, cookie := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/cookie=%t", carrier, scope, cookie), func(t *testing.T) {
					err := expr.RunInvalidDSL(t, responseCarrierDSL(carrier, scope, cookie, true))
					require.ErrorContains(t, err, "JSON-RPC response")
					require.ErrorContains(t, err, "result or error.data")
				})
			}
		}
	}
}

func TestJSONRPCResponseCarrierControls(t *testing.T) {
	for _, scope := range []string{"success", "method error", "service error", "API error"} {
		for _, cookie := range []bool{false, true} {
			t.Run(fmt.Sprintf("HTTP/%s/cookie=%t", scope, cookie), func(t *testing.T) {
				expr.RunDSL(t, responseCarrierDSL("http", scope, cookie, false))
			})
		}
	}
	for _, cookie := range []bool{false, true} {
		t.Run(fmt.Sprintf("RPC request/cookie=%t", cookie), func(t *testing.T) {
			expr.RunDSL(t, responseCarrierDSL("http", "request", cookie, true))
		})
	}
}

func responseCarrierDSL(carrier, scope string, cookie, rpc bool) func() {
	return func() {
		mapping := func() {
			if cookie {
				Cookie("value:session")
			} else {
				Header("value:X-Value")
			}
		}
		response := func() {
			Response("failed", StatusBadRequest, mapping)
		}
		transport := func(fn func()) {
			HTTP(fn)
		}
		if rpc {
			transport = JSONRPC
		}
		API("carrier", func() {
			if scope == "API error" {
				Error("failed", func() {
					Attribute("value", String)
				})
				transport(response)
			}
		})
		Service("carrier", func() {
			if scope == "service error" {
				Error("failed", func() {
					Attribute("value", String)
				})
			}
			transport(func() {
				if rpc {
					if carrier == "websocket" {
						GET("/rpc")
					} else {
						POST("/rpc")
					}
				}
				if scope == "service error" {
					response()
				}
			})
			Method("call", func() {
				payload := func() {
					Attribute("value", String)
				}
				if carrier == "websocket" {
					StreamingPayload(payload)
				} else {
					Payload(payload)
				}
				if carrier == "http" {
					Result(payload)
				} else {
					StreamingResult(payload)
				}
				if scope == "method error" {
					Error("failed", payload)
				} else if scope == "API error" {
					Error("failed", payload)
				}
				transport(func() {
					if !rpc {
						POST("/http")
					}
					if carrier == "sse" {
						ServerSentEvents()
					}
					switch scope {
					case "success":
						Response(StatusOK, mapping)
					case "method error":
						response()
					case "request":
						mapping()
					}
				})
			})
		})
	}
}
