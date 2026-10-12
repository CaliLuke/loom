package service

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestProjectedUnionValidationContext(t *testing.T) {
	for _, test := range []struct {
		name     string
		field    func()
		required bool
		value    string
	}{
		{"required range", func() {
			dsl.Attribute("value", dsl.Int, func() {
				dsl.Minimum(1)
			})
		}, true, "*v.Value < 1"},
		{"optional range", func() {
			dsl.Attribute("value", dsl.Int, func() {
				dsl.Minimum(1)
			})
		}, false, "*v.Value < 1"},
		{"required pattern", func() {
			dsl.Attribute("value", dsl.String, func() {
				dsl.Pattern("^[a-z]+$")
			})
		}, true, "*v.Value"},
		{"required enum", func() {
			dsl.Attribute("value", dsl.String, func() {
				dsl.Enum("one", "two")
			})
		}, true, "*v.Value"},
		{"required length", func() {
			dsl.Attribute("value", dsl.String, func() {
				dsl.MinLength(2)
			})
		}, true, "*v.Value"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := expr.RunDSL(t, func() {
				branch := dsl.Type("Branch", func() {
					test.field()
					if test.required {
						dsl.Required("value")
					}
				})
				other := dsl.Type("Other", func() {
					dsl.Attribute("other", dsl.Boolean)
				})
				dsl.Type("Choice", dsl.OneOf(branch, other), func() {
					dsl.Untagged()
				})
			})
			union := expr.AsUnion(root.UserType("Choice"))
			view := buildViewUnionTypeData(union, codegen.NewNameScope(), nil)
			ordinary := buildUnionTypeData(union, codegen.NewNameScope(), nil)
			require.Len(t, view.Fields, 2)
			if test.required {
				require.NotContains(t, ordinary.Fields[0].ValidateCode, "*v.Value")
			} else {
				require.Contains(t, ordinary.Fields[0].ValidateCode, "*v.Value")
			}
			require.Contains(t, view.Fields[0].ValidateCode, test.value)
			require.Contains(t, view.Fields[0].ValidateCode, "v.Value != nil")
			if test.required {
				require.Contains(t, view.Fields[0].ValidateCode, "MissingFieldError")
			}
		})
	}
}
