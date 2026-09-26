package expr

import "github.com/CaliLuke/loom/eval"

// validateAPIKeyLocations rejects the API keys of the security schemes of the
// endpoint that are read from a path parameter: an OpenAPI API key security
// scheme can only name a header, a query string parameter or a cookie.
func (e *HTTPEndpointExpr) validateAPIKeyLocations(verr *eval.ValidationErrors) {
	pathParams := e.PathParams()
	for _, requirement := range e.MethodExpr.EffectiveRequirements() {
		for _, scheme := range requirement.Schemes {
			if scheme.Kind != APIKeyKind {
				continue
			}
			field := TaggedAttribute(e.MethodExpr.Payload, "security:apikey:"+scheme.SchemeName)
			if field == "" {
				continue
			}
			if elem, ok := pathParams.FindKey(field); ok {
				verr.Add(e, "API key of security scheme %q is read from the path parameter %q, but an OpenAPI API key can only be sent in a header, a query string parameter or a cookie; map the %q attribute with Header, Cookie, or a Param that no route wildcard names", scheme.SchemeName, elem, field)
			}
		}
	}
}
