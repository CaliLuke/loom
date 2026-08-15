package codegen

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

func TestJSONRPCServerUsesDesignedTypedErrorBody(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonRPCTypedErrorBodyDSL)
	services := CreateJSONRPCServices(root)
	endpoint := services.Get("tools").Endpoint("call")
	require.Len(t, endpoint.Errors, 1)
	require.Len(t, endpoint.Errors[0].Errors, 2)

	code := jsonrpcGeneratedCode(t, ServerFiles("", services))
	for _, item := range endpoint.Errors[0].Errors {
		require.NotNil(t, item.Response)
		require.Len(t, item.Response.ServerBody, 1)
		require.NotNil(t, item.Response.ServerBody[0].Init)
		mappedCase := jsonRPCErrorCase(t, code, item.Name)
		require.Contains(t, mappedCase, item.Response.ServerBody[0].Init.Name+"(")
		require.Contains(t, mappedCase, "errors.As(err, &res)")
		require.Contains(t, mappedCase, "loom.ErrorSafeMessage(err), data")
	}
	require.Contains(t, code, "jsonrpc.NewErrorData(err)")
}

func TestJSONRPCStreamingServersUseDesignedTypedErrorBody(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonRPCTypedStreamingErrorBodyDSL)
	services := CreateJSONRPCServices(root)
	endpoint := services.Get("tools").Endpoint("watch")
	require.Len(t, endpoint.Errors, 1)
	require.Len(t, endpoint.Errors[0].Errors, 1)
	item := endpoint.Errors[0].Errors[0]
	require.NotNil(t, item.Response)
	require.NotNil(t, item.Response.EncodePlan.FirstBody.Init)
	constructor := item.Response.EncodePlan.FirstBody.Init.Name + "(res)"

	code := jsonrpcGeneratedCode(t, ServerFiles("", services)) +
		jsonrpcGeneratedCode(t, SSEServerFiles("", services))
	require.Contains(t, code, constructor)
	require.Contains(t, code, "loom.ErrorSafeMessage(err), data")

	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/jsonrpctypedstreamerror", root)
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "test", "./...")
}

func jsonRPCErrorCase(t *testing.T, code, name string) string {
	t.Helper()
	start := strings.Index(code, `case "`+name+`":`)
	require.NotEqual(t, -1, start)
	rest := code[start:]
	lines := strings.Split(rest, "\n")
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, `case "`) || line == "default:" {
			return strings.Join(lines[:i], "\n")
		}
	}
	return rest
}

func jsonRPCTypedErrorBodyDSL() {
	toolError := dsl.Type("ToolError", func() {
		dsl.ErrorName("name", dsl.String)
		dsl.Attribute("code", dsl.String)
		dsl.Attribute("message", dsl.String)
		dsl.Attribute("retry_hint", dsl.String)
		dsl.Required("name", "code", "message", "retry_hint")
	})
	dsl.Service("tools", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
		})
		dsl.Method("call", func() {
			dsl.Result(dsl.String)
			dsl.Error("forbidden", toolError)
			dsl.Error("denied", toolError)
			dsl.JSONRPC(func() {
				dsl.Response("forbidden", 4403)
				dsl.Response("denied", 4403)
			})
		})
	})
}

func jsonRPCTypedStreamingErrorBodyDSL() {
	toolError := dsl.Type("ToolError", func() {
		dsl.ErrorName("name", dsl.String)
		dsl.Attribute("message", dsl.String)
		dsl.Required("name", "message")
	})
	dsl.Service("tools", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
		})
		dsl.Method("watch", func() {
			dsl.StreamingResult(func() {
				dsl.Attribute("value", dsl.String)
				dsl.Required("value")
			})
			dsl.Error("forbidden", toolError)
			dsl.JSONRPC(func() {
				dsl.ServerSentEvents()
				dsl.Response("forbidden", 4403)
			})
		})
	})
}
