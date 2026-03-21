package ir

import (
	"testing"

	"github.com/stretchr/testify/require"

	"goa.design/goa/v3/codegen"
	"goa.design/goa/v3/expr"
	"goa.design/goa/v3/http/codegen/openapi/v3/testdata/dsls"
	"goa.design/goa/v3/http/codegen/testdata"
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
	require.NotNil(t, operation.RequestBody.Value)
	require.True(t, operation.RequestBody.Value.Required)
	require.Contains(t, operation.RequestBody.Value.Content, "application/json")
	require.NotEmpty(t, operation.RequestBody.Value.Content["application/json"].Schema.Ref)
	require.Contains(t, operation.Responses, "204")
	require.NotNil(t, operation.Responses["204"])
	require.NotNil(t, operation.Responses["204"].Value)
	require.Equal(t, "No Content response.", operation.Responses["204"].Value.Description)
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
	require.NotNil(t, operation.Responses["400"])
	require.NotNil(t, operation.Responses["400"].Value)
	require.Equal(t, "bad: Bad Request response. Remedy code: bad.fix. Safe message: Retry with a valid request. Retry hint: Correct the payload and retry.", operation.Responses["400"].Value.Description)
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
	require.NotNil(t, operation.Responses["200"])
	require.NotNil(t, operation.Responses["200"].Value)
	require.Contains(t, operation.Responses["200"].Value.Headers, "Set-Cookie")
	require.NotNil(t, operation.Responses["200"].Value.Headers["Set-Cookie"])
	require.NotNil(t, operation.Responses["200"].Value.Headers["Set-Cookie"].Value)
	require.Equal(t, "string", operation.Responses["200"].Value.Headers["Set-Cookie"].Value.Schema.Type)
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
	require.NotNil(t, response.Value)
	require.Empty(t, response.Value.Content)
}

func TestBuildDocumentComponentizesRepeatedContractNodes(t *testing.T) {
	root := codegen.RunDSL(t, testdata.OpenAPIReusableComponentsDSL)

	doc := BuildDocument(root.API, root.Types, root.ResultTypes, WithExampleValue(openAPIExampleValueForTest))
	require.NotNil(t, doc)
	require.NotNil(t, doc.Components)
	require.NotEmpty(t, doc.Components.RequestBodies)
	require.NotEmpty(t, doc.Components.Responses)
	require.NotEmpty(t, doc.Components.Headers)
	require.NotEmpty(t, doc.Components.Examples)

	signin := doc.Paths["/auth/signin"].Operations["POST"]
	refresh := doc.Paths["/auth/refresh"].Operations["POST"]
	require.NotNil(t, signin)
	require.NotNil(t, refresh)
	require.NotNil(t, signin.RequestBody)
	require.NotNil(t, refresh.RequestBody)
	require.NotEmpty(t, signin.RequestBody.Ref)
	require.Equal(t, signin.RequestBody.Ref, refresh.RequestBody.Ref)
}

func openAPIExampleValueForTest(attr *expr.AttributeExpr, raw any) (any, bool) {
	return openAPIExampleValue(attr, raw)
}
