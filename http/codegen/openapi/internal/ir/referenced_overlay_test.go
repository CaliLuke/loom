package ir

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// A referenced (componentized) base keeps its shared declaration. An occurrence
// that authors a boolean annotation decides it for that occurrence, including an
// explicit false that a sibling of the reference cannot express. A sibling use
// of the unchanged base keeps the declaration's value.
func TestReferencedBaseOverlayAuthoredBooleanAnnotations(t *testing.T) {
	for _, key := range []string{"openapi:deprecated", "openapi:readOnly", "openapi:writeOnly"} {
		for _, authored := range []string{"", "true", "false"} {
			for _, overrideFirst := range []bool{false, true} {
				for _, viaAlias := range []bool{false, true} {
					name := fmt.Sprintf("%s/authored=%q/override-first=%t/alias=%t", key, authored, overrideFirst, viaAlias)
					t.Run(name, func(t *testing.T) {
						root := codegen.RunDSL(t, referencedOverlayDSL(key, authored, overrideFirst, viaAlias))
						analyzer := NewAnalyzer(expr.NewRandom("referenced-overlay"), false)
						schema := analyzer.AnalyzeSchema(root.Services[0].Methods[0].Result)
						analyzer.finalizeRepresentations()
						require.NoError(t, analyzer.Err())
						body := resolveSpecimenSchema(analyzer.schemas, schema)
						require.True(t, booleanAnnotationAnywhere(analyzer.schemas, body.Properties["other"], key, map[string]bool{}),
							"the sibling use keeps the declaration value")
						want := authored != "false"
						require.Equal(t, want, booleanAnnotationAnywhere(analyzer.schemas, body.Properties["local"], key, map[string]bool{}),
							"the occurrence decides its own annotation")
						require.Equal(t, "true", root.UserType("Legacy").Attribute().Meta[key][0], "the shared declaration is not mutated")
					})
				}
			}
		}
	}
}

// Description and default beside a reference keep their existing sibling
// semantics: the occurrence value is local and the component keeps its own.
func TestReferencedBaseOverlaySiblingAnnotations(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		legacy := dsl.Type("Legacy", func() {
			dsl.Description("Declared legacy.")
			dsl.Attribute("id", dsl.String)
		})
		dsl.Service("overlay", func() {
			dsl.Method("show", func() {
				dsl.Result(func() {
					dsl.Attribute("local", legacy, func() {
						dsl.Description("Local legacy.")
					})
					dsl.Attribute("other", legacy)
				})
			})
		})
	})
	analyzer := NewAnalyzer(expr.NewRandom("referenced-siblings"), false)
	schema := analyzer.AnalyzeSchema(root.Services[0].Methods[0].Result)
	analyzer.finalizeRepresentations()
	body := resolveSpecimenSchema(analyzer.schemas, schema)
	require.Equal(t, "Local legacy.", body.Properties["local"].Description)
	require.Equal(t, "Declared legacy.", resolveSpecimenSchema(analyzer.schemas, body.Properties["other"]).Description)
}

func referencedOverlayDSL(key, authored string, overrideFirst, viaAlias bool) func() {
	return func() {
		legacy := dsl.Type("Legacy", func() {
			dsl.Meta(key, "true")
			dsl.Attribute("id", dsl.String)
			dsl.Required("id")
		})
		var local expr.DataType = legacy
		if viaAlias {
			local = dsl.Type("LegacyAlias", legacy, func() {
				if authored != "" {
					dsl.Meta(key, authored)
				}
			})
		}
		occurrence := func() {
			dsl.Attribute("local", local, func() {
				if authored != "" && !viaAlias {
					dsl.Meta(key, authored)
				}
			})
		}
		sibling := func() {
			dsl.Attribute("other", legacy)
		}
		dsl.Service("overlay", func() {
			dsl.Method("show", func() {
				dsl.Result(func() {
					if overrideFirst {
						occurrence()
						sibling()
					} else {
						sibling()
						occurrence()
					}
				})
			})
		})
	}
}

func booleanAnnotationAnywhere(components map[string]*Schema, schema *Schema, key string, seen map[string]bool) bool {
	if schema == nil {
		return false
	}
	switch key {
	case "openapi:deprecated":
		if schema.Deprecated {
			return true
		}
	case "openapi:readOnly":
		if schema.ReadOnly {
			return true
		}
	default:
		if schema.WriteOnly {
			return true
		}
	}
	if name, ok := schemaComponentName(schema.Ref); ok && !seen[name] {
		seen[name] = true
		if booleanAnnotationAnywhere(components, components[name], key, seen) {
			return true
		}
	}
	for _, branch := range append(append(append([]*Schema(nil), schema.AllOf...), schema.AnyOf...), schema.OneOf...) {
		if booleanAnnotationAnywhere(components, branch, key, seen) {
			return true
		}
	}
	return false
}
