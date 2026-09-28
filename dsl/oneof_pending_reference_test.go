package dsl_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestNamedUnionKeepsPendingBranches(t *testing.T) {
	for _, test := range []struct {
		name   string
		design func()
		owner  string
		branch string
	}{
		{"forward", func() {
			inner := Type("Inner", OneOf("Later", Int))
			Type("Outer", OneOf(inner, String))
			Type("Later", func() {
				Field(1, "value", String)
			})
		}, "Inner", "Later"},
		{"constructor in block", func() {
			Type("Holder", func() {
				OneOf("choice", func() {
					Attribute("nested", OneOf("Later", Int))
				})
			})
			Type("Later", func() {
				Field(1, "value", String)
			})
		}, "Holder", "Later"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := expr.RunDSL(t, test.design)
			owner := root.UserType(test.owner).Attribute()
			if test.owner == "Holder" {
				owner = expr.AsUnion(owner.Find("choice").Type).Values[0].Attribute
			}
			union := expr.AsUnion(owner.Type)
			require.NotNil(t, union)
			var referenced expr.DataType
			for _, branch := range union.Values {
				if branch.Attribute.Type.Name() == test.branch {
					referenced = branch.Attribute.Type
				}
			}
			require.Same(t, root.UserType(test.branch), referenced)
		})
	}
}

func TestNamedUnionReportsMissingBranch(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		Type("Choice", OneOf("Missing", String))
	})
	require.ErrorContains(t, err, `unknown type reference "Missing"`)
	require.NotContains(t, err.Error(), "panic")
}

func TestNamedUnionOwnsBranchMetadata(t *testing.T) {
	var base *expr.Union
	root := expr.RunDSL(t, func() {
		base = expr.AsUnion(OneOf(String, Int))
		branch := base.Values[0].Attribute
		branch.Meta["description"] = []string{"original"}
		branch.Validation = &expr.ValidationExpr{Pattern: "^value$"}
		branch.Docs = &expr.DocsExpr{URL: "https://example.com/original"}
		Type("First", base)
		Type("Second", base)
	})
	first := expr.AsUnion(root.UserType("First")).Values[0].Attribute
	second := expr.AsUnion(root.UserType("Second")).Values[0].Attribute
	first.Meta["description"][0] = "changed"
	first.Validation.Pattern = "^changed$"
	first.Docs.URL = "https://example.com/changed"
	for _, branch := range []*expr.AttributeExpr{base.Values[0].Attribute, second} {
		require.Equal(t, []string{"original"}, branch.Meta["description"])
		require.Equal(t, "^value$", branch.Validation.Pattern)
		require.Equal(t, "https://example.com/original", branch.Docs.URL)
	}
}

func TestNamedUnionRejectsInlineBranchCycles(t *testing.T) {
	for _, test := range []struct {
		name   string
		design func()
	}{
		{"self", func() {
			leaf := Type("Leaf", func() {
				Field(1, "value", String)
			})
			Type("Tree", OneOf(leaf, "Tree"))
		}},
		{"mutual", func() {
			first := Type("First", OneOf("Second", String))
			Type("Second", OneOf(first, Int))
		}},
		{"through alias", func() {
			first := Type("First", OneOf("Alias", String))
			Type("Alias", first)
		}},
		{"through array", func() {
			tree := Type("Tree", OneOf("Forest", String))
			Type("Forest", ArrayOf(tree))
		}},
		{"through map", func() {
			tree := Type("Tree", OneOf("Forest", String))
			Type("Forest", MapOf(String, tree))
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, test.design)
			require.ErrorContains(t, err, "recursive OneOf branch cycle")
			require.NotContains(t, err.Error(), "panic")
		})
	}
}
