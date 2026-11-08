package ir

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
)

// A unary response documents the annotations its alias chain inherits, and an
// authored override on the alias replaces the inherited value.
func TestUnaryAliasChainKeepsInheritedAnnotations(t *testing.T) {
	for _, override := range []bool{false, true} {
		t.Run(fmt.Sprintf("override=%t", override), func(t *testing.T) {
			root := codegen.RunDSL(t, func() {
				base := dsl.Type("DeprecatedValue", dsl.String, func() {
					dsl.Description("Deprecated value.")
					dsl.Meta("openapi:deprecated", "true")
				})
				alias := dsl.Type("ValueAlias", base, func() {
					if override {
						dsl.Meta("openapi:deprecated", "false")
					}
				})
				dsl.Service("s", func() {
					dsl.Method("show", func() {
						dsl.Result(func() {
							dsl.Attribute("value", alias)
						})
						dsl.HTTP(func() {
							dsl.GET("/show")
						})
					})
				})
			})
			doc, err := BuildDocument(root.API, root.Types, root.ResultTypes)
			require.NoError(t, err)
			body := resolveSpecimenSchema(doc.Components.Schemas, unaryResultSchema(t, doc))
			value := resolveSpecimenSchema(doc.Components.Schemas, body.Properties["value"])
			require.Equal(t, !override, deprecatedAnywhere(doc.Components.Schemas, value, map[string]bool{}))
			require.Equal(t, []string{"Deprecated value."}, descriptionsAnywhere(doc.Components.Schemas, value, map[string]bool{}))
		})
	}
}

func TestUnaryRepeatedAliasAnnotations(t *testing.T) {
	for _, direct := range []bool{false, true} {
		for _, reverse := range []bool{false, true} {
			t.Run(fmt.Sprintf("direct=%t/reverse=%t", direct, reverse), func(t *testing.T) {
				root := codegen.RunDSL(t, func() {
					base := dsl.Type("Base", dsl.String, func() {
						dsl.Description("Inherited.")
						dsl.Meta("openapi:deprecated", "true")
					})
					alias := base
					if !direct {
						alias = dsl.Type("Alias", base)
					}
					dsl.Service("repeated", func() {
						dsl.Method("show", func() {
							dsl.Result(func() {
								names := []string{"first", "second", "local"}
								if reverse {
									names[0], names[2] = names[2], names[0]
								}
								for _, name := range names {
									if name == "local" {
										dsl.Attribute(name, alias, func() {
											dsl.Description("Local.")
											dsl.Meta("openapi:deprecated", "false")
										})
									} else {
										dsl.Attribute(name, alias)
									}
								}
							})
							dsl.HTTP(func() {
								dsl.GET("/repeated")
							})
						})
					})
				})
				doc, err := BuildDocument(root.API, root.Types, root.ResultTypes)
				require.NoError(t, err)
				body := resolveSpecimenSchema(doc.Components.Schemas, unaryResultSchema(t, doc))
				for _, name := range []string{"first", "second", "local"} {
					value := body.Properties[name]
					description := "Inherited."
					if name == "local" {
						description = "Local."
					}
					require.Equal(t, []string{description}, descriptionsAnywhere(doc.Components.Schemas, value, map[string]bool{}), name)
					require.Equal(t, name != "local", deprecatedAnywhere(doc.Components.Schemas, value, map[string]bool{}), name)
				}
			})
		}
	}
}
