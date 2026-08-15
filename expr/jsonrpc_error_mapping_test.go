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
				require.Equal(t, 4403, endpoint.HTTPErrors[0].Response.StatusCode)
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
