package expr

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/eval"
)

func TestHTTPResponseProtocolArms(t *testing.T) {
	for _, rpc := range []bool{false, true} {
		for _, test := range []struct {
			name      string
			status    HTTPStatusSelector
			mapping   *JSONRPCErrorMapping
			httpValid bool
			rpcValid  bool
		}{
			{"HTTP", HTTPStatusSelector{Code: 403}, nil, true, false},
			{"RPC zero", HTTPStatusSelector{}, &JSONRPCErrorMapping{Code: 0}, false, true},
			{"RPC negative", HTTPStatusSelector{}, &JSONRPCErrorMapping{Code: -32603}, false, true},
			{"both", HTTPStatusSelector{Code: 403}, &JSONRPCErrorMapping{Code: 4403}, false, false},
			{"neither", HTTPStatusSelector{}, nil, false, false},
			{"invalid HTTP", HTTPStatusSelector{Code: 4403}, nil, false, false},
		} {
			name := "HTTP/"
			want := test.httpValid
			if rpc {
				name = "RPC/"
				want = test.rpcValid
			}
			t.Run(name+test.name, func(t *testing.T) {
				response := &HTTPResponseExpr{Status: test.status, JSONRPCError: test.mapping}
				errors := new(eval.ValidationErrors)
				response.validateProtocol(rpc, errors)
				require.Equal(t, want, len(errors.Errors) == 0)
			})
		}
	}
}

func TestHTTPErrorProtocolCheckedBeforeIdentity(t *testing.T) {
	SetupTestDSL(t)
	endpoint := &HTTPEndpointExpr{Meta: MetaExpr{"jsonrpc": {}}, MethodExpr: &MethodExpr{Name: "read", Service: &ServiceExpr{Name: "svc"}}}
	response := &HTTPResponseExpr{Parent: endpoint, Status: HTTPStatusSelector{Code: 400}}
	failure := &HTTPErrorExpr{Name: "missing", Response: response}
	errors := failure.Validate()
	require.ErrorContains(t, errors, "JSON-RPC error response requires")
	require.ErrorContains(t, errors, "does not match an error")
}
