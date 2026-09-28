package dsl_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestRepeatedBlockNamesKeepDistinctBranchDefinitions(t *testing.T) {
	root := expr.RunDSL(t, func() {
		b := Type("B", func() {
			Field(1, "value", String)
		})
		c := Type("C", func() {
			Field(1, "count", Int)
		})
		d := Type("D", func() {
			Field(1, "flag", Boolean)
		})
		first := Type("First", func() {
			OneOf("r", func() {
				Field(3, "s", OneOf(b, c))
			})
		})
		second := Type("Second", func() {
			OneOf("r", func() {
				Field(3, "s", OneOf(b, d))
			})
		})
		Type("Envelope", func() {
			Field(1, "first", first)
			Field(2, "second", second)
		})
	})
	first := expr.AsUnion(root.UserType("First").Attribute().Find("r").Type).Values[0].Attribute.Type
	second := expr.AsUnion(root.UserType("Second").Attribute().Find("r").Type).Values[0].Attribute.Type
	t.Run("reference identity", func(t *testing.T) {
		require.NotEqual(t, first.Hash(), second.Hash(), "different generated branch definitions must have different reference identities")
	})
	copy := expr.DupAtt(root.UserType("Envelope").Attribute())
	for _, test := range []struct {
		field  string
		branch string
	}{
		{"first", "C"},
		{"second", "D"},
	} {
		t.Run(test.field, func(t *testing.T) {
			object := copy.Find(test.field).Type.(expr.UserType).Attribute()
			promoted := expr.AsUnion(object.Find("r").Type).Values[0].Attribute.Type
			branches := expr.AsUnion(promoted).Values
			names := make([]string, 0, len(branches))
			for _, branch := range branches {
				names = append(names, branch.Name)
			}
			require.ElementsMatch(t, []string{"B", test.branch}, names)
		})
	}
}

func TestPromotedBranchDoesNotReplaceAuthoredType(t *testing.T) {
	for _, name := range []string{"$union-branch:1", "rS"} {
		t.Run(name, func(t *testing.T) {
			root := expr.RunDSL(t, func() {
				var authored expr.UserType
				holder := Type("Holder", func() {
					OneOf("r", func() {
						Attribute("s", String)
					})
					Attribute("authored", authored)
				})
				Type("Envelope", func() {
					Attribute("holder", holder, func() {
						Description("localized copy")
					})
				})
				// Evaluate this declaration after Envelope so localization must
				// visit it even when a promoted branch has the same name.
				authored = Type(name, func() {
					Attribute("value", String)
				})
			})
			holder := root.UserType("Envelope").Attribute().Find("holder").Type.(expr.UserType)
			copied := holder.Attribute().Find("authored").Type
			require.Equal(t, name, copied.Name())
			require.True(t, expr.IsObject(copied))
			require.Equal(t, expr.String, expr.AsObject(copied).Attribute("value").Type)
			branch := expr.AsUnion(holder.Attribute().Find("r").Type).Values[0].Attribute.Type
			require.Equal(t, expr.String, branch.(expr.UserType).Attribute().Type)
			require.NotEqual(t, copied.(expr.UserType).ID(), branch.(expr.UserType).ID())
		})
	}
}
