package codegen

import (
	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// RequestDecodePlan contains the derived control flow decisions for
// rendering a server request decoder.
type RequestDecodePlan struct {
	// HasElements is true when the request binds at least one path, query,
	// header, or cookie element.
	HasElements bool
	// HasPathParams is true when the request binds path parameters.
	HasPathParams bool
	// HasDecodedPathParams is true when at least one path parameter reads the
	// decoded Vars map rather than the escaped RawVars map used by arrays.
	HasDecodedPathParams bool
	// HasQueryParams is true when the request binds query parameters.
	HasQueryParams bool
	// HasHeaders is true when the request binds headers.
	HasHeaders bool
	// HasCookies is true when the request binds cookies.
	HasCookies bool
	// PathValuesVar is the local variable containing decoded URL captures.
	PathValuesVar string
	// JSONRPCParamsVar is the local variable containing JSON-RPC params bytes.
	// It is distinct from URL captures and all authored request locals.
	JSONRPCParamsVar string
	// QueryValuesVar is the local variable containing parsed query values.
	QueryValuesVar string
	// QueryErrorVar is the local variable containing a query parse error.
	QueryErrorVar string
	// MustValidate is true when decoded request elements or Basic credentials
	// may accumulate validation errors in the outer request decoder.
	MustValidate bool
	// NeedsErrorVar declares the decoder's shared validation/decode accumulator.
	NeedsErrorVar bool
}

func newRequestDecodePlan(request *RequestData, basic *BasicAuthData) *RequestDecodePlan {
	hasPathParams := len(request.PathParams) > 0
	hasDecodedPathParams := false
	for _, param := range request.PathParams {
		if !expr.IsArray(param.Type) {
			hasDecodedPathParams = true
			break
		}
	}
	hasQueryParams := len(request.QueryParams) > 0
	hasHeaders := len(request.Headers) > 0
	hasCookies := len(request.Cookies) > 0
	plan := &RequestDecodePlan{
		HasElements:          hasPathParams || hasQueryParams || hasHeaders || hasCookies,
		HasPathParams:        hasPathParams,
		HasDecodedPathParams: hasDecodedPathParams,
		HasQueryParams:       hasQueryParams,
		HasHeaders:           hasHeaders,
		HasCookies:           hasCookies,
		MustValidate:         request.MustValidate || requestHasPathArray(request),
	}
	customMultipart := request.Multipart && !request.MultipartGenerated
	if customMultipart {
		// The custom multipart adapter already validates its ordinary bound elements.
		plan.MustValidate = false
	}
	if basic != nil {
		plan.MustValidate = plan.MustValidate || basic.HeaderRequired || basic.UsernameValidate != "" || basic.PasswordValidate != ""
	}
	bodyError := request.ServerBody != nil && !customMultipart &&
		(!request.MultipartGenerated || request.ServerBody.ValidateRef != "")
	plan.NeedsErrorVar = plan.MustValidate || bodyError
	scope := codegen.NewNameScope()
	for _, param := range request.PathParams {
		scope.Unique(param.VarName)
	}
	for _, param := range request.QueryParams {
		scope.Unique(param.VarName)
	}
	for _, header := range request.Headers {
		scope.Unique(header.VarName)
	}
	for _, cookie := range request.Cookies {
		scope.Unique(cookie.VarName)
	}
	plan.PathValuesVar = scope.Unique("params")
	plan.JSONRPCParamsVar = scope.Unique("rpcParams")
	if hasQueryParams {
		plan.QueryValuesVar = scope.Unique("qp")
		plan.QueryErrorVar = scope.Unique("queryErr")
	}
	return plan
}

func requestHasPathArray(request *RequestData) bool {
	for _, param := range request.PathParams {
		if expr.IsArray(param.Type) {
			return true
		}
	}
	return false
}
