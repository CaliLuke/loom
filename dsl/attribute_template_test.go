package dsl_test

import (
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBodyTemplateEffectiveInheritance(t *testing.T) {
	for _, inline := range []bool{false, true} {
		name := "named"
		if inline {
			name = "inline"
		}
		t.Run(name, func(t *testing.T) {
			root := expr.RunDSL(t, func() {
				base := Type("BodyTemplateBase", func() {
					Attribute("count", Int, func() {
						Maximum(10)
						Default(7)
					})
					Attribute("name", String)
				})
				middle := Type("BodyTemplateMiddle", func() {
					Extend(base)
					Attribute("count", func() {
						Minimum(1)
					})
				})
				Service("template", func() {
					Method("create", func() {
						if inline {
							Payload(func() {
								Extend(middle)
							})
						} else {
							Payload(middle)
						}
						HTTP(func() {
							POST("/")
							Body(func() {
								Attribute("count:n")
							})
						})
					})
				})
			})
			endpoint := root.API.HTTP.Service("template").Endpoint("create")
			_, source := endpoint.MethodExpr.Payload.FindAttribute("count")
			_, body := endpoint.Body.FindAttribute("count:n")
			require.NotNil(t, source)
			require.NotNil(t, body)
			assert.Equal(t, expr.Int, body.Type)
			sourceContract, err := expr.EffectiveConstraintsFor(source)
			require.NoError(t, err)
			bodyContract, err := expr.EffectiveConstraintsFor(body)
			require.NoError(t, err)
			sourceDefault, sourceHasDefault := sourceContract.Default()
			bodyDefault, bodyHasDefault := bodyContract.Default()
			assert.True(t, sourceHasDefault)
			assert.True(t, bodyHasDefault)
			assert.Equal(t, 7, sourceDefault)
			assert.Equal(t, sourceDefault, bodyDefault)
			assert.Equal(t, sourceContract.Validation().Lowered(), bodyContract.Validation().Lowered())
			assert.Len(t, *expr.AsObject(endpoint.Body.Type), 1)
			assert.Len(t, *expr.AsObject(endpoint.MethodExpr.Payload.Type), 2)
		})
	}
}

func TestBodyTemplateResponseAndDocumentationContexts(t *testing.T) {
	root := expr.RunDSL(t, func() {
		fields := Type("TemplateContextFields", func() {
			Attribute("count", Int, func() {
				Maximum(10)
				Default(7)
			})
		})
		Service("contexts", func() {
			Method("show", func() {
				Result(func() {
					Extend(fields)
				})
				Error("invalid", fields)
				HTTP(func() {
					GET("/")
					Response(StatusOK, func() {
						Body(func() {
							Attribute("count")
						})
					})
					Response("invalid", StatusBadRequest, func() {
						Body(func() {
							Attribute("count")
						})
					})
				})
			})
			Method("document", func() {
				Payload(func() {
					Extend(fields)
				})
				Result(func() {
					Extend(fields)
				})
				HTTP(func() {
					POST("/raw")
					SkipRequestBodyEncodeDecode()
					Header("count")
					OpenAPIRequestBody(func() {
						Attribute("count")
					}, "application/json", true)
					Response(StatusOK, func() {
						OpenAPIBody(func() {
							Attribute("count")
						})
					})
				})
			})
		})
	})
	service := root.API.HTTP.Service("contexts")
	show, document := service.Endpoint("show"), service.Endpoint("document")
	for name, body := range map[string]*expr.AttributeExpr{
		"response":               show.Responses[0].Body,
		"error":                  show.HTTPErrors[0].Response.Body,
		"request documentation":  document.OpenAPIRequestBody,
		"response documentation": document.Responses[0].OpenAPIBody,
	} {
		t.Run(name, func(t *testing.T) {
			child := bodyChild(t, body, "count")
			assert.Equal(t, expr.Int, child.Type)
			assert.Equal(t, 7, child.DefaultValue)
			require.NotNil(t, child.Validation)
			require.NotNil(t, child.Validation.Maximum)
			maximum, err := expr.NewNumericBound(10)
			require.NoError(t, err)
			assert.Equal(t, maximum, *child.Validation.Maximum)
		})
	}
}

func TestBodyTemplatePreservesNamedIntrinsicConstraints(t *testing.T) {
	root := expr.RunDSL(t, func() {
		bounded := Type("IntrinsicCount", Int, func() {
			Minimum(5)
			Maximum(10)
		})
		Service("intrinsic", func() {
			Method("create", func() {
				Payload(func() {
					Attribute("count", bounded)
				})
				HTTP(func() {
					POST("/")
					Body(func() {
						Attribute("count", func() {
							Minimum(1)
						})
					})
				})
			})
		})
	})
	child := bodyChild(t, root.API.HTTP.Service("intrinsic").Endpoint("create").Body, "count")
	contract, err := expr.EffectiveConstraintsFor(child)
	require.NoError(t, err)
	minimum, err := expr.NewNumericBound(5)
	require.NoError(t, err)
	require.Equal(t, minimum, *contract.Validation().Lowered().Minimum)
}
