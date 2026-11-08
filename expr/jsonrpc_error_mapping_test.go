package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestJSONRPCErrorMappingsDoNotInheritHTTPResponses(t *testing.T) {
	tests := []struct {
		name          string
		explicit      bool
		wantResponses int
	}{
		{name: "unmapped", wantResponses: 0},
		{name: "explicit JSON-RPC mapping", explicit: true, wantResponses: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := expr.RunDSL(t, jsonRPCErrorMappingIsolationDSL(test.explicit))
			require.Equal(t, "/api", root.API.JSONRPC.Path)
			endpoint := root.API.JSONRPC.Service("tools").Endpoint("call")
			require.Len(t, endpoint.HTTPErrors, test.wantResponses)
			if test.explicit {
				require.Equal(t, 4403, endpoint.HTTPErrors[0].Response.JSONRPCError.Code)
			}
		})
	}
}

func jsonRPCErrorMappingIsolationDSL(explicit bool) func() {
	return func() {
		toolError := Type("ToolError", func() {
			ErrorName("name", String)
			Attribute("code", String)
			Attribute("message", String)
			Required("name", "code", "message")
		})
		API("example", func() {
			Error("forbidden", ErrorResult)
			HTTP(func() {
				Path("/api")
				Response("forbidden", StatusForbidden)
			})
		})
		Service("tools", func() {
			Error("forbidden", toolError)
			JSONRPC(func() {
				POST("/rpc")
			})
			Method("call", func() {
				Result(String)
				JSONRPC(func() {
					if explicit {
						Response("forbidden", 4403)
					}
				})
			})
		})
	}
}

func TestJSONRPCErrorCodeOwnership(t *testing.T) {
	cases := []struct {
		name    string
		mapping func()
		want    int
	}{
		{"omitted", func() {
			Response("boom", func() {
			})
		}, -32603},
		{"explicit zero", func() {
			Response("boom", 0)
		}, 0},
		{"callback zero", func() {
			Response("boom", func() {
				Code(0)
			})
		}, 0},
		{"negative", func() {
			Response("boom", -32601)
		}, -32601},
		{"custom", func() {
			Response("boom", 4403)
		}, 4403},
		{"callback custom", func() {
			Response("boom", func() {
				Code(5001)
			})
		}, 5001},
	}
	for _, scope := range []string{"API", "service", "method"} {
		for _, test := range cases {
			t.Run(scope+"/"+test.name, func(t *testing.T) {
				root := expr.RunDSL(t, func() {
					API("rpc", func() {
						Error("boom", ErrorResult)
						if scope == "API" {
							JSONRPC(test.mapping)
						}
					})
					Service("svc", func() {
						Error("boom", ErrorResult)
						JSONRPC(func() {
							POST("/rpc")
							if scope == "service" {
								test.mapping()
							}
						})
						for _, name := range []string{"first", "second"} {
							Method(name, func() {
								Result(String)
								JSONRPC(func() {
									if scope == "method" {
										test.mapping()
									}
								})
							})
						}
					})
				})
				svc := root.API.JSONRPC.Service("svc")
				first := svc.Endpoint("first").HTTPErrors[0]
				second := svc.Endpoint("second").HTTPErrors[0]
				require.NotNil(t, first.ErrorExpr)
				require.NotNil(t, first.Response.JSONRPCError)
				require.NotNil(t, second.Response.JSONRPCError)
				require.Equal(t, test.want, first.Response.JSONRPCError.Code)
				require.Equal(t, test.want, second.Response.JSONRPCError.Code)
				require.NotSame(t, first.Response.JSONRPCError, second.Response.JSONRPCError)
				dup := first.Dup()
				require.NotSame(t, first.Response.JSONRPCError, dup.Response.JSONRPCError)
				dup.Response.JSONRPCError.Code = 9001
				require.Equal(t, test.want, first.Response.JSONRPCError.Code)
			})
		}
	}
}

func TestJSONRPCDualTransportOwnership(t *testing.T) {
	root := expr.RunDSL(t, func() {
		Service("dual", func() {
			Error("boom", ErrorResult)
			JSONRPC(func() {
				POST("/rpc")
			})
			Method("read", func() {
				Result(String)
				HTTP(func() {
					GET("/read")
					Response("boom", 403)
				})
				JSONRPC(func() {
					Response("boom", 4403)
				})
			})
		})
	})
	http := root.API.HTTP.Service("dual").Endpoint("read")
	rpc := root.API.JSONRPC.Service("dual").Endpoint("read")
	require.False(t, http.IsJSONRPC())
	require.True(t, rpc.IsJSONRPC())
	require.Len(t, root.API.HTTP.ServicesWithEndpoints(), 1)
	require.Len(t, root.API.JSONRPC.ServicesWithEndpoints(), 1)
	require.Nil(t, http.HTTPErrors[0].Response.JSONRPCError)
	require.NotNil(t, rpc.HTTPErrors[0].Response.JSONRPCError)
	require.Equal(t, 4403, rpc.HTTPErrors[0].Response.JSONRPCError.Code)
}

func TestAuthErrorResponsesRejectsJSONRPCScope(t *testing.T) {
	for _, scope := range []string{"API", "service", "method"} {
		t.Run(scope, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, func() {
				API("rpc", func() {
					if scope == "API" {
						JSONRPC(AuthErrorResponses)
					}
				})
				Service("svc", func() {
					JSONRPC(func() {
						POST("/rpc")
						if scope == "service" {
							AuthErrorResponses()
						}
					})
					Method("read", func() {
						JSONRPC(func() {
							if scope == "method" {
								AuthErrorResponses()
							}
						})
					})
				})
			})
			require.ErrorContains(t, err, "AuthErrorResponses is HTTP only")
		})
	}
}

func TestHTTPUnsetStatusSingleDiagnostic(t *testing.T) {
	for _, body := range []bool{false, true} {
		err := expr.RunInvalidDSL(t, func() {
			Service("svc", func() {
				Method("read", func() {
					if body {
						Result(String)
					}
					HTTP(func() {
						GET("/read")
						Response(StatusOK, func() {
							Code(0)
						})
					})
				})
			})
		})
		require.ErrorContains(t, err, "HTTP response requires an exact status code")
		require.NotContains(t, err.Error(), "HTTP response status not defined")
		require.NotContains(t, err.Error(), "Response body defined for status code 0")
	}
}

func TestDualTransportStreamingAdmission(t *testing.T) {
	for _, sse := range []bool{false, true} {
		for _, rpcFirst := range []bool{false, true} {
			name := "WebSocket"
			if sse {
				name = "SSE"
			}
			if rpcFirst {
				name += "/RPC first"
			} else {
				name += "/HTTP first"
			}
			t.Run(name, func(t *testing.T) {
				err := expr.RunInvalidDSL(t, dualTransportStreamingDSL(sse, rpcFirst, "dual"))
				require.ErrorContains(t, err, "streaming method cannot define both HTTP and JSON-RPC mappings")
			})
		}
		for _, mode := range []string{"HTTP", "RPC", "distinct methods"} {
			t.Run(mode+map[bool]string{false: "/WebSocket", true: "/SSE"}[sse], func(t *testing.T) {
				expr.RunDSL(t, dualTransportStreamingDSL(sse, false, mode))
			})
		}
	}
}

func dualTransportStreamingDSL(sse, rpcFirst bool, mode string) func() {
	return func() {
		Service("streams", func() {
			if mode != "HTTP" {
				JSONRPC(func() {
					if sse {
						POST("/rpc")
					} else {
						GET("/rpc")
					}
				})
			}
			http := func() {
				HTTP(func() {
					GET("/watch")
					if sse {
						ServerSentEvents(func() {
						})
					}
				})
			}
			rpc := func() {
				JSONRPC(func() {
					if sse {
						ServerSentEvents(func() {
						})
					}
				})
			}
			Method("watch", func() {
				StreamingResult(String)
				switch mode {
				case "HTTP", "distinct methods":
					http()
				case "RPC":
					rpc()
				default:
					if rpcFirst {
						rpc()
						http()
					} else {
						http()
						rpc()
					}
				}
			})
			if mode == "distinct methods" {
				Method("events", func() {
					StreamingResult(String)
					rpc()
				})
			}
		})
	}
}

func TestDualTransportRawBodyAdmission(t *testing.T) {
	for _, flag := range []string{"SkipRequestBodyEncodeDecode", "SkipResponseBodyEncodeDecode", "FileResponse"} {
		for _, rpcFirst := range []bool{false, true} {
			order := "HTTP first"
			if rpcFirst {
				order = "RPC first"
			}
			t.Run(flag+"/"+order, func(t *testing.T) {
				err := expr.RunInvalidDSL(t, dualTransportRawBodyDSL(flag, rpcFirst, "dual"))
				require.ErrorContains(t, err, "cannot use "+flag+" and define a JSON-RPC transport")
			})
		}
		for _, mode := range []string{"HTTP", "distinct methods"} {
			t.Run(flag+"/"+mode, func(t *testing.T) {
				expr.RunDSL(t, dualTransportRawBodyDSL(flag, false, mode))
			})
		}
	}
}

func dualTransportRawBodyDSL(flag string, rpcFirst bool, mode string) func() {
	return func() {
		Service("raw", func() {
			if mode != "HTTP" {
				JSONRPC(func() {
					POST("/rpc")
				})
			}
			http := func() {
				HTTP(func() {
					GET("/raw")
					Response(StatusOK)
					switch flag {
					case "SkipRequestBodyEncodeDecode":
						SkipRequestBodyEncodeDecode()
					case "SkipResponseBodyEncodeDecode":
						SkipResponseBodyEncodeDecode()
					case "FileResponse":
						FileResponse()
					}
				})
			}
			rpc := func() {
				JSONRPC(func() {
				})
			}
			Method("call", func() {
				switch {
				case mode != "dual":
					http()
				case rpcFirst:
					rpc()
					http()
				default:
					http()
					rpc()
				}
			})
			if mode == "distinct methods" {
				Method("rpc", rpc)
			}
		})
	}
}
