package expr

import (
	"strings"

	"github.com/CaliLuke/loom/eval"
)

type (
	// HTTPErrorExpr defines a HTTP error response including its name,
	// status, headers and result type.
	HTTPErrorExpr struct {
		// ErrorExpr is the underlying Loom design error expression.
		*ErrorExpr
		// Name of error, we need a separate copy of the name to match it
		// up with the appropriate ErrorExpr.
		Name string
		// Response is the corresponding HTTP response.
		Response *HTTPResponseExpr
	}
)

// EvalName returns the generic definition name used in error messages.
func (e *HTTPErrorExpr) EvalName() string {
	return "HTTP error " + e.Name
}

// Validate makes sure there is a error expression that matches the HTTP error
// expression.
func (e *HTTPErrorExpr) Validate() *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	var ee *ErrorExpr
	switch p := e.Response.Parent.(type) {
	case *HTTPEndpointExpr:
		ee = p.MethodExpr.Error(e.Name)
		if ee == nil {
			verr.Add(e.Response, "Error %#v does not match an error defined in the method", e.Name)
			return verr
		}
	case *HTTPServiceExpr:
		ee = p.Error(e.Name)
		if ee == nil {
			verr.Add(e.Response, "Error %#v does not match an error defined in the service", e.Name)
			return verr
		}
	case *RootExpr:
		ee = Root.Error(e.Name)
		if ee == nil {
			verr.Add(e.Response, "Error %#v does not match an error defined in the API", e.Name)
			return verr
		}
	default:
		return verr
	}

	e.Response.prepareErrorWireBindings(ee.AttributeExpr)
	e.Response.validateWireBindings(verr)
	if len(e.Response.Cookies) > 0 {
		e.Response.validateCookieSecurity(verr)
		e.Response.validateCookies(
			ee.AttributeExpr,
			func(name string) DataType {
				_, att := ee.FindAttribute(name)
				if att == nil {
					return nil
				}
				return att.Type
			},
			"error type",
			"",
			"Array error",
			e.Response,
			verr,
		)
	}
	e.validateHeaders(ee, verr)
	e.validateStatusErrorName(ee.AttributeExpr, verr)
	body, _ := responseBodySource(ee.AttributeExpr, e.Response)
	if IsDefaultErrorResult(ee.Type) && IsNullable(ee.AttributeExpr) && IsNullable(body) && strings.EqualFold(strings.TrimSpace(strings.Split(e.Response.ContentType, ";")[0]), "application/problem+json") {
		verr.Add(e.Response, "a nullable whole problem body requires application/json, not application/problem+json")
	}
	validateBodyParameterMetadata(body, "body", e.Response, verr, make(map[*AttributeExpr]bool))
	// Built-in problem documents reconstruct ServiceError through the problem
	// adapter, which supplies zero values for omitted properties. Their required
	// wire fields are not required service fields. Custom errors reconstruct
	// their authored object and must retain its required-field coverage.
	if !IsDefaultErrorResult(ee.Type) {
		validateReconstruction(ee.AttributeExpr, body, responseReconstructionBindings(e.Response), "error "+e.Name, e.Response, verr)
	}
	return verr
}

func (e *HTTPErrorExpr) validateHeaders(ee *ErrorExpr, verr *eval.ValidationErrors) {
	validateHeaderCollections(e.Response.Headers, ee.AttributeExpr, e.Response, verr)
	validateResponseHeaderCodecs(e.Response.Headers, ee.AttributeExpr, e.Response, verr)
	if e.Response.Headers != nil && !e.Response.Headers.IsEmpty() {
		verr.Merge(e.Response.Headers.Validate("HTTP error response headers", e.Response))
		switch {
		case ee.Type == Empty:
			verr.Add(e.Response, "response defines headers but error type is empty")
		case IsObject(ee.Type):
			for _, h := range *AsObject(e.Response.Headers.Type) {
				_, att := ee.FindAttribute(h.Name)
				if att == nil {
					verr.Add(e.Response, "header %q has no equivalent attribute in error type, use notation 'attribute_name:header_name' to identify corresponding error type attribute.", h.Name)
				}
			}
		case len(*AsObject(e.Response.Headers.Type)) > 1:
			verr.Add(e.Response, "response defines more than one headers but error type is not an object")
		}
	}
}

// Finalize looks up the corresponding method error expression.
func (e *HTTPErrorExpr) Finalize(a *HTTPEndpointExpr) {
	e.ErrorExpr = e.designError()
	e.Response.Finalize(a, e.AttributeExpr)
	if e.Response.Body == nil {
		e.Response.Body = httpErrorResponseBody(a, e)
		e.Response.Body.Finalize()
	}

	// Initialize response content type if result is media type.
	if e.Response.Body.Type == Empty {
		return
	}
	if e.Response.ContentType != "" {
		return
	}
	mt, ok := e.Response.Body.Type.(*ResultTypeExpr)
	if !ok {
		return
	}
	if IsDefaultErrorResult(e.Type) && IsNullable(e.AttributeExpr) && IsNullable(e.Response.Body) {
		e.Response.ContentType = "application/json"
		return
	}
	e.Response.ContentType = mt.Identifier
}

// Dup creates a copy of the error expression.
func (e *HTTPErrorExpr) Dup() *HTTPErrorExpr {
	return &HTTPErrorExpr{
		ErrorExpr: e.ErrorExpr,
		Name:      e.Name,
		Response:  e.Response.Dup(),
	}
}

// designError returns the error expression that the HTTP error maps, looked
// up where the HTTP error is declared, or nil if there is none.
func (e *HTTPErrorExpr) designError() *ErrorExpr {
	switch p := e.Response.Parent.(type) {
	case *HTTPEndpointExpr:
		return p.MethodExpr.Error(e.Name)
	case *HTTPServiceExpr:
		return p.Error(e.Name)
	case *RootExpr:
		return Root.Error(e.Name)
	}
	return nil
}
