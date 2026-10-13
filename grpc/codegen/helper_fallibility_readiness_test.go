package codegen

import (
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestHelperFallibilityReadiness(t *testing.T) {
	for _, shape := range []string{"sibling Any", "union Any", "recursive union Any", "nested Any", "pure control"} {
		for _, reverse := range []bool{false, true} {
			order := "plain-first"
			if reverse {
				order = "fallible-first"
			}
			t.Run(shape+"/"+order, func(t *testing.T) {
				root := RunGRPCDSL(t, helperFallibilityDSL(shape, reverse))
				runWrapperContractModule(t, root, `package probe

import "testing"

func TestCompiledHelperContracts(t *testing.T) {
}
`)
			})
		}
	}
}

func helperFallibilityDSL(shape string, reverse bool) func() {
	return func() {
		var leaf expr.UserType
		leaf = Type("Leaf", func() {
			Field(1, "name", String)
			if shape == "recursive union Any" {
				Field(2, "children", ArrayOf(leaf))
			}
		})
		plain := Type("Plain", func() {
			Field(1, "leaf", leaf)
		})
		var fallible expr.UserType
		switch shape {
		case "union Any", "recursive union Any":
			fallible = Type("Fallible", func() {
				OneOf("pick", func() {
					Field(1, "leaf", leaf)
					Field(2, "blob", Any)
				})
			})
		default:
			child := Any
			if shape == "pure control" {
				child = String
			}
			var nested expr.UserType
			if shape == "nested Any" {
				nested = Type("Nested", func() {
					Field(1, "value", child)
				})
			}
			fallible = Type("Fallible", func() {
				Field(1, "leaf", leaf)
				if shape == "nested Any" {
					Field(2, "nested", nested)
				} else {
					Field(2, "value", child)
				}
			})
		}
		Service("mixed", func() {
			first, second := plain, fallible
			if reverse {
				first, second = second, first
			}
			Method("first", func() {
				Payload(first)
				Result(first)
				GRPC(func() {
				})
			})
			Method("second", func() {
				Payload(second)
				Result(second)
				GRPC(func() {
				})
			})
		})
	}
}
