package expr

import (
	"testing"

	"github.com/CaliLuke/loom/eval"
	"github.com/stretchr/testify/require"
)

func TestHTTPEndpointValidateBodyAndPayloadWithoutPayload(t *testing.T) {
	mapQueryParams := "filters"
	endpoint := &HTTPEndpointExpr{
		MethodExpr: &MethodExpr{
			Name:    "Method",
			Payload: &AttributeExpr{Type: Empty},
		},
		Service:             &HTTPServiceExpr{ServiceExpr: &ServiceExpr{Name: "Service"}},
		Params:              NewEmptyMappedAttributeExpr(),
		Headers:             NewEmptyMappedAttributeExpr(),
		Cookies:             NewEmptyMappedAttributeExpr(),
		MapQueryParams:      &mapQueryParams,
		MultipartRequest:    true,
		FormRequest:         true,
		OptionalRequestBody: true,
	}

	var verr eval.ValidationErrors
	endpoint.validateBodyAndPayload(&verr)

	require.Equal(t, []string{
		"MapParams is set but Payload is not defined",
		"MultipartRequest is set but Payload is not defined",
		"FormRequest is set but Payload is not defined",
		"OptionalRequestBody is set but Payload is not defined",
	}, validationErrorMessages(verr.Errors))
}

func TestHTTPEndpointPrepareNormalizesRouteMethods(t *testing.T) {
	previousRoot := Root
	api := NewAPIExpr("route-method", func() {})
	Root = &RootExpr{API: api}
	t.Cleanup(func() {
		Root = previousRoot
	})

	serviceExpr := &ServiceExpr{Name: "Service"}
	service := &HTTPServiceExpr{Root: api.HTTP, ServiceExpr: serviceExpr}
	method := &MethodExpr{
		Name:    "Call",
		Payload: &AttributeExpr{Type: Empty},
		Result:  &AttributeExpr{Type: Empty},
		Service: serviceExpr,
	}
	endpoint := &HTTPEndpointExpr{MethodExpr: method, Service: service}
	route := &RouteExpr{Method: "purge", Path: "/cache", Endpoint: endpoint}
	endpoint.Routes = []*RouteExpr{route}

	endpoint.Prepare()

	require.Equal(t, "PURGE", route.Method)
}

func validationErrorMessages(errs []error) []string {
	msgs := make([]string, 0, len(errs))
	for _, err := range errs {
		msgs = append(msgs, err.Error())
	}
	return msgs
}

func TestHTTPEndpointValidateParams(t *testing.T) {
	queryElem := &AttributeExpr{Type: &Object{}}
	queryArray := &AttributeExpr{Type: &Array{ElemType: queryElem}}
	service := &HTTPServiceExpr{
		ServiceExpr: &ServiceExpr{Name: "Service"},
		Root:        &HTTPExpr{Path: ""},
	}

	paramsObject := &Object{}
	paramsObject.Set("id", &AttributeExpr{Type: String})
	paramsObject.Set("filter", queryArray)

	payloadObject := &Object{}
	payloadObject.Set("filter", queryArray)

	endpoint := &HTTPEndpointExpr{
		MethodExpr: &MethodExpr{
			Name:    "Method",
			Payload: &AttributeExpr{Type: payloadObject},
		},
		Service: service,
		Routes: []*RouteExpr{{
			Method:   "GET",
			Path:     "/{id}",
			Endpoint: nil,
		}},
		Params: NewMappedAttributeExpr(&AttributeExpr{Type: paramsObject}),
	}
	endpoint.Routes[0].Endpoint = endpoint

	verr := endpoint.validateParams()
	require.NotNil(t, verr)
	require.Equal(t, []string{
		`elements of array query parameter "filter" must be primitive`,
		`Path parameter "id" not found in payload.`,
	}, validationErrorMessages(verr.Errors))
}

func TestHTTPEndpointValidateHeadersAndCookies(t *testing.T) {
	payloadObject := &Object{}
	payloadObject.Set("user", &AttributeExpr{
		Type: String,
		Meta: MetaExpr{"security:username": []string{""}},
	})
	payloadObject.Set("token", &AttributeExpr{Type: String})
	payloadObject.Set("session", &AttributeExpr{
		Type: &Array{ElemType: &AttributeExpr{Type: String}},
	})

	headersObject := &Object{}
	headersObject.Set("token", &AttributeExpr{Type: String})

	cookiesObject := &Object{}
	cookiesObject.Set("session", &AttributeExpr{
		Type: &Array{ElemType: &AttributeExpr{Type: String}},
	})

	headers := NewMappedAttributeExpr(&AttributeExpr{Type: headersObject})
	headers.Map("Authorization", "token")

	endpoint := &HTTPEndpointExpr{
		MethodExpr: &MethodExpr{
			Name:    "Method",
			Payload: &AttributeExpr{Type: payloadObject},
		},
		Service: &HTTPServiceExpr{ServiceExpr: &ServiceExpr{Name: "Service"}},
		Headers: headers,
		Cookies: NewMappedAttributeExpr(&AttributeExpr{Type: cookiesObject}),
	}

	verr := endpoint.validateHeadersAndCookies()
	require.NotNil(t, verr)
	require.Equal(t, []string{
		`cookie "session" must be primitive`,
		`Attribute "token" is mapped to "Authorization" header in the endpoint secured by BasicAuth which also sets "Authorization" header. Specify a different header to map attribute "token".`,
	}, validationErrorMessages(verr.Errors))
}
