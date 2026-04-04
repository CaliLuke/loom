package transportir_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	testcodegen "github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	http_testdata "github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestBuildEndpointPlainHTTP(t *testing.T) {
	root := testcodegen.RunDSL(t, func() {
		jwt := dsl.JWTSecurity("jwt")
		dsl.Service("widgets", func() {
			dsl.HTTP(func() {
				dsl.Path("/widgets")
			})
			dsl.Method("show", func() {
				dsl.Security(jwt)
				dsl.Payload(func() {
					dsl.Attribute("id", dsl.String)
					dsl.Attribute("filter", dsl.String)
					dsl.Attribute("trace", dsl.String)
					dsl.Attribute("session", dsl.String)
					dsl.Token("token", dsl.String)
					dsl.Attribute("body", dsl.String)
					dsl.Required("id", "token")
				})
				dsl.Result(dsl.String)
				dsl.HTTP(func() {
					dsl.POST("/{id}")
					dsl.Param("filter")
					dsl.Header("trace")
					dsl.Cookie("session")
					dsl.Body("body")
				})
			})
		})
	})

	endpoint := transportir.BuildEndpoint(root.API.HTTP.Services[0].HTTPEndpoints[0])
	require.Equal(t, "show", endpoint.Name)
	require.False(t, endpoint.IsJSONRPC)
	require.Len(t, endpoint.Routes, 1)
	require.Equal(t, "POST", endpoint.Routes[0].Method)
	require.Equal(t, "/widgets/{id}", endpoint.Routes[0].Path)
	require.Len(t, endpoint.Request.PathParams, 1)
	require.Equal(t, "id", endpoint.Request.PathParams[0].Name)
	require.Len(t, endpoint.Request.QueryParams, 1)
	require.Equal(t, "filter", endpoint.Request.QueryParams[0].Name)
	require.Len(t, endpoint.Request.Headers, 2)
	require.Len(t, endpoint.Request.Cookies, 1)
	require.NotNil(t, endpoint.Request.Body)
	require.Empty(t, endpoint.Request.IDAttribute)
	require.Len(t, endpoint.Security.Requirements, 1)
	require.NotEmpty(t, endpoint.Security.Parameters)
	require.Equal(t, "Authorization", endpoint.Security.Parameters[0].Name)
	require.Equal(t, "header", endpoint.Security.Parameters[0].In)
}

func TestBuildEndpointJSONRPCPost(t *testing.T) {
	endpoint := firstJSONRPCEndpoint(t, func() {
		dsl.API("jsonrpc-post", func() {
			dsl.JSONRPC(func() {})
		})
		dsl.Service("rpc", func() {
			dsl.JSONRPC(func() {
				dsl.POST("/rpc")
			})
			dsl.Method("ping", func() {
				dsl.Payload(func() {
					dsl.ID("id", dsl.String)
				})
				dsl.Result(func() {
					dsl.ID("id", dsl.String)
					dsl.Attribute("value", dsl.String)
				})
				dsl.JSONRPC(func() {})
			})
		})
	})

	require.True(t, endpoint.IsJSONRPC)
	require.Len(t, endpoint.Routes, 1)
	require.Equal(t, "POST", endpoint.Routes[0].Method)
	require.Equal(t, "id", endpoint.Request.IDAttribute)
	require.False(t, endpoint.Request.IDAttributeRequired)
	require.Equal(t, "id", endpoint.Response.IDAttribute)
	require.False(t, endpoint.Response.IDAttributeRequired)
	require.False(t, endpoint.Stream.IsStreaming)
}

func TestBuildEndpointJSONRPCSSE(t *testing.T) {
	endpoint := firstJSONRPCEndpoint(t, func() {
		dsl.API("jsonrpc-sse", func() {
			dsl.JSONRPC(func() {})
		})
		dsl.Service("rpc", func() {
			dsl.JSONRPC(func() {
				dsl.POST("/rpc")
			})
			dsl.Method("events/stream", func() {
				dsl.Payload(func() {
					dsl.ID("id", dsl.String)
				})
				dsl.StreamingResult(func() {
					dsl.Attribute("value", dsl.String)
				})
				dsl.JSONRPC(func() {
					dsl.ServerSentEvents()
				})
			})
		})
	})

	require.True(t, endpoint.Stream.IsStreaming)
	require.True(t, endpoint.Stream.IsSSE)
	require.Equal(t, "sse", endpoint.Stream.Transport)
	require.Equal(t, "POST", endpoint.Stream.HandshakeMethod)
	require.Equal(t, expr.StatusOK, endpoint.Stream.HandshakeStatus)
}

func TestBuildEndpointJSONRPCWebSocket(t *testing.T) {
	endpoint := firstJSONRPCEndpoint(t, func() {
		dsl.API("jsonrpc-ws", func() {
			dsl.JSONRPC(func() {})
		})
		dsl.Service("rpc", func() {
			dsl.JSONRPC(func() {
				dsl.GET("/ws")
			})
			dsl.Method("stream", func() {
				dsl.Payload(func() {
					dsl.ID("id", dsl.String)
					dsl.Attribute("value", dsl.String)
				})
				dsl.StreamingResult(func() {
					dsl.Attribute("event", dsl.String)
				})
				dsl.JSONRPC(func() {})
			})
		})
	})

	require.True(t, endpoint.Stream.IsStreaming)
	require.True(t, endpoint.Stream.IsWebSocket)
	require.Equal(t, "websocket", endpoint.Stream.Transport)
	require.Equal(t, "GET", endpoint.Stream.HandshakeMethod)
	require.Equal(t, expr.StatusSwitchingProtocols, endpoint.Stream.HandshakeStatus)
}

func TestBuildEndpointMixedResults(t *testing.T) {
	root := testcodegen.RunDSL(t, http_testdata.MixedResultsDSL)
	endpoint := transportir.BuildEndpoint(root.API.HTTP.Services[0].HTTPEndpoints[0])

	require.True(t, endpoint.Response.HasMixedResults)
	require.True(t, endpoint.Stream.IsSSE)
	require.Equal(t, "sse", endpoint.Stream.Transport)
	require.NotNil(t, endpoint.Request.Body)
	require.NotNil(t, endpoint.Response.Result)
	require.Len(t, endpoint.Response.Responses, 1)
}

func firstJSONRPCEndpoint(t *testing.T, dslFn func()) *transportir.Endpoint {
	t.Helper()
	root := testcodegen.RunDSL(t, dslFn)
	require.NotEmpty(t, root.API.JSONRPC.Services)
	require.NotEmpty(t, root.API.JSONRPC.Services[0].HTTPEndpoints)
	return transportir.BuildEndpoint(root.API.JSONRPC.Services[0].HTTPEndpoints[0])
}
