package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// ProjectedUnionValidationDSL exercises the physical contexts of union branches.
func ProjectedUnionValidationDSL(shape string, required, viewed bool) func() {
	return func() {
		leaf := Type("Leaf", func() {
			Attribute("value", Int, func() {
				Minimum(1)
			})
			if required {
				Required("value")
			}
		})
		var branch expr.UserType = Type("Branch", func() {
			switch shape {
			case "nested":
				Attribute("child", leaf)
				Required("child")
			case "collections":
				Attribute("items", ArrayOf(leaf))
				Required("items")
			case "strings":
				Attribute("value", String, func() {
					Pattern("^[a-z]+$")
					MinLength(2)
					Enum("one", "two")
				})
				if required {
					Required("value")
				}
			default:
				Attribute("value", Int, func() {
					Minimum(1)
					Maximum(10)
					if shape == "default" {
						Default(7)
					}
				})
				if required {
					Required("value")
				}
			}
		})
		if shape == "array" {
			branch = Type("BranchList", ArrayOf(leaf))
		}
		if shape == "selected" {
			branch = ResultType("application/vnd.union.branch", func() {
				TypeName("SelectedBranch")
				Attributes(func() {
					Attribute("value", Int, func() {
						Minimum(1)
						Maximum(10)
					})
					Attribute("hidden", String, func() {
						Pattern("^[a-z]+$")
					})
					Required("value", "hidden")
				})
				View("default", func() {
					Attribute("value")
				})
				View("full", func() {
					Attribute("value")
					Attribute("hidden")
				})
			})
		}
		other := Type("Other", func() {
			Attribute("other", Boolean)
			Required("other")
		})
		choice := Type("Choice", OneOf(branch, other), func() {
			Untagged()
		})
		var result expr.DataType = choice
		if viewed {
			result = ResultType("application/vnd.union.validation", func() {
				TypeName("Result")
				Attributes(func() {
					Attribute("choice", choice)
					Required("choice")
				})
				View("default", func() {
					Attribute("choice")
				})
			})
		}
		Service("validation", func() {
			Method("show", func() {
				Result(result)
			})
		})
	}
}
