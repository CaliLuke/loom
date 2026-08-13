package dsl_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestRepeatedTransportBlocksCompose(t *testing.T) {
	t.Run("HTTP service", func(t *testing.T) {
		root := expr.RunDSL(t, func() {
			Service("svc", func() {
				HTTP(func() {
					Path("/base")
				})
				Error("boom", ErrorResult, "boom")
				HTTP(func() {
					Response("boom", StatusConflict)
				})
				Method("get", func() {
					Result(String)
					HTTP(func() {
						GET("/thing")
					})
				})
			})
		})

		svc := root.API.HTTP.Service("svc")
		require.Equal(t, []string{"/base"}, svc.Paths)
		require.Len(t, svc.HTTPErrors, 1)
		require.Equal(t, []string{"/base/thing"}, svc.Endpoint("get").Routes[0].FullPaths())
	})

	t.Run("HTTP method", func(t *testing.T) {
		root := expr.RunDSL(t, func() {
			Service("svc", func() {
				Method("get", func() {
					Result(String)
					HTTP(func() {
						GET("/thing")
					})
					HTTP(func() {
						Response(StatusCreated)
					})
				})
			})
		})

		endpoint := root.API.HTTP.Service("svc").Endpoint("get")
		require.Len(t, endpoint.Routes, 1)
		require.Equal(t, expr.StatusCreated, endpoint.Responses[0].StatusCode)
	})

	t.Run("JSON-RPC service", func(t *testing.T) {
		root := expr.RunDSL(t, func() {
			Service("svc", func() {
				JSONRPC(func() {
					POST("/rpc")
				})
				Error("boom", ErrorResult, "boom")
				JSONRPC(func() {
					Response("boom", RPCInternalError)
				})
				Method("call", func() {
					Payload(func() {
						ID("id", String)
					})
					Result(func() {
						ID("id", String)
					})
					JSONRPC(func() {})
				})
			})
		})

		svc := root.API.JSONRPC.Service("svc")
		require.Equal(t, "/rpc", svc.JSONRPCRoute.Path)
		require.Len(t, svc.HTTPErrors, 1)
	})

	t.Run("JSON-RPC method", func(t *testing.T) {
		root := expr.RunDSL(t, func() {
			Service("svc", func() {
				Error("boom", ErrorResult, "boom")
				JSONRPC(func() {
					POST("/rpc")
				})
				Method("call", func() {
					Payload(func() {
						ID("id", String)
					})
					Result(func() {
						ID("id", String)
					})
					Error("boom")
					JSONRPC(func() {
						Response("boom", RPCInternalError)
					})
					JSONRPC(func() {})
				})
			})
		})

		endpoint := root.API.JSONRPC.Service("svc").Endpoint("call")
		require.Len(t, endpoint.HTTPErrors, 1)
		require.Equal(t, expr.RPCInternalError, endpoint.HTTPErrors[0].Response.StatusCode)
	})

	t.Run("gRPC service", func(t *testing.T) {
		root := expr.RunDSL(t, func() {
			Service("svc", func() {
				GRPC(func() {
					Package("custom")
				})
				Error("boom", ErrorResult, "boom")
				GRPC(func() {
					Response("boom", CodeUnavailable)
				})
				Method("call", func() {
					Result(String)
					GRPC(func() {})
				})
			})
		})

		svc := root.API.GRPC.Service("svc")
		require.Equal(t, "custom", svc.ProtoPkg)
		require.Len(t, svc.GRPCErrors, 1)
	})

	t.Run("gRPC method", func(t *testing.T) {
		root := expr.RunDSL(t, func() {
			Service("svc", func() {
				Method("call", func() {
					Payload(func() {
						Field(1, "message", String)
					})
					Result(String)
					GRPC(func() {
						Message(func() {
							Attribute("message")
						})
					})
					GRPC(func() {
						Response(CodeOK)
					})
				})
			})
		})

		endpoint := root.API.GRPC.Service("svc").Endpoint("call")
		require.NotNil(t, endpoint.Request)
		require.NotNil(t, expr.AsObject(endpoint.Request.Type).Attribute("message"))
		require.Equal(t, CodeOK, endpoint.Response.StatusCode)
	})
}

func TestRepeatedAPITransportBlocksCompose(t *testing.T) {
	root := expr.RunDSL(t, func() {
		API("api", func() {
			Error("first", ErrorResult, "first")
			Error("second", ErrorResult, "second")

			HTTP(func() {
				Response("first", StatusBadRequest)
			})
			HTTP(func() {
				Response("second", StatusConflict)
			})

			JSONRPC(func() {
				Response("first", RPCInvalidRequest)
			})
			JSONRPC(func() {
				Response("second", RPCInternalError)
			})

			GRPC(func() {
				Response("first", CodeInvalidArgument)
			})
			GRPC(func() {
				Response("second", CodeUnavailable)
			})
		})
	})

	require.Len(t, root.API.HTTP.Errors, 2)
	require.Len(t, root.API.JSONRPC.Errors, 2)
	require.Len(t, root.API.GRPC.Errors, 2)
}

func TestRepeatedTransportBlockErrorLocations(t *testing.T) {
	tests := []struct {
		name   string
		blocks func()
	}{
		{
			name: "first block",
			blocks: func() {
				HTTP(func() {
					Path("/invalid-at-method-scope")
				})
				HTTP(func() {
					GET("/thing")
				})
			},
		},
		{
			name: "second block",
			blocks: func() {
				HTTP(func() {
					GET("/thing")
				})
				HTTP(func() {
					Path("/invalid-at-method-scope")
				})
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, func() {
				Service("svc", func() {
					Method("get", func() {
						Result(String)
						test.blocks()
					})
				})
			})

			require.Contains(t, err.Error(), "transport_composition_test.go:")
			require.False(t, strings.Contains(err.Error(), "compose.go:"), err.Error())
		})
	}
}
