package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"goa.design/goa/v3/dsl"
	"goa.design/goa/v3/http/codegen/openapi"
	openapiv3 "goa.design/goa/v3/http/codegen/openapi/v3"
)

func TestOpenAPIApiLevelSSEConfiguration(t *testing.T) {
	root := RunHTTPDSL(t, apiLevelSSEOpenAPIDSL)
	openapi.Definitions = make(map[string]*openapi.Schema)

	v3JSON := renderOpenAPIJSON(t, openapiv3.Files, root)
	doc := parseOpenAPIV3Document(t, v3JSON)

	pathItem, ok := doc.Paths.PathItems.Get("/watch")
	require.True(t, ok)
	require.NotNil(t, pathItem)
	require.NotNil(t, pathItem.Get)
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
