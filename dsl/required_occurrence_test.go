package dsl_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestRequiredAliasesPreserveDeclaration(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		t.Run(map[bool]string{false: "forward", true: "reverse"}[reverse], func(t *testing.T) {
			root := expr.RunDSL(t, func() {
				base := Type("Base", func() {
					Attribute("id", String)
					Attribute("left", String)
					Attribute("right", String)
					Required("id")
				})
				names := []string{"left", "right"}
				if reverse {
					names[0], names[1] = names[1], names[0]
				}
				for _, name := range names {
					Type(name, base, func() {
						Required(name)
					})
				}
			})
			base := root.UserType("Base").Attribute()
			require.Equal(t, []string{"id"}, base.AllRequired())
			for _, name := range []string{"left", "right"} {
				alias := root.UserType(name).Attribute()
				require.ElementsMatch(t, []string{"id", name}, alias.AllRequired())
				facts := expr.NewObjectFields(alias).FieldFacts(name)
				require.True(t, facts.Required)
				require.False(t, facts.LayoutRequired, "a refinement cannot change the referenced declaration layout")
			}
		})
	}
}

func TestRequiredAdditionalAliasPreservesLiteralKeys(t *testing.T) {
	root := expr.RunDSL(t, func() {
		base := Type("Extras", func() {
			AdditionalAttributes(Int)
			Required("a:b")
		})
		Type("RefinedExtras", base, func() {
			Required("a,b")
		})
	})
	require.Equal(t, []string{"a:b"}, root.UserType("Extras").Attribute().AllRequired())
	require.ElementsMatch(t, []string{"a:b", "a,b"}, root.UserType("RefinedExtras").Attribute().AllRequired())
}
