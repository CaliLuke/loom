package ir

import (
	"encoding/json/jsontext"
	"testing"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/require"
)

func TestDiscriminatorNamedOccurrenceOverlay(t *testing.T) {
	for _, parent := range []bool{false, true} {
		for _, allOf := range []bool{false, true} {
			name := "union/ref"
			if parent {
				name = "parent/ref"
			}
			if allOf {
				name += "/allOf"
			}
			t.Run(name, func(t *testing.T) {
				root := discriminatorOverlayDSL(t, parent)
				owner := root.UserType("Owner")
				analyzer := NewAnalyzer(root.API.ExampleGenerator, false, WithExampleValue(nil))
				analyzer.initializeDiscriminatorHints(root.API, root.Types, root.ResultTypes)
				original := analyzer.AnalyzeSchema(&expr.AttributeExpr{Type: owner})
				require.NoError(t, analyzer.Err())
				originalComponent := discriminatorOverlayComponent(t, analyzer, original)
				meta := expr.MetaExpr{
					"openapi:discriminator:propertyName":      {"localKind"},
					"openapi:discriminator:extension:x-owner": {`"local"`},
				}
				if parent {
					meta["openapi:discriminator:mapping"] = []string{"local=Child"}
				} else {
					meta["openapi:discriminator:defaultMapping"] = []string{"Cat"}
				}
				if allOf {
					meta["openapi:allOf:reference"] = []string{"true"}
				}
				occurrence := &expr.AttributeExpr{Type: owner, Description: "local annotation", Title: "Local title", Meta: meta}
				if parent {
					occurrence.Validation = &expr.ValidationExpr{Required: []string{"other"}}
				}
				var schema *Schema
				require.NotPanics(t, func() {
					schema = analyzer.AnalyzeSchema(occurrence)
				})
				require.NoError(t, analyzer.Err())
				require.Equal(t, "local annotation", schema.Description)
				require.Equal(t, "Local title", schema.Title)
				require.Nil(t, schema.Discriminator, "reference annotation does not own the discriminator graph")
				component := discriminatorOverlayComponent(t, analyzer, schema)
				require.Equal(t, "localKind", component.Discriminator.PropertyName)
				require.Equal(t, jsontext.Value(`"local"`), component.Discriminator.Extensions["x-owner"])
				require.Equal(t, "kind", originalComponent.Discriminator.PropertyName)
				require.NotSame(t, originalComponent, component)
				require.NotContains(t, originalComponent.Discriminator.Extensions, "x-owner")
				if parent {
					require.Equal(t, map[string]string{"local": "#/components/schemas/Child"}, component.Discriminator.Mapping)
					require.Equal(t, map[string]string{"child": "#/components/schemas/Child"}, originalComponent.Discriminator.Mapping)
					require.Contains(t, schema.Required, "other")
				} else {
					require.Len(t, component.OneOf, 2)
					require.Equal(t, component.OneOf[0].Ref, component.Discriminator.Mapping["alias"])
					require.Equal(t, component.Discriminator.Mapping["cat"], component.Discriminator.Mapping["alias"])
					require.Equal(t, "#/components/schemas/Cat", component.Discriminator.DefaultMapping)
				}
				require.Equal(t, []string{"kind"}, owner.Attribute().Meta["openapi:discriminator:propertyName"])
				require.Equal(t, []string{"localKind"}, occurrence.Meta["openapi:discriminator:propertyName"])
				again := analyzer.AnalyzeSchema(&expr.AttributeExpr{Type: owner})
				require.NoError(t, analyzer.Err())
				require.Equal(t, original.Ref, again.Ref)
			})
		}
	}
}

func TestDiscriminatorNamedOccurrenceOverlayInheritsHint(t *testing.T) {
	for _, parent := range []bool{false, true} {
		name := "union"
		if parent {
			name = "parent"
		}
		t.Run(name, func(t *testing.T) {
			root := discriminatorOverlayDSL(t, parent)
			owner := root.UserType("Owner")
			analyzer := NewAnalyzer(root.API.ExampleGenerator, false, WithExampleValue(nil))
			analyzer.initializeDiscriminatorHints(root.API, root.Types, root.ResultTypes)
			annotated := analyzer.AnalyzeSchema(&expr.AttributeExpr{Type: owner, Description: "local description"})
			require.NoError(t, analyzer.Err())
			plain := analyzer.AnalyzeSchema(&expr.AttributeExpr{Type: owner})
			require.NoError(t, analyzer.Err())
			require.Equal(t, plain.Ref, annotated.Ref)
			require.Equal(t, "local description", annotated.Description)
			require.Nil(t, annotated.Discriminator)
			require.Empty(t, plain.Description)
			component := discriminatorOverlayComponent(t, analyzer, annotated)
			require.Equal(t, "kind", component.Discriminator.PropertyName)
			require.Empty(t, component.Description)
		})
	}
}

func discriminatorOverlayDSL(t *testing.T, parent bool) *expr.RootExpr {
	t.Helper()
	return codegen.RunDSL(t, func() {
		dsl.API("overlay", func() {
		})
		if parent {
			owner := dsl.Type("Owner", func() {
				dsl.Attribute("kind", dsl.String)
				dsl.Attribute("localKind", dsl.String)
				dsl.Attribute("other", dsl.String)
				dsl.Required("kind", "localKind")
				dsl.Meta("openapi:discriminator:propertyName", "kind")
				dsl.Meta("openapi:discriminator:mapping", "child=Child")
			})
			dsl.Type("Child", func() {
				dsl.Extend(owner)
				dsl.Meta("openapi:allOf:reference", "true")
				dsl.Attribute("name", dsl.String)
			})
			return
		}
		cat := dsl.Type("Cat", func() {
			dsl.Attribute("kind", dsl.String)
			dsl.Attribute("localKind", dsl.String)
			dsl.Required("kind", "localKind")
		})
		dog := dsl.Type("Dog", func() {
			dsl.Attribute("kind", dsl.String)
			dsl.Attribute("localKind", dsl.String)
			dsl.Required("kind", "localKind")
		})
		dsl.Type("Owner", dsl.OneOf(func() {
			dsl.Untagged()
			dsl.Attribute("Cat", cat, func() {
				dsl.Meta("openapi:discriminator:mapping", "cat", "alias")
			})
			dsl.Attribute("Dog", dog)
		}), func() {
			dsl.Meta("openapi:discriminator:propertyName", "kind")
		})
	})
}

func discriminatorOverlayComponent(t *testing.T, analyzer *Analyzer, schema *Schema) *Schema {
	t.Helper()
	if schema.Ref == "" {
		require.Len(t, schema.AllOf, 1)
		schema = schema.AllOf[0]
	}
	name, ok := schemaComponentName(schema.Ref)
	require.True(t, ok)
	result := analyzer.schemas[name]
	require.NotNil(t, result)
	require.NotNil(t, result.Discriminator)
	return result
}
