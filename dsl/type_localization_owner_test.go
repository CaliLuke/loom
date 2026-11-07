package dsl_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestLocalizedStructuralBackReference(t *testing.T) {
	for _, nested := range []bool{false, true} {
		t.Run(map[bool]string{false: "direct", true: "nested"}[nested], func(t *testing.T) {
			var before, declaration expr.UserType
			root := expr.RunDSL(t, func() {
				y := Type("OwnerY", func() {
					Attribute("v", String)
					Attribute("x", "OwnerX")
				})
				declaration = Type("OwnerX", func() {
					Attribute("name", String)
					fields := func() {
						Attribute("left", y, func() { Required("v") })
						Attribute("right", y, func() { Description("independent") })
					}
					if nested {
						Attribute("holder", fields)
					} else {
						fields()
					}
				})
				Service("OwnerService", func() {
					Method("Show", func() {
						owner := declaration.Attribute()
						if nested {
							owner = owner.Find("holder")
						}
						before = owner.Find("left").Type.(expr.UserType).Attribute().Find("x").Type.(expr.UserType)
						Payload(declaration, func() { Required("name") })
					})
				})
			})
			require.Same(t, declaration, before, "bind structural back-reference before Payload invokes generic Dup")
			payload := root.Service("OwnerService").Method("Show").Payload.Type.(expr.UserType)
			owner := payload.Attribute()
			if nested {
				owner = owner.Find("holder")
			}
			left := owner.Find("left").Type.(expr.UserType)
			right := owner.Find("right").Type.(expr.UserType)
			require.NotSame(t, left, right, "independent localized occurrences remain separate")
			require.Same(t, payload, left.Attribute().Find("x").Type)
			require.Same(t, payload, right.Attribute().Find("x").Type)
			require.True(t, owner.Find("left").IsRequired("v"))
			require.False(t, owner.Find("right").IsRequired("v"))
			require.False(t, root.UserType("OwnerY").Attribute().IsRequired("v"))
		})
	}
}

func TestPayloadOwnsDirectSelfReference(t *testing.T) {
	root := expr.RunDSL(t, func() {
		node := Type("DirectOwner", func() {
			Attribute("name", String)
			Attribute("next", "DirectOwner")
		})
		Service("DirectService", func() {
			Method("Show", func() {
				Payload(node, func() { Required("name") })
			})
		})
	})
	payload := root.Service("DirectService").Method("Show").Payload.Type.(expr.UserType)
	require.Same(t, payload, payload.Attribute().Find("next").Type)
	require.NotSame(t, root.UserType("DirectOwner"), payload)
	require.False(t, root.UserType("DirectOwner").Attribute().IsRequired("name"))
}
