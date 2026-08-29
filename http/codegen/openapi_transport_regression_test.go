package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
)

func TestOpenAPIApiLevelSSEConfiguration(t *testing.T) {
	root := RunHTTPDSL(t, apiLevelSSEOpenAPIDSL)

	v3JSON := renderOpenAPIJSON(t, openapiv3.Files, root)
	doc := parseOpenAPIV3Document(t, v3JSON)

	pathItem, ok := doc.Paths.PathItems.Get("/watch")
	require.True(t, ok)
	require.NotNil(t, pathItem)
	require.NotNil(t, pathItem.Get)

	responses := pathItem.Get.Responses
	require.NotNil(t, responses)
	response, ok := responses.Codes.Get("200")
	require.True(t, ok)
	require.NotNil(t, response)
	_, ok = response.Content.Get("text/event-stream")
	require.True(t, ok)
	_, ok = response.Content.Get("application/json")
	require.False(t, ok)
}

var apiLevelSSEOpenAPIDSL = func() {
	dsl.API("apiLevelSSEOpenAPI", func() {
		dsl.HTTP(func() {
			dsl.ServerSentEvents("event")
		})
	})

	dsl.Service("streams", func() {
		dsl.Method("watch", func() {
			dsl.StreamingResult(func() {
				dsl.Attribute("event", dsl.String)
				dsl.Required("event")
			})
			dsl.HTTP(func() {
				dsl.GET("/watch")
			})
		})
	})
}
