package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

func TestNullableObjectBodyProjection(t *testing.T) {
	endpoint := firstEndpointData(t, func() {
		value := dsl.Type("Value", func() {
			dsl.Nullable()
			dsl.Attribute("message", dsl.String)
		})
		dsl.Service("objects", func() {
			dsl.Method("show", func() {
				dsl.Result(value)
				dsl.HTTP(func() {
					dsl.GET("/")
					dsl.Response(dsl.StatusOK, func() {
						dsl.Body(func() {
							dsl.Attribute("message")
						})
					})
				})
			})
		})
	})
	response := endpoint.Result.Responses[0]
	require.Contains(t, response.EncodePlan.SourcePreparation(nil, ""), "if !responseHasValue")
	require.Equal(t, "&responseValue", response.ServerBody[0].Init.ServerArgs[0].Ref)
}
