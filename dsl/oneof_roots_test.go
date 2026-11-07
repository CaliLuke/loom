package dsl_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestOneOfBranchDeclarationConstructor(t *testing.T) {
	choice := func() expr.DataType {
		return OneOf(func() {
			Untagged()
			Attribute("Text", String, func() {
				Pattern("^[a-z]+$")
			})
			Attribute("Items", ArrayOf(Int))
		})
	}
	root := expr.RunDSL(t, func() {
		Type("Choice", choice())
		Type("Envelope", func() {
			Attribute("value", choice())
		})
		Service("choices", func() {
			Method("echo", func() {
				Payload(choice())
				Result(choice())
			})
		})
	})
	method := root.Service("choices").Method("echo")
	for _, attribute := range []*expr.AttributeExpr{
		root.UserType("Choice").Attribute(),
		root.UserType("Envelope").Attribute().Find("value"),
		method.Payload, method.Result,
	} {
		union := expr.AsUnion(attribute.Type)
		require.NotNil(t, union)
		require.True(t, union.Untagged)
		require.Len(t, union.Values, 2)
		require.Equal(t, "^[a-z]+$", union.Values[0].Attribute.Validation.Pattern)
		require.NotEmpty(t, union.TypeName)
	}
}

func TestUntaggedAcceptsRootKinds(t *testing.T) {
	root := expr.RunDSL(t, func() {
		alias := Type("Code", String, func() {
			Pattern("^[a-z]+$")
		})
		typed := Type("Record", func() {
			Attribute("name", String)
			AdditionalAttributes(Int)
		})
		open := Type("Open", func() {
			Attribute("name", String)
			Meta("openapi:additionalProperties", "true")
		})
		nested := OneOf(String, Int)
		expr.AsUnion(nested).Untagged = true
		for index, typ := range []expr.DataType{Boolean, String, Bytes, Int, Int32, UInt64, Float64, Any, alias, MapOf(String, Int), ArrayOf(Int), nested, typed, open} {
			rootType := Type(fmt.Sprintf("Root%d", index), OneOf(func() {
				Untagged()
				Attribute("First", typ)
				Attribute("Second", Boolean)
			}))
			Service(fmt.Sprintf("root%d", index), func() {
				Method("show", func() {
					Result(rootType)
				})
			})
		}
		nullable := Type("NullableChoice", OneOf(func() {
			Untagged()
			Attribute("Text", String, Nullable)
			Attribute("Integer", Int)
		}))
		Service("nullable", func() {
			Method("show", func() {
				Result(nullable)
			})
		})
	})
	require.NotNil(t, root.UserType("NullableChoice"))
}

func TestUntaggedRejectsTaggedBranch(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		outer := Type("Outer", OneOf(func() {
			Untagged()
			Attribute("Tagged", OneOf(String, Int))
			Attribute("Text", String)
		}))
		Service("outer", func() {
			Method("show", func() {
				Result(outer)
			})
		})
	})
	require.ErrorContains(t, err, "tagged union")
}
