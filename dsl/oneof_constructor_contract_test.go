package dsl_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestOneOfConstructorRejectsDiscardedDirectives(t *testing.T) {
	for _, tc := range []struct {
		name string
		dsl  func()
	}{
		{"Nullable", Nullable},
		{"Reference", func() {
			Reference(&expr.Object{})
		}},
		{"Extend", func() {
			Extend(&expr.Object{})
		}},
		{"Description", func() {
			Description("a choice")
		}},
		{"Title", func() {
			Title("A choice")
		}},
		{"Docs", func() {
			Docs(func() {
				URL("https://example.com/choices")
			})
		}},
		{"Example", func() {
			Example("chosen")
		}},
		{"Enum", func() {
			Enum("chosen")
		}},
		{"object policy", func() {
			Meta("openapi:additionalProperties", "false")
		}},
		{"schema annotation", func() {
			Meta("openapi:deprecated", "true")
		}},
		{"custom metadata", func() {
			Meta("plugin:hint", "value")
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, func() {
				Type("Choice", OneOf(func() {
					Untagged()
					Attribute("Text", String)
					Attribute("Number", Int)
					tc.dsl()
				}))
			})
			require.ErrorContains(t, err, "OneOf constructor")
		})
	}
}

func TestOneOfConstructorRetainsSupportedSettings(t *testing.T) {
	root := expr.RunDSL(t, func() {
		tagged := OneOf(func() {
			Attribute("Text", String)
			Attribute("Number", Int)
			Meta("oneof:typename", "WireChoice")
			Meta("oneof:type:field", "kind")
			Meta("oneof:value:field", "data")
		})
		require.Equal(t, "WireChoice", expr.AsUnion(tagged).TypeName)
		Type("Tagged", tagged)
		Type("Choice", OneOf(func() {
			Untagged()
			Attribute("Text", String, func() {
				Nullable()
				Pattern("^[a-z]+$")
				Description("text branch")
				Example("chosen")
			})
			Attribute("Number", Int, func() {
				Default(7)
			})
			Meta("openapi:unevaluatedProperties", "false")
		}), func() {
			Nullable()
			Title("Choice title")
			Description("choice description")
			Meta("openapi:deprecated", "true")
			Example("chosen")
		})
	})
	tagged := expr.AsUnion(root.UserType("Tagged"))
	require.Equal(t, "Tagged", tagged.TypeName)
	require.Equal(t, "kind", tagged.TypeKey)
	require.Equal(t, "data", tagged.ValueKey)
	choice := root.UserType("Choice").Attribute()
	require.True(t, expr.IsNullable(choice))
	require.Equal(t, "Choice title", choice.Title)
	require.Equal(t, "choice description", choice.Description)
	require.Equal(t, []string{"true"}, choice.Meta["openapi:deprecated"])
	require.Len(t, choice.UserExamples, 1)
	union := expr.AsUnion(choice.Type)
	require.True(t, union.Untagged)
	require.True(t, union.Closed())
	branch := union.Values[0].Attribute
	require.True(t, expr.IsNullable(branch))
	require.Equal(t, "^[a-z]+$", branch.Validation.Pattern)
	require.Equal(t, "text branch", branch.Description)
	require.Len(t, branch.UserExamples, 1)
	require.Equal(t, 7, union.Values[1].Attribute.DefaultValue)
}

func TestOneOfConstructorOccurrenceDirectives(t *testing.T) {
	choice := func() expr.DataType {
		return OneOf(func() {
			Untagged()
			Attribute("Text", String)
			Attribute("Number", Int)
		})
	}
	occurrence := func() {
		Nullable()
		Description("choice occurrence")
		Title("Choice")
		Meta("openapi:deprecated", "true")
		Example("chosen")
	}
	root := expr.RunDSL(t, func() {
		Type("Choice", choice(), occurrence)
		Type("Envelope", func() {
			Attribute("value", choice(), occurrence)
		})
		Service("choices", func() {
			Method("echo", func() {
				Payload(choice(), occurrence)
				Result(choice(), occurrence)
			})
		})
	})
	method := root.Service("choices").Method("echo")
	for _, attribute := range []*expr.AttributeExpr{
		root.UserType("Choice").Attribute(),
		root.UserType("Envelope").Attribute().Find("value"),
		method.Payload, method.Result,
	} {
		require.True(t, expr.IsNullable(attribute))
		require.Equal(t, "choice occurrence", attribute.Description)
		require.Equal(t, "Choice", attribute.Title)
		require.Equal(t, []string{"true"}, attribute.Meta["openapi:deprecated"])
		require.Len(t, attribute.UserExamples, 1)
	}
	t.Run("union defaults remain rejected", func(t *testing.T) {
		err := expr.RunInvalidDSL(t, func() {
			Type("Choice", choice(), func() {
				Default("chosen")
			})
		})
		require.ErrorContains(t, err, "default values are not supported for union attributes")
	})
}
