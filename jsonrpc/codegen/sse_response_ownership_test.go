package codegen

import (
	"testing"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/jsonrpc/codegen/testdata"
	"github.com/stretchr/testify/require"
)

func TestJSONRPCSSEHandshakeUsesResponseOwner(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.JSONRPCSSEObjectDSL)
	services := CreateJSONRPCServices(root)
	for _, service := range services.Services {
		for _, endpoint := range services.Get(service.Name).Endpoints {
			code := codegen.SectionCode(t, jsonrpcClientEndpointInitSection(endpoint))
			require.Contains(t, code, "loomhttp.DecodeResponse(resp, false, true,")
			require.Contains(t, code, "loomhttp.ReadUnexpectedResponseBody(resp)")
			require.Contains(t, code, "loomhttp.ErrDecodingError(")
			require.NotContains(t, code, "resp.Body.Close()")
			require.NotContains(t, code, "io.ReadAll(")
		}
	}
}
