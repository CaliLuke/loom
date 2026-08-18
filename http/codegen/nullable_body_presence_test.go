package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

func TestNullableNamedResponseBodyUsesPresenceValueType(t *testing.T) {
	endpoint := firstEndpointData(t, nullableNamedResponseBodyDSL)
	require.Len(t, endpoint.Result.Responses, 1)
	response := endpoint.Result.Responses[0]
	require.Len(t, response.ServerBody, 1)
	require.NotNil(t, response.ClientBody)

	const valueRef = "loom.Nullable[GetWidgetResponseBody]"
	require.Equal(t, valueRef, response.ServerBody[0].ValueRef)
	require.Equal(t, valueRef, response.ServerBody[0].Ref)
	require.Equal(t, valueRef, response.ServerBody[0].Init.ReturnTypeRef)
	require.Equal(t, valueRef, response.ClientBody.ValueRef)
	require.Equal(t, valueRef, response.ClientBody.Ref)
	require.Equal(t, "body", response.ResultInit.ClientArgs[0].Ref)
}

func nullableNamedResponseBodyDSL() {
	widget := Type("Widget", func() {
		Attribute("name", String)
		Required("name")
	})
	nullableWidget := Type("NullableWidget", widget, func() {
		Nullable()
	})
	Service("Widgets", func() {
		Method("GetWidget", func() {
			Result(nullableWidget, func() {
				Nullable()
			})
			HTTP(func() {
				GET("/widget")
				Response(StatusOK)
			})
		})
	})
}
