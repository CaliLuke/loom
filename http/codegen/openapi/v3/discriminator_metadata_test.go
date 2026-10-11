package openapiv3_test

import (
	"path/filepath"
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	httpgen "github.com/CaliLuke/loom/http/codegen"
	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
	"github.com/stretchr/testify/require"
)

func TestDiscriminatorMetadataAcrossVersions(t *testing.T) {
	for _, target := range []string{"3.1", "3.2"} {
		for _, optional := range []bool{false, true} {
			name := target + "/required"
			if optional {
				name = target + "/optional"
			}
			t.Run(name, func(t *testing.T) {
				root := httpgen.RunHTTPDSL(t, discriminatorMetadataDSL(optional, "#/components/schemas/Fall%62ack"))
				root.API.Meta = map[string][]string{"openapi:version": {target}}
				files, err := openapiv3.Files(root)
				require.NoError(t, err)
				for _, file := range files {
					buffer := renderSection(t, file.AllSections()[0])
					raw := buffer.Bytes()
					version := "3.2.0"
					if target == "3.1" {
						version = "3.1.1"
					}
					validateOpenAPIVersion(t, raw, version)
					if filepath.Ext(file.Path) != ".json" {
						continue
					}
					spec := decodeOpenAPIJSON(t, raw)
					schemas := requireMap(t, requireMap(t, spec["components"], "components")["schemas"], "schemas")
					found := 0
					var visit func(any)
					visit = func(value any) {
						switch node := value.(type) {
						case map[string]any:
							if discriminator, ok := node["discriminator"].(map[string]any); ok {
								found++
								property := discriminator["propertyName"]
								required, _ := node["required"].([]any)
								switch {
								case optional && target == "3.1":
									require.Contains(t, required, property)
								case optional:
									require.NotContains(t, required, property)
								default:
									for _, branch := range requireSlice(t, node["oneOf"], "union branches") {
										resolved := resolveCardinalitySchema(t, spec, requireMap(t, branch, "branch"))
										require.Contains(t, resolved["required"], property)
									}
								}
								if target == "3.1" {
									require.NotContains(t, discriminator, "defaultMapping")
								} else {
									require.Equal(t, "#/components/schemas/Fallback", discriminator["defaultMapping"])
									require.Contains(t, schemas, "Fallback", "fallback-only components must survive pruning")
								}
							}
							for _, child := range node {
								visit(child)
							}
						case []any:
							for _, child := range node {
								visit(child)
							}
						}
					}
					visit(schemas)
					require.Positive(t, found)
				}
			})
		}
		t.Run(target+"/missing", func(t *testing.T) {
			root := httpgen.RunHTTPDSL(t, discriminatorMetadataDSL(true, ""))
			root.API.Meta = map[string][]string{"openapi:version": {target}}
			files, err := openapiv3.Files(root)
			require.ErrorContains(t, err, "requires an explicit defaultMapping")
			require.Empty(t, files)
		})
	}
}

func discriminatorMetadataDSL(optional bool, fallback string) func() {
	return func() {
		Type("Fallback", func() {
			Attribute("fallback", String)
		})
		alpha := Type("Alpha", func() {
			Attribute("alpha", String)
		})
		beta := Type("Beta", func() {
			Attribute("beta", String)
		})
		choice := Type("Choice", func() {
			OneOf("event", func() {
				if optional {
					Meta("openapi:discriminator:optional", "true")
				}
				if fallback != "" {
					Meta("openapi:discriminator:defaultMapping", fallback)
				}
				Attribute("alpha", alpha)
				Attribute("beta", beta)
			})
		})
		Service("events", func() {
			Method("one", func() {
				NoSecurity()
				Result(choice)
				HTTP(func() {
					GET("/one")
				})
			})
			Method("many", func() {
				NoSecurity()
				Result(ArrayOf(choice))
				HTTP(func() {
					GET("/many")
				})
			})
		})
	}
}
