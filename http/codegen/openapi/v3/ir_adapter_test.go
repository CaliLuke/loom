package openapiv3

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/v3/codegen"
	openapiir "github.com/CaliLuke/loom/v3/http/codegen/openapi/internal/ir"
	"github.com/CaliLuke/loom/v3/http/codegen/openapi/v3/testdata/dsls"
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

func TestReusableComponentsFromIRSkipsNilValues(t *testing.T) {
	components := reusableComponentsFromIR(&openapiir.Components{
		Parameters: map[string]*openapiir.ParameterRef{
			"Nil": nil,
		},
		Headers: map[string]*openapiir.HeaderRef{
			"Nil": nil,
		},
		RequestBodies: map[string]*openapiir.RequestBodyRef{
			"Nil": nil,
		},
		Responses: map[string]*openapiir.ResponseRef{
			"Nil": nil,
		},
		Examples: map[string]*openapiir.ExampleRef{
			"Nil": nil,
		},
	})

	require.Nil(t, components.Parameters)
	require.Nil(t, components.Headers)
	require.Nil(t, components.RequestBodies)
	require.Nil(t, components.Responses)
	require.Nil(t, components.Examples)
}

func TestResponseFromIRCarriesLinks(t *testing.T) {
	response := responseFromIR(&openapiir.Response{
		Description: "Accepted response.",
		Links: map[string]*openapiir.ResponseLinkRef{
			"self": {
				Value: &openapiir.ResponseLink{
					OperationID: "thread_ops.get_thread",
					Parameters: map[string]any{
						"thread_id": "$response.body#/thread_id",
					},
				},
			},
		},
	})

	require.NotNil(t, response)
	require.Contains(t, response.Links, "self")
	require.Equal(t, "thread_ops.get_thread", response.Links["self"].Value.OperationID)
	require.Equal(t, "$response.body#/thread_id", response.Links["self"].Value.Parameters["thread_id"])
}
