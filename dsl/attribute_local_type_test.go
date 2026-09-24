package dsl_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// TestAttributeDSLKeepsLaterDeclaredUserTypes verifies that an attribute DSL
// refines a complete local copy of a referenced user type whose DSL has not
// run yet rather than an empty snapshot.
func TestAttributeDSLKeepsLaterDeclaredUserTypes(t *testing.T) {
	cases := []struct {
		Name  string
		DSL   func()
		Field func(*testing.T, *expr.AttributeExpr) *expr.AttributeExpr
	}{
		{"direct-by-name", func() {
			Type("Holder", func() {
				Attribute("field", "Later", func() {
					Description("described locally")
				})
			})
			laterType()
		}, directField},
		{"direct-by-variable", func() {
			var later expr.UserType
			Type("Holder", func() {
				Attribute("field", later, func() {
					Description("described locally")
				})
			})
			later = laterType()
		}, directField},
		{"array", func() {
			Type("Holder", func() {
				Attribute("field", ArrayOf("Later"), func() {
					MinLength(1)
				})
			})
			laterType()
		}, arrayElem},
		{"array-constructor-dsl", func() {
			Type("Holder", func() {
				Attribute("field", ArrayOf("Later", func() {
					Description("element")
				}))
			})
			laterType()
		}, arrayElem},
		{"map", func() {
			Type("Holder", func() {
				Attribute("field", MapOf(String, "Later"), func() {
					MinLength(1)
				})
			})
			laterType()
		}, mapElem},
		{"map-constructor-dsl", func() {
			Type("Holder", func() {
				Attribute("field", MapOf(String, "Later", func() {
					Key(func() {
						MinLength(1)
					})
				}))
			})
			laterType()
		}, mapElem},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := expr.RunDSL(t, c.DSL)
			later := root.UserType("Later")
			require.NotNil(t, later)
			field := root.UserType("Holder").Attribute().Find("field")
			require.NotNil(t, field)

			local, ok := c.Field(t, field).Type.(expr.UserType)
			require.True(t, ok)
			require.Equal(t, "Later", local.Name())
			require.NotNil(t, expr.AsObject(local).Attribute("name"))
			require.NotNil(t, expr.AsObject(local).Attribute("count"))
		})
	}
}

// TestRequiredInAttributeDSLRefinesLaterDeclaredUserType verifies that an
// attribute-level Required on a later-declared type yields a complete local
// copy that requires the field and leaves the shared type unchanged.
func TestRequiredInAttributeDSLRefinesLaterDeclaredUserType(t *testing.T) {
	cases := []struct {
		Name  string
		DSL   func()
		Field func(*testing.T, *expr.AttributeExpr) *expr.AttributeExpr
	}{
		{"direct", func() {
			Type("Holder", func() {
				Attribute("field", "Later", func() {
					Required("name")
				})
				Attribute("plain", "Later")
			})
			laterType()
		}, directField},
		{"array-elem", func() {
			Type("Holder", func() {
				Attribute("field", ArrayOf("Later"), func() {
					Elem(func() {
						Required("name")
					})
				})
				Attribute("plain", "Later")
			})
			laterType()
		}, arrayElem},
		{"map-elem", func() {
			Type("Holder", func() {
				Attribute("field", MapOf(String, "Later"), func() {
					Elem(func() {
						Required("name")
					})
				})
				Attribute("plain", "Later")
			})
			laterType()
		}, mapElem},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := expr.RunDSL(t, c.DSL)
			later := root.UserType("Later")
			require.NotNil(t, later)
			require.False(t, later.Attribute().IsRequired("name"), "shared type must not change")
			holder := root.UserType("Holder").Attribute()
			require.Same(t, later, holder.Find("plain").Type)

			refined := c.Field(t, holder.Find("field"))
			local, ok := refined.Type.(expr.UserType)
			require.True(t, ok)
			require.NotSame(t, later, local)
			require.Equal(t, "Later", local.Name())
			require.True(t, local.Attribute().IsRequired("name"))
			require.NotNil(t, expr.AsObject(local).Attribute("name"))
			require.NotNil(t, expr.AsObject(local).Attribute("count"))
		})
	}
}

// TestRequiredRefinementsNestInLocalCopies verifies that a local copy of a
// user type keeps the local refinements declared inside that type, whichever
// order the DSL declares the types in.
func TestRequiredRefinementsNestInLocalCopies(t *testing.T) {
	trip := func() {
		Type("Trip", func() {
			Attribute("driver", "Person", func() {
				Required("car")
			})
		})
	}
	person := func() {
		Type("Person", func() {
			Attribute("car", "Car", func() {
				Required("plate")
			})
		})
	}
	car := func() {
		Type("Car", func() {
			Attribute("plate", String)
		})
	}
	cases := []struct {
		Name string
		DSL  func()
	}{
		{"outer-first", func() {
			trip()
			person()
			car()
		}},
		{"inner-first", func() {
			car()
			person()
			trip()
		}},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := expr.RunDSL(t, c.DSL)
			require.False(t, root.UserType("Person").Attribute().IsRequired("car"))
			require.False(t, root.UserType("Car").Attribute().IsRequired("plate"))

			driver := root.UserType("Trip").Attribute().Find("driver")
			require.NotNil(t, driver)
			require.True(t, driver.IsRequired("car"))
			car := driver.Find("car")
			require.NotNil(t, car)
			require.True(t, car.IsRequired("plate"))
			require.NotNil(t, car.Find("plate"))
		})
	}
}

// TestAttributeExtendAndReferenceDoNotChangeSharedType verifies that Extend
// and Reference in an attribute DSL apply to a local copy of the attribute
// type and leave the shared type and its other references unchanged.
func TestAttributeExtendAndReferenceDoNotChangeSharedType(t *testing.T) {
	cases := []struct {
		Name   string
		Refine func(base expr.UserType)
		Check  func(t *testing.T, local, shared *expr.AttributeExpr)
	}{
		{"extend", func(base expr.UserType) {
			Extend(base)
		}, func(t *testing.T, local, shared *expr.AttributeExpr) {
			require.NotNil(t, local.Find("extra"))
			require.Nil(t, shared.Find("extra"))
		}},
		{"reference", func(base expr.UserType) {
			Reference(base)
		}, func(t *testing.T, local, shared *expr.AttributeExpr) {
			require.Equal(t, "base name", local.Find("name").Description)
			require.Empty(t, shared.Find("name").Description)
		}},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := expr.RunDSL(t, func() {
				base := Type("RefineBase", func() {
					Attribute("name", String, "base name")
					Attribute("extra", String)
				})
				shared := Type("RefineShared", func() {
					Attribute("name", String)
				})
				Type("RefineHolder", func() {
					Attribute("a", shared, func() {
						c.Refine(base)
					})
					Attribute("b", shared)
				})
			})

			shared := root.UserType("RefineShared")
			holder := root.UserType("RefineHolder").Attribute()
			require.Same(t, shared, holder.Find("b").Type)
			local, ok := holder.Find("a").Type.(expr.UserType)
			require.True(t, ok)
			require.NotSame(t, shared, local)
			c.Check(t, local.Attribute(), shared.Attribute())
		})
	}
}

// TestLocalRequirementSurvivesAttributeCopy verifies that an attribute copied
// during DSL execution keeps the local refinement of the original.
func TestLocalRequirementSurvivesAttributeCopy(t *testing.T) {
	root := expr.RunDSL(t, func() {
		shared := Type("CopyShared", func() {
			Attribute("name", String)
		})
		first := Type("CopyFirst", func() {
			Attribute("owner", shared, func() {
				Required("name")
			})
		})
		Type("CopySecond", func() {
			Reference(first)
			Attribute("owner")
		})
	})

	shared := root.UserType("CopyShared")
	require.False(t, shared.Attribute().IsRequired("name"))
	for _, name := range []string{"CopyFirst", "CopySecond"} {
		owner := root.UserType(name).Attribute().Find("owner")
		require.NotNil(t, owner, name)
		require.NotSame(t, shared, owner.Type, name)
		require.True(t, owner.IsRequired("name"), name)
	}
}

// TestLocalRequirementSurvivesStructureCopy verifies that a refinement
// recorded on a collection element keeps applying to the element's local type
// when an enclosing attribute DSL, a redeclaration, or union metadata copies
// the structure that holds it.
func TestLocalRequirementSurvivesStructureCopy(t *testing.T) {
	item := func() expr.UserType {
		return Type("StructItem", func() {
			Attribute("name", String)
			Attribute("code", String)
		})
	}
	cases := []struct {
		Name    string
		DSL     func()
		Element func(*expr.RootExpr) *expr.AttributeExpr
	}{
		{"array-constructor-in-attribute-dsl", func() {
			it := item()
			Type("StructHolder", func() {
				Attribute("list", ArrayOf(it, func() {
					Required("name")
				}), func() {
					MinLength(1)
				})
			})
		}, func(root *expr.RootExpr) *expr.AttributeExpr {
			return expr.AsArray(root.UserType("StructHolder").Attribute().Find("list").Type).ElemType
		}},
		{"map-constructor-in-attribute-dsl", func() {
			it := item()
			Type("StructHolder", func() {
				Attribute("index", MapOf(String, it, func() {
					Elem(func() {
						Required("name")
					})
				}), func() {
					MinLength(1)
				})
			})
		}, func(root *expr.RootExpr) *expr.AttributeExpr {
			return expr.AsMap(root.UserType("StructHolder").Attribute().Find("index").Type).ElemType
		}},
		{"reference-and-redeclaration", func() {
			it := item()
			holder := Type("StructHolder", func() {
				Attribute("list", ArrayOf(it), func() {
					Elem(func() {
						Required("name")
					})
				})
			})
			Type("StructCopy", func() {
				Reference(holder)
				Attribute("list", func() {
					Description("redeclared")
				})
			})
		}, func(root *expr.RootExpr) *expr.AttributeExpr {
			return expr.AsArray(root.UserType("StructCopy").Attribute().Find("list").Type).ElemType
		}},
		{"union-meta-branch", func() {
			it := item()
			other := Type("StructOther", func() {
				Attribute("other", String)
			})
			choice := Type("StructChoice", OneOf(other, Int), func() {
				Attribute("item", it, func() {
					Required("name")
				})
			})
			Type("StructHolder", func() {
				Attribute("choice", choice, func() {
					Meta("oneof:type:field", "kind")
				})
			})
		}, func(root *expr.RootExpr) *expr.AttributeExpr {
			union := expr.AsUnion(root.UserType("StructHolder").Attribute().Find("choice").Type)
			for _, nat := range union.Values {
				if nat.Name == "item" {
					return nat.Attribute
				}
			}
			return nil
		}},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := expr.RunDSL(t, c.DSL)
			shared := root.UserType("StructItem")
			require.False(t, shared.Attribute().IsRequired("name"), "shared type must not change")

			elem := c.Element(root)
			require.NotNil(t, elem)
			local, ok := elem.Type.(expr.UserType)
			require.True(t, ok)
			require.NotSame(t, shared, local)
			require.NotNil(t, local.Attribute().Validation)
			require.Contains(t, local.Attribute().Validation.Required, "name")
		})
	}
}

// TestAttributeUnionMetaDoesNotChangeSharedNamedUnion verifies that union
// metadata set by an attribute DSL applies to a local copy of the union.
func TestAttributeUnionMetaDoesNotChangeSharedNamedUnion(t *testing.T) {
	root := expr.RunDSL(t, func() {
		first := Type("MetaFirst", func() {
			Attribute("a", String)
		})
		second := Type("MetaSecond", func() {
			Attribute("b", String)
		})
		shared := Type("MetaChoice", OneOf(first, second))
		Type("Holder", func() {
			Attribute("custom", shared, func() {
				Meta("oneof:type:field", "kind")
			})
			Attribute("plain", shared)
		})
	})

	shared := root.UserType("MetaChoice")
	require.Equal(t, "type", expr.AsUnion(shared).GetTypeKey())
	holder := root.UserType("Holder").Attribute()
	require.Same(t, shared, holder.Find("plain").Type)
	custom := holder.Find("custom").Type
	require.NotSame(t, shared, custom)
	require.Equal(t, "kind", expr.AsUnion(custom).GetTypeKey())
}

func laterType() expr.UserType {
	return Type("Later", func() {
		Attribute("name", String)
		Attribute("count", Int)
	})
}

func directField(_ *testing.T, field *expr.AttributeExpr) *expr.AttributeExpr {
	return field
}

func arrayElem(t *testing.T, field *expr.AttributeExpr) *expr.AttributeExpr {
	array := expr.AsArray(field.Type)
	require.NotNil(t, array)
	return array.ElemType
}

func mapElem(t *testing.T, field *expr.AttributeExpr) *expr.AttributeExpr {
	m := expr.AsMap(field.Type)
	require.NotNil(t, m)
	return m.ElemType
}
