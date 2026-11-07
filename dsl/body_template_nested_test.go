package dsl_test

import (
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBodyTemplateNestedRefinements(t *testing.T) {
	callbacks := 0
	scalar := func() {
		Minimum(0)
		Maximum(10)
		Default(5)
	}
	root := expr.RunDSL(t, func() {
		payload := Type("NestedTemplatePayload", func() {
			Attribute("object", func() {
				Attribute("count", Int, scalar)
				Attribute("note", String)
			})
			Attribute("values", ArrayOf(Int, scalar))
			Attribute("lookup", MapOf(String, Int), func() {
				Key(func() {
					MinLength(2)
				})
				Elem(scalar)
			})
		})
		Service("nested", func() {
			for _, name := range []string{"local", "plain"} {
				Method(name, func() {
					Payload(payload)
					HTTP(func() {
						POST("/" + name)
						Body(func() {
							callbacks++
							if name == "plain" {
								Attribute("object")
								Attribute("values")
								Attribute("lookup")
								return
							}
							Attribute("object", func() {
								Attribute("count", func() {
									Minimum(1)
									Default(7)
								})
								Attribute("count", func() {
									Maximum(9)
								})
							})
							Attribute("values", func() {
								Elem(func() {
									Minimum(1)
								})
								Elem(func() {
									Maximum(9)
									Default(7)
								})
							})
							Attribute("lookup", func() {
								Key(func() {
									Pattern("^x")
								})
								Elem(func() {
									Minimum(1)
									Maximum(9)
									Default(7)
								})
							})
						})
					})
				})
			}
		})
	})
	assert.Equal(t, 2, callbacks)
	service := root.API.HTTP.Service("nested")
	for _, name := range []string{"local", "plain"} {
		endpoint := service.Endpoint(name)
		body := endpoint.Body
		for _, pair := range []struct {
			label     string
			attribute *expr.AttributeExpr
		}{
			{"object", bodyChild(t, bodyChild(t, body, "object"), "count")},
			{"array", expr.AsArray(bodyChild(t, body, "values").Type).ElemType},
			{"map", expr.AsMap(bodyChild(t, body, "lookup").Type).ElemType},
		} {
			t.Run(name+"/"+pair.label, func(t *testing.T) {
				contract, err := expr.EffectiveConstraintsFor(pair.attribute)
				require.NoError(t, err)
				lower, upper, value := 0, 10, 5
				if name == "local" {
					lower, upper, value = 1, 9, 7
				}
				validation := contract.Validation().Lowered()
				require.NotNil(t, validation.Minimum)
				require.NotNil(t, validation.Maximum)
				wantLow, err := expr.NewNumericBound(lower)
				require.NoError(t, err)
				wantHigh, err := expr.NewNumericBound(upper)
				require.NoError(t, err)
				assert.Equal(t, wantLow, *validation.Minimum)
				assert.Equal(t, wantHigh, *validation.Maximum)
				got, present := contract.Default()
				assert.True(t, present)
				assert.Equal(t, value, got)
			})
		}
		assert.Len(t, *expr.AsObject(bodyChild(t, body, "object").Type), 2)
		key := expr.AsMap(bodyChild(t, body, "lookup").Type).KeyType
		require.NotNil(t, key.Validation.MinLength)
		assert.Equal(t, 2, *key.Validation.MinLength)
		if name == "local" {
			assert.Equal(t, "^x", key.Validation.Pattern)
		} else {
			assert.Empty(t, key.Validation.Pattern)
		}
	}
	source := bodyChild(t, service.Endpoint("local").MethodExpr.Payload, "object")
	assert.Equal(t, 5, bodyChild(t, source, "count").DefaultValue)
}

func TestBodyTemplateAuthoredComposition(t *testing.T) {
	for _, extend := range []bool{false, true} {
		name := "reference"
		if extend {
			name = "extend"
		}
		t.Run(name, func(t *testing.T) {
			root := expr.RunDSL(t, func() {
				extra := Type("TemplateExtra", func() {
					Meta("trace", "base")
					Attribute("local", Int, func() {
						Maximum(10)
					})
				})
				Service("edges", func() {
					Method("create", func() {
						Payload(func() {
							Attribute("object", func() {
								Attribute("existing", String)
								Meta("trace", "source")
							})
						})
						HTTP(func() {
							POST("/")
							Body(func() {
								Attribute("object", func() {
									if extend {
										RemoveMeta("trace")
										Extend(extra)
									} else {
										Reference(extra)
										Attribute("local")
									}
								})
							})
						})
					})
				})
			})
			endpoint := root.API.HTTP.Service("edges").Endpoint("create")
			nested := bodyChild(t, endpoint.Body, "object")
			local := bodyChild(t, nested, "local")
			assert.Equal(t, expr.Int, local.Type)
			effective, err := expr.EffectiveConstraintsFor(local)
			require.NoError(t, err)
			require.NotNil(t, effective.Validation().Lowered().Maximum)
			assert.Len(t, *expr.AsObject(bodyChild(t, endpoint.MethodExpr.Payload, "object").Type), 1)
			if extend {
				assert.Len(t, nested.CompositionBases(), 1)
				assert.NotContains(t, nested.Meta, "trace")
				assert.Equal(t, []string{"source"}, bodyChild(t, endpoint.MethodExpr.Payload, "object").Meta["trace"])
			} else {
				assert.Len(t, nested.References, 1)
			}
		})
	}
}
