package dsl_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestUntaggedNestedComposition(t *testing.T) {
	root := expr.RunDSL(t, func() {
		person := Type("Person", func() {
			Attribute("email", String)
			Required("email")
		})
		group := Type("Group", func() {
			Attribute("label", String)
			Required("label")
		})
		entry := Type("Entry", OneOf(person, group), Untagged)
		page := Type("Page", func() {
			Attribute("results", ArrayOf(entry))
			Attribute("lookup", MapOf(String, entry))
			Attribute("single", entry)
			Required("results")
		})
		Service("directory", func() {
			Method("read", func() {
				Result(OneOf(person, page), Untagged)
			})
		})
	})
	require.NotNil(t, root.Service("directory"))
}

func TestUntaggedRecursiveNestedComposition(t *testing.T) {
	root := expr.RunDSL(t, func() {
		node := Type("Node", func() {
			Attribute("children", ArrayOf("Tree"))
			Required("children")
		})
		leaf := Type("Leaf", func() {
			Attribute("name", String)
			Required("name")
		})
		tree := Type("Tree", OneOf(node, leaf), Untagged)
		Service("tree", func() {
			Method("read", func() {
				Result(tree)
			})
		})
	})
	require.NotNil(t, root.Service("tree"))
}

func TestUntaggedNestedRejectsUnsupportedBranches(t *testing.T) {
	for _, tagged := range []bool{false, true} {
		err := expr.RunInvalidDSL(t, func() {
			person := Type("Person", func() {
				Attribute("name", String)
				Required("name")
			})
			choice := Type("Choice", OneOf(person, String), func() {
				if !tagged {
					Untagged()
				}
			})
			page := Type("Page", func() {
				Attribute("rows", ArrayOf(choice))
			})
			Service("unsupported", func() {
				Method("read", func() {
					Result(OneOf(person, page), Untagged)
				})
			})
		})
		if tagged {
			require.ErrorContains(t, err, "array elements must be primitive")
		} else {
			require.ErrorContains(t, err, "must be a concrete named object or array type")
		}
	}
}
