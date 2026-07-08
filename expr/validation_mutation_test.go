package expr

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/eval"
)

func TestAttributeValidateDoesNotPropagatePkgPath(t *testing.T) {
	SetupTestDSL(t)

	childAttr := &AttributeExpr{Type: &Object{}}
	childType := &UserTypeExpr{TypeName: "Child", AttributeExpr: childAttr}
	parentObj := &Object{}
	parentObj.Set("child", &AttributeExpr{Type: childType})
	parentAttr := &AttributeExpr{
		Type: parentObj,
		Meta: MetaExpr{"struct:pkg:path": []string{"example/types"}},
	}
	parentType := &UserTypeExpr{TypeName: "Parent", AttributeExpr: parentAttr}
	rootAttr := &AttributeExpr{Type: parentType}

	validated = make(map[*AttributeExpr]bool)
	require.Empty(t, rootAttr.Validate("root", rootAttr).Errors)
	require.NotContains(t, childAttr.Meta, "struct:pkg:path")

	rootAttr.Prepare()
	require.Equal(t, []string{"example/types"}, childAttr.Meta["struct:pkg:path"])

	validated = make(map[*AttributeExpr]bool)
	require.Empty(t, rootAttr.Validate("root", rootAttr).Errors)
	validated = make(map[*AttributeExpr]bool)
	require.Empty(t, rootAttr.Validate("root", rootAttr).Errors)
	require.Equal(t, []string{"example/types"}, childAttr.Meta["struct:pkg:path"])
}

func TestMethodValidateDoesNotMergeInheritedInterceptors(t *testing.T) {
	root := SetupTestDSL(t)
	root.API.ClientInterceptors = []*InterceptorExpr{{Name: "api-client"}}
	root.API.ServerInterceptors = []*InterceptorExpr{{Name: "api-server"}}

	service := &ServiceExpr{
		Name:               "svc",
		ClientInterceptors: []*InterceptorExpr{{Name: "service-client"}},
		ServerInterceptors: []*InterceptorExpr{{Name: "service-server"}},
	}
	method := &MethodExpr{
		Name:               "show",
		Service:            service,
		Payload:            &AttributeExpr{Type: &Object{}},
		Result:             &AttributeExpr{Type: &Object{}},
		ClientInterceptors: []*InterceptorExpr{{Name: "method-client"}},
		ServerInterceptors: []*InterceptorExpr{{Name: "method-server"}},
	}
	service.Methods = []*MethodExpr{method}
	root.Services = []*ServiceExpr{service}

	method.Prepare()
	requireNoValidationErrors(t, method.Validate())
	requireInterceptorNames(t, method.ClientInterceptors, "method-client")
	requireInterceptorNames(t, method.ServerInterceptors, "method-server")

	method.Finalize()
	requireInterceptorNames(t, method.ClientInterceptors, "method-client", "service-client", "api-client")
	requireInterceptorNames(t, method.ServerInterceptors, "method-server", "service-server", "api-server")
}

func TestHTTPEndpointValidateDoesNotInferSessionCookieMappings(t *testing.T) {
	root := SetupTestDSL(t)
	root.API.HTTP.Headers = NewEmptyMappedAttributeExpr()
	root.API.HTTP.Cookies = NewEmptyMappedAttributeExpr()
	root.API.HTTP.Params = NewEmptyMappedAttributeExpr()
	scheme := &SchemeExpr{
		Kind:       APIKeyKind,
		SchemeName: "browser_session_key",
	}
	sessionAuth := &SessionAuthExpr{
		Name: "app_session",
		Transports: []*SessionTransportExpr{{
			Kind:     SessionCookieTransportKind,
			Scheme:   scheme,
			HTTPName: "__Host-browser_session",
		}},
	}
	payloadObj := &Object{}
	payloadObj.Set("message", &AttributeExpr{Type: String})
	service := &ServiceExpr{Name: "svc"}
	method := &MethodExpr{
		Name:         "show",
		Service:      service,
		Payload:      &AttributeExpr{Type: payloadObj},
		Result:       &AttributeExpr{Type: String},
		SessionAuths: []*SessionAuthExpr{sessionAuth},
	}
	service.Methods = []*MethodExpr{method}
	root.Services = []*ServiceExpr{service}
	httpService := root.API.HTTP.ServiceFor(service, root.API.HTTP)
	httpService.Headers = NewEmptyMappedAttributeExpr()
	httpService.Cookies = NewEmptyMappedAttributeExpr()
	httpService.Params = NewEmptyMappedAttributeExpr()

	endpoint := &HTTPEndpointExpr{
		MethodExpr: method,
		Service:    httpService,
		Routes:     []*RouteExpr{{Method: "POST", Path: "/show"}},
	}
	endpoint.Routes[0].Endpoint = endpoint
	httpService.HTTPEndpoints = []*HTTPEndpointExpr{endpoint}
	method.Prepare()
	endpoint.Prepare()

	require.Nil(t, endpoint.Cookies.Find("browser_session_key"))
	requireNoValidationErrors(t, endpoint.Validate())
	require.Nil(t, endpoint.Cookies.Find("browser_session_key"))

	endpoint.Finalize()
	require.NotNil(t, endpoint.Cookies.Find("browser_session_key"))
}

func requireNoValidationErrors(t *testing.T, err error) {
	t.Helper()

	var verr *eval.ValidationErrors
	require.True(t, errors.As(err, &verr))
	require.Empty(t, verr.Errors)
}

func requireInterceptorNames(t *testing.T, interceptors []*InterceptorExpr, names ...string) {
	t.Helper()

	actual := make([]string, 0, len(interceptors))
	for _, interceptor := range interceptors {
		actual = append(actual, interceptor.Name)
	}
	require.Equal(t, names, actual)
}
