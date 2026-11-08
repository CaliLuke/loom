package transportir_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	testcodegen "github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestResponseSelectorProjection(t *testing.T) {
	root := testcodegen.RunDSL(t, testdata.OpenAPIStatusSelectorDSL)
	for _, endpoint := range root.API.HTTP.Services[0].HTTPEndpoints {
		t.Run(endpoint.Name(), func(t *testing.T) {
			var projected *transportir.Endpoint
			require.NotPanics(t, func() {
				projected = transportir.BuildEndpoint(endpoint)
			})
			analysis := transportir.AnalyzeResponseContractCases(projected)
			require.True(t, analysis.Supported())
			require.Len(t, analysis.Cases, 5)
			require.Equal(t, expr.HTTPStatusSelector{Code: 204}, analysis.Cases[0].Selector)
			require.Equal(t, []int{204}, analysis.Cases[0].Domain)
			for i, er := range endpoint.HTTPErrors {
				c := analysis.Cases[i+1]
				require.Equal(t, er.Response.Status, c.Selector)
				require.Equal(t, er.Response.StatusDomain(), c.Domain)
				require.Contains(t, c.ID, "."+c.Selector.OpenAPIKey())
				if c.Selector.Class == expr.Status4XX {
					require.NotContains(t, c.Domain, 401)
					require.Contains(t, c.Domain, 400)
				}
				if c.Selector.Class == expr.StatusFallback {
					require.NotContains(t, c.Domain, 204)
					require.NotContains(t, c.Domain, 304)
					require.NotContains(t, c.Domain, 400)
					require.Contains(t, c.Domain, 200)
				}
			}
			analysis.Cases[1].Domain[0] = 999
			require.NotContains(t, transportir.AnalyzeResponseContractCases(projected).Cases[1].Domain, 999)
		})
	}
}

func TestSuccessSelectorProjection(t *testing.T) {
	root := testcodegen.RunDSL(t, func() {
		Service("selected", func() {
			Method("show", func() {
				Result(func() {
					Attribute("status", Int)
					Attribute("message", String)
					Required("status", "message")
				})
				HTTP(func() {
					GET("/")
					Response(StatusOK, func() {
						StatusField("status")
					})
					Response(Status2XX, func() {
						StatusField("status")
					})
				})
			})
		})
	})
	projected := transportir.BuildEndpoint(root.API.HTTP.Services[0].HTTPEndpoints[0])
	analysis := transportir.AnalyzeResponseContractCases(projected)
	require.True(t, analysis.Supported())
	require.Len(t, analysis.Cases, 2)
	require.Equal(t, expr.HTTPStatusSelector{Code: 200}, analysis.Cases[0].Selector)
	require.Equal(t, []int{200}, analysis.Cases[0].Domain)
	require.Equal(t, expr.HTTPStatusSelector{Class: expr.Status2XX}, analysis.Cases[1].Selector)
	require.Contains(t, analysis.Cases[1].ID, ".2XX")
	require.NotContains(t, analysis.Cases[1].Domain, 200)
	require.NotContains(t, analysis.Cases[1].Domain, 204)
	require.Contains(t, analysis.Cases[1].Domain, 299)
	projected.Response.Responses[1].Domain[0] = 999
	require.NotContains(t, root.API.HTTP.Services[0].HTTPEndpoints[0].Responses[1].StatusDomain(), 999)
}
