package ir

import (
	"testing"

	"github.com/stretchr/testify/require"

	"goa.design/goa/v3/codegen"
	"goa.design/goa/v3/expr"
	"goa.design/goa/v3/http/codegen/openapi/v3/testdata/dsls"
)

func TestBuildDocumentIncludesRequestBodyAndResponses(t *testing.T) {
	const (
		serviceName = "test service"
		methodName  = "request_object_body"
	)

	root := codegen.RunDSL(t, dsls.RequestObjectBody(serviceName, methodName))
	doc := BuildDocument(root.API, root.Types, root.ResultTypes, WithExampleValue(openAPIExampleValueForTest))

	path := root.API.HTTP.Services[0].HTTPEndpoints[0].Routes[0].FullPaths()[0]
	operation := doc.Paths[path].Operations["POST"]
	require.NotNil(t, operation)
	require.NotNil(t, operation.RequestBody)
	require.True(t, operation.RequestBody.Required)
	require.Contains(t, operation.RequestBody.Content, "application/json")
	require.NotEmpty(t, operation.RequestBody.Content["application/json"].Schema.Ref)
	require.Contains(t, operation.Responses, "204")
	require.Equal(t, "No Content response.", operation.Responses["204"].Description)
}

func TestBuildDocumentCarriesErrorRemedyDescriptions(t *testing.T) {
	const (
		serviceName = "test service"
		methodName  = "error_remedy"
	)

	root := codegen.RunDSL(t, dsls.ErrorRemedyResponseBodyDSL(serviceName, methodName))
	doc := BuildDocument(root.API, root.Types, root.ResultTypes, WithExampleValue(openAPIExampleValueForTest))

	path := root.API.HTTP.Services[0].HTTPEndpoints[0].Routes[0].FullPaths()[0]
	operation := doc.Paths[path].Operations["POST"]
	require.NotNil(t, operation)
	require.Contains(t, operation.Responses, "400")
	require.Equal(t, "bad: Bad Request response. Remedy code: bad.fix. Safe message: Retry with a valid request. Retry hint: Correct the payload and retry.", operation.Responses["400"].Description)
}

func TestBuildOperationAddsResponseCookieHeader(t *testing.T) {
	const (
		serviceName = "test service"
		methodName  = "cookie_body"
	)

	root := codegen.RunDSL(t, dsls.MultiCookieResponseBodyDSL(serviceName, methodName))
	bodyTypes := BuildBodyTypes(root.API, root.Types, root.ResultTypes, WithExampleValue(openAPIExampleValueForTest))

	var endpoint *expr.HTTPEndpointExpr
	for _, svc := range root.API.HTTP.Services {
		if svc.Name() != serviceName {
			continue
		}
		endpoint = svc.Endpoint("other")
		break
	}
	require.NotNil(t, endpoint)

	operation := BuildOperation(endpoint, bodyTypes.Services[serviceName]["other"], root.API.ExampleGenerator, false)
	require.NotNil(t, operation)
	require.Contains(t, operation.Responses, "200")
	require.Contains(t, operation.Responses["200"].Headers, "Set-Cookie")
	require.Equal(t, "string", operation.Responses["200"].Headers["Set-Cookie"].Schema.Type)
}

func TestBuildOperationSuppressesStreamingResponseExamples(t *testing.T) {
	const (
		serviceName = "test service"
		methodName  = "streaming_object"
	)

	root := codegen.RunDSL(t, dsls.ObjectStreamingResponseBodyDSL(serviceName, methodName))
	bodyTypes := BuildBodyTypes(root.API, root.Types, root.ResultTypes, WithExampleValue(openAPIExampleValueForTest))

	var endpoint *expr.HTTPEndpointExpr
	for _, svc := range root.API.HTTP.Services {
		if svc.Name() != serviceName {
			continue
		}
		endpoint = svc.Endpoint(methodName)
		break
	}
	require.NotNil(t, endpoint)

	operation := BuildOperation(endpoint, bodyTypes.Services[serviceName][methodName], root.API.ExampleGenerator, false)
	require.NotNil(t, operation)
	require.Contains(t, operation.Responses, "101")
	response := operation.Responses["101"]
	require.NotNil(t, response)
	require.Contains(t, response.Content, "application/json")
	require.Nil(t, response.Content["application/json"].Example)
	require.Nil(t, response.Content["application/json"].Examples)
}

func openAPIExampleValueForTest(attr *expr.AttributeExpr, raw any) (any, bool) {
	return openAPIExampleValue(attr, raw)
}
