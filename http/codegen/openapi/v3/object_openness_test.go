package openapiv3_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"

	. "github.com/CaliLuke/loom/dsl"
)

// TestObjectOpennessOccurrencePolicy checks that a named-type occurrence's
// openness selects its own component representation, while other uses of the
// type keep the type's policy. A reference wrapper never carries the policy,
// because additionalProperties beside $ref would reject every member.
func TestObjectOpennessOccurrencePolicy(t *testing.T) {
	for _, tc := range []struct {
		name        string
		closedAPI   bool
		occurrence  string
		description string
		childOpen   bool
		siblingOpen bool
	}{
		{"explicit open under API closed", true, "true", "", true, false},
		{"explicit closed on open type", false, "false", "", false, true},
		{"repeated closed under API closed", true, "false", "", false, false},
		{"closed occurrence with annotation overlay", false, "false", "Child record.", false, true},
		{"open occurrence with annotation overlay", true, "true", "Child record.", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			design := func() {
				API("openness", func() {
					if tc.closedAPI {
						Meta("openapi:closed-objects", "true")
					}
				})
				base := Type("OpennessBase", func() {
					Attribute("id", String)
				})
				Service("records", func() {
					Method("create", func() {
						Payload(func() {
							Attribute("child", base, func() {
								if tc.description != "" {
									Description(tc.description)
								}
								Meta("openapi:additionalProperties", tc.occurrence)
							})
							Attribute("sibling", base)
						})
						HTTP(func() {
							POST("/records")
						})
					})
				})
			}
			var spec struct {
				Components struct {
					Schemas map[string]map[string]any `yaml:"schemas"`
				} `yaml:"components"`
			}
			require.NoError(t, yaml.Unmarshal([]byte(renderYAMLOpenAPI(t, design)), &spec))
			schemas := spec.Components.Schemas
			body := schemas["CreateRequestBody"]["properties"].(map[string]any)
			for field, wantOpen := range map[string]bool{"child": tc.childOpen, "sibling": tc.siblingOpen} {
				wrapper := body[field].(map[string]any)
				require.NotContains(t, wrapper, "additionalProperties", "%s reference wrapper", field)
				require.NotContains(t, wrapper, "allOf", "%s must reference one component", field)
				ref, _ := wrapper["$ref"].(string)
				require.NotEmpty(t, ref, field)
				component := schemas[strings.TrimPrefix(ref, "#/components/schemas/")]
				additional, closed := component["additionalProperties"]
				if wantOpen {
					require.False(t, closed, "%s component %s must stay open", field, ref)
					continue
				}
				require.True(t, closed, "%s component %s must be closed", field, ref)
				require.Equal(t, false, additional)
			}
		})
	}
}

// TestObjectOpennessOccurrenceVariants checks occurrence variants of recursive
// and Extend-composed types: the variant is analyzed from the finalized type,
// keeps its constraints, and a recursive occurrence references its own variant.
func TestObjectOpennessOccurrenceVariants(t *testing.T) {
	components := func(t *testing.T, design func()) map[string]map[string]any {
		t.Helper()
		var spec struct {
			Components struct {
				Schemas map[string]map[string]any `yaml:"schemas"`
			} `yaml:"components"`
		}
		require.NoError(t, yaml.Unmarshal([]byte(renderYAMLOpenAPI(t, design)), &spec))
		return spec.Components.Schemas
	}
	target := func(schemas map[string]map[string]any, owner, field string) (string, map[string]any) {
		ref := schemas[owner]["properties"].(map[string]any)[field].(map[string]any)["$ref"].(string)
		name := strings.TrimPrefix(ref, "#/components/schemas/")
		return name, schemas[name]
	}

	t.Run("recursive occurrence", func(t *testing.T) {
		schemas := components(t, func() {
			API("openness", func() {})
			node := Type("OpennessNode", func() {
				Attribute("id", String)
				Attribute("next", "OpennessNode", func() {
					Meta("openapi:additionalProperties", "false")
				})
			})
			Service("records", func() {
				Method("create", func() {
					Payload(func() {
						Attribute("node", node)
					})
					HTTP(func() {
						POST("/records")
					})
				})
			})
		})
		name, node := target(schemas, "CreateRequestBody", "node")
		require.NotContains(t, node, "additionalProperties", "the authored use of %s stays open", name)
		variantName, variant := target(schemas, name, "next")
		require.NotEqual(t, name, variantName)
		require.Equal(t, false, variant["additionalProperties"])
		selfName, _ := target(schemas, variantName, "next")
		require.Equal(t, variantName, selfName, "the closed variant's own next keeps the occurrence policy")
	})

	t.Run("result type occurrence", func(t *testing.T) {
		schemas := components(t, func() {
			API("openness", func() {
				Meta("openapi:closed-objects", "true")
			})
			record := ResultType("application/vnd.openness.record", func() {
				TypeName("OpennessRecord")
				Attributes(func() {
					Attribute("id", String)
					Attribute("name", String)
					Required("id")
				})
				View("default", func() {
					Attribute("id")
					Attribute("name")
				})
			})
			Service("records", func() {
				Method("create", func() {
					Payload(func() {
						Attribute("open", record, func() {
							Meta("openapi:additionalProperties", "true")
						})
						Attribute("plain", record)
					})
					HTTP(func() {
						POST("/records")
					})
				})
			})
		})
		openName, open := target(schemas, "CreateRequestBody", "open")
		plainName, plain := target(schemas, "CreateRequestBody", "plain")
		require.NotEqual(t, openName, plainName)
		require.NotContains(t, open, "additionalProperties")
		require.Equal(t, false, plain["additionalProperties"])
		require.Len(t, open["properties"], 2, "the variant keeps the selected view's members")
		require.Len(t, plain["properties"], 2)
		require.Equal(t, []any{"id"}, open["required"])
	})

	t.Run("Extend-composed type", func(t *testing.T) {
		schemas := components(t, func() {
			API("openness", func() {})
			base := Type("OpennessBase", func() {
				Attribute("id", String)
				Required("id")
			})
			derived := Type("OpennessDerived", func() {
				Extend(base)
				Attribute("name", String)
			})
			Service("records", func() {
				Method("create", func() {
					Payload(func() {
						Attribute("child", derived, func() {
							Meta("openapi:additionalProperties", "false")
						})
						Attribute("sibling", derived)
					})
					HTTP(func() {
						POST("/records")
					})
				})
			})
		})
		childName, child := target(schemas, "CreateRequestBody", "child")
		siblingName, sibling := target(schemas, "CreateRequestBody", "sibling")
		require.NotEqual(t, childName, siblingName)
		require.Equal(t, false, child["additionalProperties"])
		require.NotContains(t, sibling, "additionalProperties")
		require.Equal(t, []any{"id"}, child["required"], "the variant keeps inherited requirements")
		require.Equal(t, []any{"id"}, sibling["required"])
		require.Len(t, child["properties"], 2)
	})
}
