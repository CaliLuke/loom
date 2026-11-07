package dsl_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestBodyTemplatePropertySelection(t *testing.T) {
	cases := []struct {
		name   string
		kind   expr.DataType
		source func()
		refine func()
		local  func()
		check  func(*testing.T, *expr.AttributeExpr, *expr.AttributeExpr)
	}{
		{
			name: "explicit empty pattern",
			kind: String,
			source: func() {
				Pattern("^seed$")
				MinLength(2)
			},
			local: func() {
				Pattern("")
			},
			check: func(t *testing.T, body, source *expr.AttributeExpr) {
				constraints, err := expr.EffectiveConstraintsFor(body)
				require.NoError(t, err)
				require.Empty(t, constraints.Validation().Clauses())
				require.Equal(t, 2, *constraints.Validation().Lowered().MinLength)
				sourceConstraints, err := expr.EffectiveConstraintsFor(source)
				require.NoError(t, err)
				require.Len(t, sourceConstraints.Validation().Clauses(), 1)
				require.Equal(t, "^seed$", sourceConstraints.Validation().Clauses()[0].Value)
			},
		},
		{
			name: "explicit empty title",
			kind: String,
			source: func() {
				Title("Source title")
			},
			local: func() {
				Title("")
			},
			check: func(t *testing.T, body, source *expr.AttributeExpr) {
				require.Empty(t, body.Title)
				require.Equal(t, "Source title", source.Title)
			},
		},
		{
			name: "metadata remove readd order",
			kind: String,
			source: func() {
				Meta("test:trail", "source")
				Meta("test:remove", "source")
			},
			local: func() {
				RemoveMeta("test:trail")
				Meta("test:trail", "first")
				Meta("test:trail", "second")
				Meta("test:remove", "discard")
				RemoveMeta("test:remove")
			},
			check: func(t *testing.T, body, source *expr.AttributeExpr) {
				require.Equal(t, []string{"first", "second"}, body.Meta["test:trail"])
				require.NotContains(t, body.Meta, "test:remove")
				require.Equal(t, []string{"source"}, source.Meta["test:trail"])
				require.Equal(t, []string{"source"}, source.Meta["test:remove"])
			},
		},
		{
			name: "inherited nullable permits null example",
			kind: String,
			source: func() {
				Nullable()
			},
			local: func() {
				Example(Null())
			},
			check: func(t *testing.T, body, source *expr.AttributeExpr) {
				require.True(t, body.Nullable)
				require.Len(t, body.UserExamples, 1)
				require.True(t, body.UserExamples[0].ExplicitNull)
				require.Nil(t, body.UserExamples[0].Value)
				require.Empty(t, source.UserExamples)
			},
		},
		{
			name: "explicit zero default",
			kind: Int,
			source: func() {
				Default(5)
			},
			local: func() {
				Default(0)
			},
			check: func(t *testing.T, body, source *expr.AttributeExpr) {
				constraints, err := expr.EffectiveConstraintsFor(body)
				require.NoError(t, err)
				value, present := constraints.Default()
				require.True(t, present)
				require.Equal(t, 0, value)
				require.Equal(t, 5, source.DefaultValue)
			},
		},
		{
			name: "local minimum overrides template and retains inherited maximum",
			kind: Int,
			source: func() {
				Minimum(0)
				Maximum(10)
			},
			refine: func() {
				Minimum(5)
			},
			local: func() {
				Minimum(1)
			},
			check: func(t *testing.T, body, source *expr.AttributeExpr) {
				constraints, err := expr.EffectiveConstraintsFor(body)
				require.NoError(t, err)
				require.True(t, constraints.LowerBound().Present)
				require.Equal(t, "1", constraints.LowerBound().Value.String())
				require.True(t, constraints.UpperBound().Present)
				require.Equal(t, "10", constraints.UpperBound().Value.String())
				sourceConstraints, err := expr.EffectiveConstraintsFor(source)
				require.NoError(t, err)
				require.Equal(t, "5", sourceConstraints.LowerBound().Value.String())
				require.Equal(t, "10", sourceConstraints.UpperBound().Value.String())
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var payload expr.UserType
			root := expr.RunDSL(t, func() {
				base := Type("PropertyBase", func() {
					Attribute("value", tc.kind, tc.source)
				})
				payload = base
				if tc.refine != nil {
					payload = Type("PropertyPayload", func() {
						Extend(base)
						Attribute("value", tc.refine)
					})
				}
				Service("svc", func() {
					Method("create", func() {
						Payload(payload)
						HTTP(func() {
							POST("/")
							Body(func() {
								Attribute("value", tc.local)
							})
						})
					})
				})
			})
			body := root.API.HTTP.Service("svc").Endpoint("create").Body
			tc.check(t, bodyChild(t, body, "value"), expr.AsObject(payload).Attribute("value"))
		})
	}
}

func TestBodyTemplatePropertyRejections(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		payload := Type("AdditionalPayload", func() {
			Attribute("value", func() {
				AdditionalAttributes(String)
			})
		})
		Service("svc", func() {
			Method("create", func() {
				Payload(payload)
				HTTP(func() {
					POST("/")
					Body(func() {
						Attribute("value", func() {
							AdditionalAttributes(Int)
						})
					})
				})
			})
		})
	})
	require.ErrorContains(t, err, "AdditionalAttributes is already declared for this object")
}
