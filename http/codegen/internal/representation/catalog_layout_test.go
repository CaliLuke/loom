package representation

import (
	"fmt"
	"testing"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/stretchr/testify/require"
)

func TestErrorCatalogProjectionLayoutScaling(t *testing.T) {
	for _, methods := range []int{2, 4, 8} {
		t.Run(fmt.Sprintf("methods_%d", methods), func(t *testing.T) {
			root := codegen.RunDSL(t, func() {
				dsl.API("catalog", func() {
					for index := range 8 {
						dsl.Error(fmt.Sprintf("failure_%d", index), dsl.ErrorResult)
					}
					dsl.HTTP(func() {
						for index := range 8 {
							dsl.Response(fmt.Sprintf("failure_%d", index), 400)
						}
					})
				})
				dsl.Service("catalog", func() {
					for index := range 8 {
						dsl.Error(fmt.Sprintf("failure_%d", index))
					}
					for index := range methods {
						dsl.Method(fmt.Sprintf("method_%d", index), func() {
							dsl.Result(dsl.String)
							dsl.HTTP(func() {
								dsl.GET(fmt.Sprintf("/method/%d", index))
							})
						})
					}
				})
			})
			prepared, err := PrepareService(root.API.HTTP.Services[0], nil, nil)
			require.NoError(t, err)
			for _, endpoint := range prepared.Endpoints {
				require.Len(t, endpoint.Response.ErrorResponses, 8)
				for _, response := range endpoint.Response.ErrorResponses {
					target := response.BodyValue
					require.NoError(t, target.Error)
					context := target.ProjectionContext
					require.NotNil(t, context.TypeLayouts)
					require.Same(t, prepared.SchemaTypeLayouts, context.TypeLayouts)
					entries := len(context.JSONPresenceTypes) + len(context.PresencePointerTypes) + len(context.PresenceUseDefaultTypes)
					require.LessOrEqual(t, entries, 3, "a response must not retain service-wide layout copies")
				}
			}
		})
	}
}

func TestPreparedStreamSchemaSharesServiceLayouts(t *testing.T) {
	root := codegen.RunDSL(t, testdata.StreamingPayloadResultCollectionWithExplicitViewDSL)
	prepared, err := PrepareService(root.API.HTTP.Services[0], nil, nil)
	require.NoError(t, err)
	endpoint := prepared.Endpoints[0]
	require.NotNil(t, endpoint.Stream)
	for range 2 {
		target := PrepareStreamSchema(endpoint, endpoint.Stream.ResponseMessage, false)
		require.NotNil(t, target)
		require.NoError(t, target.Error)
		require.Same(t, prepared.SchemaTypeLayouts, target.ProjectionContext.TypeLayouts)
	}
}

func TestUnpreparedStreamSchemaCapturesFreshLayouts(t *testing.T) {
	root := codegen.RunDSL(t, testdata.StreamingPayloadResultCollectionWithExplicitViewDSL)
	prepared, err := PrepareService(root.API.HTTP.Services[0], nil, nil)
	require.NoError(t, err)
	endpoint := prepared.Endpoints[0]
	prepared.SchemaTypeLayouts = nil
	first := PrepareStreamSchema(endpoint, endpoint.Stream.ResponseMessage, false)
	require.NoError(t, first.Error)
	prepared.Endpoints = append(prepared.Endpoints, &transportir.Endpoint{
		Request:  &transportir.Request{Body: endpoint.Stream.ResponseMessage},
		Response: &transportir.Response{},
	})
	second := PrepareStreamSchema(endpoint, endpoint.Stream.ResponseMessage, false)
	require.NoError(t, second.Error)
	require.NotSame(t, first.ProjectionContext.TypeLayouts, second.ProjectionContext.TypeLayouts,
		"an unprepared mutable service must not acquire a persistent snapshot")
	require.Nil(t, prepared.SchemaTypeLayouts)
}
