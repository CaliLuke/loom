package openapiv3

import (
	"testing"

	"github.com/stretchr/testify/require"

	"goa.design/goa/v3/codegen"
	"goa.design/goa/v3/http/codegen/openapi/v3/testdata/dsls"
)

func TestNewUsesIRBuiltOperationBodies(t *testing.T) {
	const (
		serviceName = "test service"
		methodName  = "request_object_body"
	)

	root := codegen.RunDSL(t, dsls.RequestObjectBody(serviceName, methodName))
	spec := New(root)

	require.NotNil(t, spec)
	path := root.API.HTTP.Services[0].HTTPEndpoints[0].Routes[0].FullPaths()[0]
	operation := spec.Paths[path].Post
	require.NotNil(t, operation)
	require.NotNil(t, operation.RequestBody)
	require.True(t, operation.RequestBody.Value.Required)
	require.Contains(t, operation.RequestBody.Value.Content, "application/json")
	require.NotEmpty(t, operation.RequestBody.Value.Content["application/json"].Schema.Ref)
}
