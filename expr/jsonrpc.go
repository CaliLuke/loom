package expr

type (
	// JSONRPCExpr contains the API level JSON-RPC specific expressions.
	JSONRPCExpr struct {
		HTTPExpr
	}
)

// EvalName returns the name printed in case of evaluation error.
func (*JSONRPCExpr) EvalName() string {
	return "API JSON-RPC"
}

// ServicesWithEndpoints returns the JSON-RPC services that have method
// endpoints, preserving the configured service order.
func (j *JSONRPCExpr) ServicesWithEndpoints() []*HTTPServiceExpr {
	return httpServicesWithEndpoints(&j.HTTPExpr, func(endpoint *HTTPEndpointExpr) bool {
		return endpoint.IsJSONRPC()
	})
}

// Prepare copies shared HTTP API constructs over to the JSON-RPC API.
func (j *JSONRPCExpr) Prepare() {
	j.Path = Root.API.HTTP.Path
	j.Params = Root.API.HTTP.Params
	j.Headers = Root.API.HTTP.Headers
	j.Cookies = Root.API.HTTP.Cookies
	j.SSE = Root.API.HTTP.SSE
}
