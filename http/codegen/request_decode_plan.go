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
	// QueryValuesVar is the local variable containing parsed query values.
	QueryValuesVar string
	// QueryErrorVar is the local variable containing a query parse error.
	QueryErrorVar string
	// MustValidate is true when decoded request elements may accumulate
	// validation errors.
	MustValidate bool
}

// requestNeedsServerErrorVar reports whether the generated server request
// decoder declares the shared "err" accumulator variable. Multipart-generated
// requests need it when field or body validation runs or an array path may
// report malformed escaping. A required multipart file field always produces
// a non-empty ServerBody.ValidateRef (the generated Validate<Body> function
// treats the resulting nil field as missing).
func requestNeedsServerErrorVar(request *RequestData) bool {
	return !request.MultipartGenerated || request.MustValidate ||
		(request.ServerBody != nil && request.ServerBody.ValidateRef != "") ||
		requestHasPathArray(request)
}

func newRequestDecodePlan(request *RequestData) *RequestDecodePlan {
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
	if !hasQueryParams {
		return plan
	}

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
	plan.QueryValuesVar = scope.Unique("qp")
	plan.QueryErrorVar = scope.Unique("queryErr")
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
