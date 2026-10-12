package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestExtendCompositionRecursiveBase(t *testing.T) {
	for _, nodeFirst := range []bool{false, true} {
		t.Run(map[bool]string{false: "base first", true: "node first"}[nodeFirst], func(t *testing.T) {
			var base, node expr.UserType
			expr.RunDSL(t, func() {
				baseDSL := func() {
					Attribute("next", node)
					Attribute("count", Int, func() {
						Minimum(0)
					})
				}
				nodeDSL := func() {
					Extend(base)
					Attribute("n", Int)
					Attribute("count", Int, func() {
						Maximum(100)
					})
				}
				if nodeFirst {
					node = Type("Node", nodeDSL)
					base = Type("Base", baseDSL)
				} else {
					base = Type("Base", baseDSL)
					node = Type("Node", nodeDSL)
				}
			})
			for _, attribute := range []*expr.AttributeExpr{base.Attribute(), node.Attribute()} {
				next := attribute.Find("next")
				require.NotNil(t, next)
				require.NotNil(t, expr.AsObject(next.Type).Attribute("n"))
				require.NotNil(t, expr.AsObject(next.Type).Attribute("next"))
				count := expr.AsObject(next.Type).Attribute("count")
				constraints, err := expr.EffectiveConstraintsFor(count)
				require.NoError(t, err)
				require.Equal(t, *boundForTest(0.0), constraints.LowerBound().Value)
				require.Equal(t, *boundForTest(100.0), constraints.UpperBound().Value)
			}
			require.Same(t, node, node.Attribute().Find("next").Type,
				"a structural self-reference must bind to its owning declaration")
			// Inherited recursive fields and retained constraints own their data.
			require.NotSame(t, base.Attribute().Find("next"), node.Attribute().Find("next"))
			*base.Attribute().Find("count").Validation.Minimum = *boundForTest(7)
			nestedCount := expr.AsObject(node.Attribute().Find("next").Type).Attribute("count")
			constraints, err := expr.EffectiveConstraintsFor(nestedCount)
			require.NoError(t, err)
			require.Equal(t, *boundForTest(0.0), constraints.LowerBound().Value)
		})
	}
}

func TestExtendCompositionPeerJSONTags(t *testing.T) {
	for _, test := range []struct {
		name, full, nameOnly string
		valid                bool
	}{
		{"implicit omission differs", "one", "one", false},
		{"different name", "one,omitempty", "two", false},
		{"equivalent omission", "one,omitempty", "one", true},
	} {
		for _, reversed := range []bool{false, true} {
			t.Run(test.name+map[bool]string{false: "/forward", true: "/reverse"}[reversed], func(t *testing.T) {
				design := func() {
					a := Type("A", func() {
						Attribute("n", String, func() {
							Meta("struct:tag:json", test.full)
						})
					})
					b := Type("B", func() {
						Attribute("n", String, func() {
							Meta("struct:tag:json:name", test.nameOnly)
						})
					})
					Type("Derived", func() {
						if reversed {
							Extend(b)
							Extend(a)
						} else {
							Extend(a)
							Extend(b)
						}
					})
				}
				if test.valid {
					expr.RunDSL(t, design)
				} else {
					err := expr.RunInvalidDSL(t, design)
					require.ErrorContains(t, err, "inherited JSON tag")
				}
			})
		}
	}
}

func TestExtendCompositionMutualRecursion(t *testing.T) {
	for _, nodeFirst := range []bool{false, true} {
		t.Run(map[bool]string{false: "base first", true: "node first"}[nodeFirst], func(t *testing.T) {
			var base, node, other expr.UserType
			expr.RunDSL(t, func() {
				baseDSL := func() {
					Attribute("next", other)
				}
				nodeDSL := func() {
					Extend(base)
					Attribute("n", Int)
				}
				if nodeFirst {
					node = Type("Node", nodeDSL)
					base = Type("Base", baseDSL)
				} else {
					base = Type("Base", baseDSL)
					node = Type("Node", nodeDSL)
				}
				other = Type("Other", func() {
					Attribute("back", node)
				})
			})
			next := expr.AsObject(node.Attribute().Find("next").Type)
			require.NotNil(t, next)
			require.Same(t, node, next.Attribute("back").Type)
			require.NotSame(t, base.Attribute().Find("next"), node.Attribute().Find("next"))
		})
	}
}
