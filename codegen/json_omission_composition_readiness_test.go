package codegen

import (
	"testing"

	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/require"
)

func TestJSONOmissionCompositionReadiness(t *testing.T) {
	for _, tc := range []struct {
		name, full, want              string
		defaulted, required, conflict bool
	}{
		{"pointer absence equivalent", "value,omitzero", " `json:\"value,omitzero\"`", false, false, false},
		{"pointer empty loss conflicts", "value,omitempty", "", false, false, true},
		{"pointer forced inclusion conflicts", "value", "", false, false, true},
		{"materialized value retains zero", "value", " `json:\"value\"`", true, false, false},
		{"materialized zero omission conflicts", "value,omitzero", "", true, false, true},
		{"required retention control", "value", " `json:\"value\"`", false, true, false},
	} {
		for _, local := range []bool{false, true} {
			for _, reverse := range []bool{false, true} {
				t.Run(tc.name+map[bool]string{false: "/peer", true: "/local"}[local]+map[bool]string{false: "/forward", true: "/reverse"}[reverse], func(t *testing.T) {
					var derived expr.UserType
					design := func() {
						full := func() {
							dsl.Meta("struct:tag:json", tc.full)
						}
						name := func() {
							dsl.Meta("struct:tag:json:name", "value")
						}
						first, second := full, name
						if reverse {
							first, second = second, first
						}
						base := dsl.Type("Base", func() {
							dsl.Attribute("value", dsl.String, func() {
								first()
								if tc.defaulted {
									dsl.Default("fallback")
								}
							})
							if tc.required {
								dsl.Required("value")
							}
						})
						middle := dsl.Type("Middle", func() {
							dsl.Extend(base)
						})
						var peer expr.UserType
						if !local {
							peer = dsl.Type("Peer", func() {
								dsl.Attribute("value", dsl.String, second)
							})
						}
						derived = dsl.Type("Derived", func() {
							dsl.Extend(middle)
							if local {
								dsl.Attribute("value", dsl.String, second)
							} else {
								dsl.Extend(peer)
							}
						})
					}
					if tc.conflict {
						require.ErrorContains(t, expr.RunInvalidDSL(t, design), "inherited JSON tag")
						return
					}
					expr.RunDSL(t, design)
					require.Equal(t, tc.want, AttributeTagsWithName(derived.Attribute(), "value", derived.Attribute().Find("value")))
				})
			}
		}
	}
}
