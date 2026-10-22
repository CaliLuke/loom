package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestDocumentationUnionSchemas(t *testing.T) {
	for _, shape := range []string{"scalar-object", "inline-scalar-object", "nested-array-object"} {
		for _, request := range []bool{false, true} {
			name := shape + "/response"
			if request {
				name = shape + "/request"
			}
			t.Run(name, func(t *testing.T) {
				var source expr.DataType
				root := expr.RunDSL(t, func() {
					leaf := Type("Leaf", func() {
						Attribute("label", String)
						Required("label")
					})
					choice := Type("Choice", OneOf(String, leaf))
					item := Type("Item", func() {
						Attribute("choice", choice)
						Required("choice")
					})
					source = Type("Documented", OneOf(String, item))
					if shape == "inline-scalar-object" {
						source = OneOf(String, item)
					}
					if shape == "nested-array-object" {
						items := Type("Items", ArrayOf(item))
						page := Type("Page", func() {
							Attribute("results", items)
							Required("results")
						})
						source = Type("DocumentedPage", OneOf(items, page))
					}
					Service("documentation", func() {
						Method("raw", func() {
							HTTP(func() {
								POST("/raw")
								if request {
									SkipRequestBodyEncodeDecode()
									OpenAPIRequestBody(source, "application/json", true, Untagged)
								} else {
									SkipResponseBodyEncodeDecode()
									Response(StatusOK, func() {
										OpenAPIBody(source, Untagged)
									})
								}
							})
						})
						Method("typed", func() {
							Result(source)
							HTTP(func() {
								GET("/typed")
							})
						})
					})
				})
				endpoint := root.API.HTTP.Services[0].HTTPEndpoints[0]
				body := endpoint.OpenAPIRequestBody
				if !request {
					body = endpoint.Responses[0].OpenAPIBody
				}
				require.True(t, expr.AsUnion(body.Type).Untagged)
				require.False(t, expr.AsUnion(source).Untagged, "documentation must not change the source codec")
				require.False(t, expr.AsUnion(root.Services[0].Methods[1].Result.Type).Untagged)
			})
		}
	}
}
