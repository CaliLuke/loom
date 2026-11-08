package ir

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// Named components reached without a captured value-plan position must still
// acquire their authored defaults. Each valid design must build its document.
func TestPlanFreeComponentDefaults(t *testing.T) {
	for _, tc := range []struct {
		name    string
		mapping bool
		nested  bool
	}{
		{"allof-reference/root", false, false},
		{"allof-reference/nested", false, true},
		{"parent-hint-mapping/root", true, false},
		{"parent-hint-mapping/nested", true, true},
	} {
		for _, withDefault := range []bool{false, true} {
			name := tc.name + map[bool]string{false: "/control", true: "/default"}[withDefault]
			t.Run(name, func(t *testing.T) {
				root := codegen.RunDSL(t, func() {
					dsl.API("probe", func() {})
					member := func(name string) {
						dsl.Attribute(name, dsl.String, func() {
							if withDefault {
								dsl.Default("authored-default")
							}
						})
					}
					var top expr.UserType
					if tc.mapping {
						top = dsl.Type("Pet", func() {
							dsl.Attribute("kind", dsl.String)
							dsl.Required("kind")
							dsl.Meta("openapi:discriminator:propertyName", "kind")
							dsl.Meta("openapi:discriminator:mapping", "cat=Cat")
						})
						dsl.Type("Cat", func() {
							dsl.Extend(top)
							dsl.Meta("openapi:allOf:reference", "true")
							member("name")
						})
					} else {
						base := dsl.Type("Base", func() {
							member("inherited")
						})
						top = dsl.Type("Child", func() {
							dsl.Extend(base)
							dsl.Meta("openapi:allOf:reference", "true")
							dsl.Attribute("local", dsl.String)
						})
					}
					dsl.Service("probe", func() {
						dsl.Method("show", func() {
							if tc.nested {
								dsl.Payload(func() {
									dsl.Attribute("value", top)
								})
							} else {
								dsl.Payload(top)
							}
							dsl.HTTP(func() {
								dsl.POST("/probe")
							})
						})
					})
				})
				doc, err := BuildDocument(root.API, root.Types, root.ResultTypes)
				if doc != nil && doc.Components != nil {
					t.Logf("components: %v", slices.Sorted(func(yield func(string) bool) {
						for key := range doc.Components.Schemas {
							if !yield(key) {
								return
							}
						}
					}))
				}
				require.NoError(t, err)
			})
		}
	}
}

// One declaration with an opaque default reused at several occurrences keeps
// one component, regardless of the number of prepared occurrence positions.
func TestOpaqueDefaultDeclarationReuse(t *testing.T) {
	for _, value := range []any{defaultAppendKey(1), map[string]any{"k": defaultAppendKey(1)}} {
		root := codegen.RunDSL(t, func() {
			dsl.API("probe", func() {})
			foo := dsl.Type("Foo", func() {
				dsl.Attribute("v", dsl.Any, func() {
					dsl.Default(value)
				})
			})
			dsl.Service("probe", func() {
				dsl.Method("first", func() {
					dsl.Result(func() {
						dsl.Attribute("a", foo)
						dsl.Attribute("b", foo)
					})
					dsl.HTTP(func() {
						dsl.GET("/first")
					})
				})
				dsl.Method("second", func() {
					dsl.Payload(func() {
						dsl.Attribute("c", foo)
					})
					dsl.HTTP(func() {
						dsl.POST("/second")
					})
				})
			})
		})
		doc, err := BuildDocument(root.API, root.Types, root.ResultTypes)
		require.NoError(t, err)
		names := slices.Sorted(func(yield func(string) bool) {
			for key := range doc.Components.Schemas {
				if !yield(key) {
					return
				}
			}
		})
		t.Logf("components: %v", names)
		require.Equal(t, []string{"Foo"}, slices.DeleteFunc(names, func(name string) bool {
			return !strings.HasPrefix(name, "Foo")
		}), "one declaration with one default must publish one component")
	}
}

func TestPlanFreeBytesBaseDefault(t *testing.T) {
	base := &expr.UserTypeExpr{TypeName: "BytesBase", UID: "bytes-base", AttributeExpr: &expr.AttributeExpr{Type: expr.String, DefaultValue: "base-default"}}
	attribute := &expr.AttributeExpr{Type: expr.Bytes, Bases: []expr.DataType{base}}
	analyzer := NewAnalyzer(nil, false)
	schema := analyzer.AnalyzeSchema(attribute)
	require.NoError(t, analyzer.Err())
	require.NotNil(t, schema)
	require.Len(t, schema.AnyOf, 1)
	require.Equal(t, "base-default", schema.AnyOf[0].DefaultValue)
}

func TestPlanFreeReferenceDefaultErrors(t *testing.T) {
	for _, entry := range []string{"mapping", "allof", "bytes"} {
		t.Run(entry, func(t *testing.T) {
			target := &expr.UserTypeExpr{TypeName: "InvalidDefault", UID: "invalid-default", AttributeExpr: &expr.AttributeExpr{Type: expr.String, DefaultValue: 1}}
			analyzer := NewAnalyzer(nil, false)
			require.NotPanics(t, func() {
				switch entry {
				case "mapping":
					discriminator := &Discriminator{Mapping: make(map[string]string)}
					analyzer.recordProjectedMapping(discriminator, "invalid", target.Attribute(), target, "mapping-default")
					require.Empty(t, discriminator.Mapping["invalid"])
				case "allof":
					attribute := &expr.AttributeExpr{Type: &expr.Object{}, Bases: []expr.DataType{target}, Meta: expr.MetaExpr{"openapi:allOf:reference": {"true"}}}
					schema := analyzer.analyzeCompositionReferences(&Schema{}, attribute, "allof-default")
					require.Nil(t, schema.AllOf[0], "a rejected reference has no substitute schema")
				case "bytes":
					attribute := &expr.AttributeExpr{Type: expr.Bytes, Bases: []expr.DataType{target}}
					schema := &Schema{}
					analyzer.analyzeInlineBytes(schema, attribute, "bytes-default")
					require.Nil(t, schema.AnyOf[0], "a rejected reference has no substitute schema")
				}
			})
			require.ErrorContains(t, analyzer.Err(), "prepare referenced schema annotations")
			require.ErrorContains(t, analyzer.Err(), "default")
		})
	}
}
