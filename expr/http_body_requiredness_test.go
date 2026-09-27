package expr_test

import (
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/require"
)

// TestExplicitBodyRequirednessInheritance preserves selected payload requirements
// for inline and named sources without leaking path-only fields.
func TestExplicitBodyRequirednessInheritance(t *testing.T) {
	for _, tc := range []struct {
		name                string
		named, bodyRequired bool
	}{
		{"inline payload", false, false},
		{"named payload", true, false},
		{"named payload explicit required", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := expr.RunDSL(t, func() {
				payload := func() {
					Attribute("id", String)
					Attribute("name", String)
					Attribute("optional", String)
					Required("id", "name")
				}
				named := Type("Input", payload)
				Service("sender", func() {
					Method("send", func() {
						if tc.named {
							Payload(named)
						} else {
							Payload(payload)
						}
						HTTP(func() {
							POST("/{id}")
							Param("id")
							Body(func() {
								Attribute("name")
								Attribute("optional")
								if tc.bodyRequired {
									Required("name")
								}
							})
						})
					})
				})
			})
			body := root.API.HTTP.Services[0].HTTPEndpoints[0].Body
			require.ElementsMatch(t, []string{"name"}, body.AllRequired())
		})
	}
}

// TestExplicitBodyReferencedRequiredness resolves inherited requirements before
// validating an explicitly selected HTTP body.
func TestExplicitBodyReferencedRequiredness(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		t.Run(map[bool]string{false: "inherited", true: "explicit"}[explicit], func(t *testing.T) {
			root := expr.RunDSL(t, func() {
				source := Type("Source", func() {
					Attribute("id", String)
					Attribute("name", String)
					Attribute("optional", String)
					Required("id", "name")
				})
				Service("sender", func() {
					Method("send", func() {
						Payload(func() {
							Reference(source)
							Attribute("id")
							Attribute("name")
							Attribute("optional")
						})
						HTTP(func() {
							POST("/{id}")
							Param("id")
							Body(func() {
								Attribute("name")
								Attribute("optional")
								if explicit {
									Required("name")
								}
							})
						})
					})
				})
			})
			require.ElementsMatch(t, []string{"name"}, root.API.HTTP.Services[0].HTTPEndpoints[0].Body.AllRequired())
		})
	}
}
