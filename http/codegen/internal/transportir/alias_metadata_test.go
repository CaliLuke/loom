package transportir_test

import (
	"testing"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/stretchr/testify/require"
)

func TestNormalizedAliasesRetainOccurrenceMetadata(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		choice := Type("Choice", OneOf(String, Int), func() {
			Untagged()
			Meta("openapi:discriminator:propertyName", "kind")
			Meta("openapi:discriminator:defaultMapping", "https://example.test/fallback")
			Meta("openapi:discriminator:extension:x-audit", `{"v":1}`)
			Meta("openapi:extension:x-schema", `true`)
		})
		Service("aliases", func() {
			Method("show", func() {
				Payload(choice, func() {
					Meta("openapi:discriminator:extension:x-audit", `{"v":2}`)
				})
				Result(choice)
				HTTP(func() {
					POST("/alias")
					Response(StatusOK)
				})
			})
		})
	})
	expected := expr.HTTPBodyProjectionMetadata(root.API.HTTP.Services[0].HTTPEndpoints[0].Body)
	service := transportir.BuildService(root.API.HTTP.Services[0])
	request := service.Endpoints[0].Request.Body
	require.IsType(t, &expr.Union{}, request.Type)
	require.Equal(t, []string{"kind"}, request.Meta["openapi:discriminator:propertyName"])
	require.Equal(t, []string{"https://example.test/fallback"}, request.Meta["openapi:discriminator:defaultMapping"])
	require.Equal(t, expected, request.Meta)
	value, present := request.Meta.Last("openapi:discriminator:extension:x-audit")
	require.True(t, present)
	require.Equal(t, `{"v":2}`, value)
	require.NotContains(t, request.Meta, "openapi:extension:x-schema")
	request.Meta["openapi:discriminator:propertyName"][0] = "local"
	require.Equal(t, []string{"kind"}, root.UserType("Choice").Attribute().Meta["openapi:discriminator:propertyName"])
}
