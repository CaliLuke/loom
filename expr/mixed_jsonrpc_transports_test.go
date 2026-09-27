package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/expr/testdata"
)

func TestExportedJSONRPCTransportFixtures(t *testing.T) {
	for _, tc := range []struct {
		name      string
		design    func()
		method    string
		path      string
		endpoints int
	}{
		{"HTTP and SSE", testdata.MixedJSONRPCTransportsAPI, "POST", "/api/rpc", 3},
		{"WebSocket", testdata.ValidWebSocketOnlyAPI, "GET", "/ws", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := expr.RunDSL(t, tc.design)
			require.Len(t, root.API.JSONRPC.Services, 1)
			service := root.API.JSONRPC.Services[0]
			require.Len(t, service.HTTPEndpoints, tc.endpoints)
			for _, endpoint := range service.HTTPEndpoints {
				require.True(t, endpoint.IsJSONRPC())
				require.Len(t, endpoint.Routes, 1)
				require.Equal(t, tc.method, endpoint.Routes[0].Method)
				require.Equal(t, tc.path, endpoint.Routes[0].Path)
				if endpoint.Name() == "WatchUsers" {
					require.NotNil(t, endpoint.SSE)
				} else {
					require.Nil(t, endpoint.SSE)
				}
			}
		})
	}
	t.Run("invalid mixed WebSocket", func(t *testing.T) {
		err := expr.RunInvalidDSL(t, testdata.InvalidMixedWebSocketAPI)
		require.ErrorContains(t, err, `JSON-RPC service "InvalidService" cannot mix WebSocket with other transports`)
		require.NotContains(t, err.Error(), "cannot define routes at the method level")
	})
}
