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

func TestFunctionUnionKeepsPendingBranches(t *testing.T) {
	for _, tagged := range []bool{true, false} {
		root := expr.RunDSL(t, func() {
			event := Type("Event", OneOf(func() {
				if !tagged {
					Untagged()
				}
				Attribute("created", "EventData", func() {
					Meta("openapi:discriminator:mapping", "created", "alias")
					Required("id")
				})
				Attribute("deleted", String)
			}))
			Type("EventData", func() {
				Attribute("id", String)
				Attribute("child", event)
			})
			Service("events", func() {
				Method("Create", func() {
					Payload(func() {
						Attribute("body", event)
					})
					Result(event)
					HTTP(func() {
						POST("/events")
						Body("body", func() {
							Meta("openapi:typename", "Event")
						})
						Response(StatusOK, func() {
							OpenAPIBody(event)
						})
					})
				})
			})
		})
		branch := expr.AsUnion(root.UserType("Event")).Values[0].Attribute
		require.Equal(t, "EventData", branch.Type.Name())
		require.Equal(t, []string{"created", "alias"}, branch.Meta["openapi:discriminator:mapping"])
		require.True(t, branch.IsRequired("id"))
		require.NotNil(t, expr.AsObject(branch.Type).Attribute("child"))
	}
}

func TestFunctionUnionReportsMissingBranch(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		Type("Choice", OneOf(func() {
			Attribute("missing", "Missing", func() {
				Description("retained branch annotation")
			})
		}))
	})
	require.ErrorContains(t, err, `unknown type reference "Missing"`)
	require.NotContains(t, err.Error(), "panic")
}

func TestFunctionUnionRefinementOwnership(t *testing.T) {
	for _, hints := range []bool{false, true} {
		calls := 0
		root := expr.RunDSL(t, func() {
			constructor := OneOf(func() {
				Attribute("record", "Record", func() {
					calls++
					Required("id")
					if hints {
						Meta("openapi:discriminator:mapping", "record", "alias")
					}
				})
				Attribute("text", String)
			})
			first := Type("First", constructor)
			second := Type("Second", constructor)
			Type("Record", func() {
				Attribute("id", String)
				Attribute("first", first)
				Attribute("second", second)
			})
			Service("records", func() {
				Method("Create", func() {
					Payload(first, func() {
						Description("local payload occurrence")
					})
					Result(second)
					HTTP(func() {
						POST("/records")
						Response(StatusOK)
					})
				})
			})
		})
		require.Equal(t, 2, calls, "each copied pending occurrence refines exactly once")
		first := expr.AsUnion(root.UserType("First")).Values[0].Attribute
		second := expr.AsUnion(root.UserType("Second")).Values[0].Attribute
		for _, branch := range []*expr.AttributeExpr{first, second} {
			require.NoError(t, branch.ResolveTypeReference())
			require.True(t, branch.IsRequired("id"))
			require.NotNil(t, branch.Find("first"))
			require.NotNil(t, branch.Find("second"))
		}
		require.Equal(t, 2, calls, "repeated completion cannot replay a refinement")
		require.False(t, root.UserType("Record").Attribute().IsRequired("id"))
		first.Validation.Required = append(first.Validation.Required, "first")
		require.False(t, second.IsRequired("first"), "copied occurrence constraints stay detached")
		if hints {
			first.Meta["openapi:discriminator:mapping"][0] = "changed"
			require.Equal(t, "record", second.Meta["openapi:discriminator:mapping"][0])
		}
	}
}
