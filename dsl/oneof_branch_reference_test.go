package dsl_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// TestOneOfInlineBranchKeepsUserTypeReferences verifies that promoting an
// inline union branch object to a user type keeps references to the declared
// user types instead of snapshotting them before their DSL has run.
func TestOneOfInlineBranchKeepsUserTypeReferences(t *testing.T) {
	cases := []struct {
		Name       string
		Referenced string
		DSL        func()
	}{
		{"recursive-cycle-through-inline-object", "CycleNode", func() {
			Type("CycleNode", func() {
				Field(1, "holder", func() {
					OneOf("choice", func() {
						Field(1, "br", func() {
							Field(1, "branches", ArrayOf("CycleNode"))
						})
					})
				})
			})
		}},
		{"later-declared-type", "Later", func() {
			Type("CycleNode", func() {
				Field(1, "holder", func() {
					OneOf("choice", func() {
						Field(1, "br", func() {
							Field(1, "branches", ArrayOf("Later"))
						})
					})
				})
			})
			Type("Later", func() {
				Field(1, "name", String)
			})
		}},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := expr.RunDSL(t, c.DSL)
			referenced := root.UserType(c.Referenced)
			require.NotNil(t, referenced)

			holder := root.UserType("CycleNode").Attribute().Find("holder")
			require.NotNil(t, holder)
			choice := holder.Find("choice")
			require.NotNil(t, choice)
			union := expr.AsUnion(choice.Type)
			require.NotNil(t, union)
			require.Len(t, union.Values, 1)

			branch, ok := union.Values[0].Attribute.Type.(expr.UserType)
			require.True(t, ok, "inline branch must be promoted to a user type")
			require.Equal(t, "choiceBr", branch.Name())
			branches := branch.Attribute().Find("branches")
			require.NotNil(t, branches)
			elem := expr.AsArray(branches.Type).ElemType.Type
			require.Same(t, referenced, elem)
			require.NotEmpty(t, *expr.AsObject(elem))
		})
	}
}
