package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestEffectiveCompositionRenderedJSONTags(t *testing.T) {
	for _, tc := range []struct {
		name, full, expected string
		required, nullable   bool
	}{
		{"optional equivalent", "one,omitempty", " `json:\"one,omitempty\"`", false, false},
		{"optional conflict", "one", "", false, false},
		{"required equivalent", "one", " `json:\"one\"`", true, false},
		{"required conflict", "one,omitempty", "", true, false},
		{"nullable normalizes full", "one", " `json:\"one,omitzero\"`", false, true},
		{"nullable normalizes omission", "one,omitempty", " `json:\"one,omitzero\"`", false, true},
	} {
		for _, local := range []bool{false, true} {
			for _, reverse := range []bool{false, true} {
				t.Run(tc.name+map[bool]string{false: "/peer", true: "/local"}[local]+map[bool]string{false: "/forward", true: "/reverse"}[reverse], func(t *testing.T) {
					var derived expr.UserType
					design := func() {
						full := func() {
							dsl.Meta("struct:tag:json", tc.full)
							if tc.nullable {
								dsl.Nullable()
							}
						}
						name := func() {
							dsl.Meta("struct:tag:json:name", "one")
							if tc.nullable {
								dsl.Nullable()
							}
						}
						first, second := full, name
						if reverse {
							first, second = second, first
						}
						a := dsl.Type("A", func() {
							dsl.Attribute("n", dsl.String, first)
							if tc.required {
								dsl.Required("n")
							}
						})
						if tc.required {
							ancestor := a
							a = dsl.Type("Middle", func() {
								dsl.Extend(ancestor)
							})
						}
						var b expr.UserType
						if !local {
							b = dsl.Type("B", func() {
								dsl.Attribute("n", dsl.String, second)
							})
						}
						derived = dsl.Type("Derived", func() {
							dsl.Extend(a)
							if local {
								dsl.Attribute("n", dsl.String, second)
							} else {
								dsl.Extend(b)
							}
						})
					}
					if tc.expected == "" {
						err := expr.RunInvalidDSL(t, design)
						require.ErrorContains(t, err, "inherited JSON tag")
						return
					}
					expr.RunDSL(t, design)
					require.Equal(t, tc.expected, AttributeTagsWithName(derived.Attribute(), "n", derived.Attribute().Find("n")))
				})
			}
		}
	}
}

func TestEffectiveCompositionSkipTags(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		t.Run(map[bool]string{false: "full first", true: "name first"}[reverse], func(t *testing.T) {
			var derived expr.UserType
			expr.RunDSL(t, func() {
				first, second := "struct:tag:json", "struct:tag:json:name"
				if reverse {
					first, second = second, first
				}
				a := dsl.Type("A", func() {
					dsl.Attribute("secret", dsl.String, func() {
						dsl.Meta(first, "-")
					})
				})
				b := dsl.Type("B", func() {
					dsl.Attribute("secret", dsl.String, func() {
						dsl.Meta(second, "-")
					})
				})
				derived = dsl.Type("Derived", func() {
					dsl.Extend(a)
					dsl.Extend(b)
				})
			})
			require.Equal(t, " `json:\"-\"`", AttributeTagsWithName(derived.Attribute(), "secret", derived.Attribute().Find("secret")))
		})
	}
}
