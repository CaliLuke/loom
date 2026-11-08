package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestStatusDocumentationPublishedMembers(t *testing.T) {
	for _, shape := range []string{"reference only", "member", "mapped member", "composed member", "selected"} {
		t.Run(shape, func(t *testing.T) {
			design := func() {
				value := Type("StatusValue", func() {
					Attribute("status", Int)
					Attribute("message", String)
					Required("status")
				})
				Service("statusdoc", func() {
					Method("read", func() {
						Result(value)
						HTTP(func() {
							GET("/")
							Response(StatusOK, func() {
								StatusField("status")
								switch shape {
								case "reference only":
									OpenAPIBody(func() {
										Attribute("message")
									})
								case "member":
									OpenAPIBody(func() {
										Attribute("status")
									})
								case "mapped member":
									OpenAPIBody(func() {
										Attribute("status:wire")
									})
								case "composed member":
									OpenAPIBody(func() {
										Extend(value)
									})
								case "selected":
									OpenAPIBody("status")
								}
							})
						})
					})
				})
			}
			if shape != "reference only" {
				require.ErrorContains(t, expr.RunInvalidDSL(t, design), "explicit body binding")
				return
			}
			root := expr.RunDSL(t, design)
			body := root.API.HTTP.Service("statusdoc").Endpoint("read").Responses[0].OpenAPIBody
			require.NotNil(t, body.Find("status"), "template lookup still finds its source")
			require.Nil(t, expr.AsObject(body.Type).Attribute("status"), "source status is not published")
			require.NotNil(t, expr.AsObject(body.Type).Attribute("message"))
		})
	}
}
