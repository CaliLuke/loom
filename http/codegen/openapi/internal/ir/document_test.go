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

func openAPIExampleValueForTest(attr *expr.AttributeExpr, raw any) (any, bool) {
	return openAPIExampleValue(attr, raw)
}
