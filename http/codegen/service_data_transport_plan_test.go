package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestRequestDecodePlanSummarizesBoundElements(t *testing.T) {
	root := RunHTTPDSL(t, testdata.PayloadBodyQueryPathObjectValidateDSL)
	endpoint := firstHTTPEndpoint(t, root)

	plan := endpoint.Payload.Request.DecodePlan

	require.NotNil(t, plan)
	require.True(t, plan.HasElements)
	require.True(t, plan.HasPathParams)
	require.True(t, plan.HasQueryParams)
	require.False(t, plan.HasHeaders)
	require.False(t, plan.HasCookies)
	require.True(t, plan.MustValidate)
}

func TestResponseEncodePlanSummarizesViewedBodyVariants(t *testing.T) {
	root := RunHTTPDSL(t, testdata.ResultBodyMultipleViewsDSL)
	endpoint := firstHTTPEndpoint(t, root)
	response := endpoint.Result.Responses[0]

	plan := response.EncodePlan

	require.NotNil(t, plan)
	require.True(t, plan.HasBody)
	require.Greater(t, plan.BodyCount, 1)
	require.NotNil(t, plan.FirstBody)
	require.True(t, plan.HasMultipleBodies)
	require.True(t, plan.UseViewedBodySwitch)
	require.False(t, plan.NeedsProblemSource)
}

func firstHTTPEndpoint(t *testing.T, root *expr.RootExpr) *EndpointData {
	t.Helper()
	require.NotEmpty(t, root.API.HTTP.Services)
	services := CreateHTTPServices(root)
	service := services.Get(root.API.HTTP.Services[0].Name())
	require.NotNil(t, service)
	require.NotEmpty(t, service.Endpoints)
	return service.Endpoints[0]
}
