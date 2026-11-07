package dsl_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestUnionClosureOccurrenceIdentity(t *testing.T) {
	for _, reversed := range []bool{false, true} {
		root := expr.RunDSL(t, func() {
			record := Type("Record", func() {
				Attribute("x", Int)
				Attribute("next", "Choice")
				Required("x")
			})
			choice := Type("Choice", OneOf(record, String), Untagged)
			Type("ChoiceClosed", String)
			Type("ChoiceClosed2", String)
			closed := Type("DeclaredClosed", OneOf(record, String), func() {
				Untagged()
				Meta("openapi:unevaluatedProperties", "false")
			})
			Service("closure", func() {
				Method("echo", func() {
					Payload(func() {
						fields := []string{"open", "closed", "closedAgain"}
						if reversed {
							fields = []string{"closedAgain", "closed", "open"}
						}
						for _, field := range fields {
							Attribute(field, choice, func() {
								if field != "open" {
									Meta("openapi:unevaluatedProperties", "false")
								}
							})
						}
						Attribute("inherited", closed)
					})
				})
			})
		})
		payload := root.Service("closure").Method("echo").Payload
		open := payload.Find("open").Type.(expr.UserType)
		closed := payload.Find("closed").Type.(expr.UserType)
		require.NotEqual(t, open.ID(), closed.ID())
		require.NotEqual(t, open.Hash(), closed.Hash())
		require.Equal(t, "ChoiceClosed3", closed.Name())
		require.Equal(t, closed.ID(), payload.Find("closedAgain").Type.(expr.UserType).ID())
		require.Equal(t, open.Hash(), root.UserType("Choice").Hash())
		require.Equal(t, open.Hash(), root.UserType("Record").Attribute().Find("next").Type.Hash())
		require.False(t, root.UserType("Choice").Attribute().UnionClosed())
		require.False(t, open.Attribute().UnionClosed())
		require.True(t, closed.Attribute().UnionClosed())
		require.Equal(t, "DeclaredClosed", payload.Find("inherited").Type.Name())
		require.True(t, payload.Find("inherited").UnionClosed())
		copied := expr.DupAtt(payload)
		require.NotEqual(t, copied.Find("open").Type.Hash(), copied.Find("closed").Type.Hash())
		require.True(t, copied.Find("closed").UnionClosed())
	}
}

func TestUnionConstructorRetainsClosure(t *testing.T) {
	root := expr.RunDSL(t, func() {
		choice := Type("Choice", OneOf(func() {
			Untagged()
			Attribute("Any", Any)
			Attribute("Text", String)
			Meta("openapi:unevaluatedProperties", "false")
		}))
		Service("constructor", func() {
			Method("echo", func() {
				Payload(choice)
			})
		})
	})
	choice := root.UserType("Choice")
	require.True(t, choice.Attribute().UnionClosed())
	require.True(t, expr.AsUnion(choice).Closed())
	require.True(t, expr.AsUnion(expr.Dup(choice)).Closed())
	err := expr.RunInvalidDSL(t, func() {
		Type("Bad", OneOf(func() {
			Untagged()
			Attribute("Any", Any)
			Attribute("Text", String)
			Meta("openapi:unevaluatedProperties", "true")
		}))
	})
	require.ErrorContains(t, err, "supports only false")
}

func TestUnionClosurePreservesLocalizedKeys(t *testing.T) {
	root := expr.RunDSL(t, func() {
		shared := Type("Choice", OneOf(String, Int))
		holder := Type("Holder", func() {
			for _, name := range []string{"a", "aAgain", "b"} {
				Attribute(name, shared, func() {
					key := name
					if key == "aAgain" {
						key = "a"
					}
					Meta("oneof:type:field", key+"Kind")
					Meta("oneof:value:field", key+"Value")
					Meta("openapi:unevaluatedProperties", "false")
				})
			}
		})
		Service("keys", func() {
			Method("echo", func() {
				Payload(holder)
			})
		})
	})
	holder := root.UserType("Holder").Attribute()
	first, second := holder.Find("a").Type.(expr.UserType), holder.Find("b").Type.(expr.UserType)
	require.Equal(t, "aKind", expr.AsUnion(first).GetTypeKey())
	require.Equal(t, "aValue", expr.AsUnion(first).GetValueKey())
	require.Equal(t, "bKind", expr.AsUnion(second).GetTypeKey())
	require.Equal(t, "bValue", expr.AsUnion(second).GetValueKey())
	require.NotEqual(t, first.ID(), second.ID())
	require.Equal(t, first.ID(), holder.Find("aAgain").Type.(expr.UserType).ID())
}
