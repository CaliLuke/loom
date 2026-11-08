package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestInformationalClassAnalysis(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		hints := Type("Hints", func() {
			Attribute("status", Int)
			Attribute("link", String)
			Required("status")
		})
		Service("analysis", func() {
			Method("show", func() {
				HTTP(func() {
					GET("/")
					Informational("hints", Status1XX, hints, func() {
						StatusField("status")
						Header("link:Link")
					})
					Response(StatusNoContent)
				})
			})
		})
	})
	var endpoint *EndpointData
	require.NotPanics(t, func() { endpoint = CreateHTTPServices(root).Get("analysis").Endpoint("show") })
	require.Len(t, endpoint.Informational, 1)
	response := endpoint.Informational[0].Response
	require.Equal(t, expr.HTTPStatusSelector{Class: expr.Status1XX}, response.Selector)
	require.Contains(t, response.Domain, 103)
	require.NotContains(t, response.Domain, 100)
	require.NotContains(t, response.Domain, 101)
	require.NotNil(t, response.ClientStatusField)
}
