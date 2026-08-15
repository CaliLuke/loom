package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

func TestJSONRPCClientDoesNotUseHTTPErrorMappings(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonRPCErrorMappingIsolationDSL)
	services := CreateJSONRPCServices(root)
	endpoint := services.Get("tools").Endpoint("call")
	require.Empty(t, endpoint.Errors)

	code := jsonrpcGeneratedCode(t, ClientFiles("", services))
	require.Contains(t, code, "return nil, jresp.Error")
	require.NotContains(t, code, "case 403:")
}

func jsonRPCErrorMappingIsolationDSL() {
	toolError := dsl.Type("ToolError", func() {
		dsl.ErrorName("name", dsl.String)
		dsl.Attribute("code", dsl.String)
		dsl.Attribute("message", dsl.String)
		dsl.Required("name", "code", "message")
	})
	dsl.API("example", func() {
		dsl.Error("forbidden", dsl.ErrorResult)
		dsl.HTTP(func() {
			dsl.Response("forbidden", dsl.StatusForbidden)
		})
	})
	dsl.Service("tools", func() {
		dsl.Error("forbidden", toolError)
		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
		})
		dsl.Method("call", func() {
			dsl.Result(dsl.String)
			dsl.JSONRPC(func() {})
		})
	})
}
