package ir

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
)

// Same-shaped declarations that author different annotations are different
// contracts. Component acquisition must not let the first acquired declaration
// stand in for the other, in either declaration order or naming style.
func TestComponentAcquisitionKeepsAnnotationDistinctDeclarations(t *testing.T) {
	for _, annotation := range []string{"default", "description", "deprecated"} {
		for _, registered := range []bool{false, true} {
			for _, annotatedFirst := range []bool{false, true} {
				name := fmt.Sprintf("%s/registered=%t/annotated-first=%t", annotation, registered, annotatedFirst)
				t.Run(name, func(t *testing.T) {
					root := codegen.RunDSL(t, annotationSiblingsDSL(annotation, registered, annotatedFirst))
					doc, err := BuildDocument(root.API, root.Types, root.ResultTypes)
					require.NoError(t, err)
					body := resolveSpecimenSchema(doc.Components.Schemas, unaryResultSchema(t, doc))
					for _, field := range []string{"plain", "annotated"} {
						member := resolveSpecimenSchema(doc.Components.Schemas, body.Properties[field])
						value := member.Properties["value"]
						require.NotNil(t, value, field)
						actual, present := schemaAnnotation(value, annotation)
						require.Equal(t, field == "annotated", present, "%s %s presence", field, annotation)
						if present {
							require.Equal(t, map[string]any{"default": "owned-default", "description": "Owned nested description.", "deprecated": true}[annotation], actual)
						}
					}
				})
			}
		}
	}
}

// An occurrence that authors an annotation replaces the inherited value. An
// explicit false cannot be expressed beside an inherited true layer, and an
// authored description must not leave the inherited one on the base.
func TestUserTypeOverlayAuthoredAnnotationReplacesInherited(t *testing.T) {
	for _, test := range []struct {
		annotation string
		override   bool
	}{
		{"deprecated", false}, {"deprecated", true}, {"description", false}, {"description", true},
	} {
		t.Run(fmt.Sprintf("%s/override=%t", test.annotation, test.override), func(t *testing.T) {
			root := codegen.RunDSL(t, func() {
				base := dsl.Type("AnnotatedValue", dsl.String, func() {
					if test.annotation == "deprecated" {
						dsl.Meta("openapi:deprecated", "true")
					} else {
						dsl.Description("Inherited description.")
					}
				})
				alias := dsl.Type("ValueAlias", base, func() {
					if !test.override {
						return
					}
					if test.annotation == "deprecated" {
						dsl.Meta("openapi:deprecated", "false")
					} else {
						dsl.Description("Authored description.")
					}
				})
				dsl.Service("annotation-override", func() {
					dsl.Method("show", func() {
						dsl.Result(func() {
							dsl.Attribute("value", alias)
						})
					})
				})
			})
			analyzer := NewAnalyzer(expr.NewRandom("overlay"), false)
			schema := analyzer.AnalyzeSchema(root.Services[0].Methods[0].Result)
			analyzer.finalizeRepresentations()
			value := resolveSpecimenSchema(analyzer.schemas, schema).Properties["value"]
			if test.annotation == "deprecated" {
				require.Equal(t, !test.override, deprecatedAnywhere(analyzer.schemas, value, map[string]bool{}),
					"the authored override decides deprecation; no inherited layer survives")
				return
			}
			want := []string{"Inherited description."}
			if test.override {
				want = []string{"Authored description."}
			}
			require.Equal(t, want, descriptionsAnywhere(analyzer.schemas, value, map[string]bool{}))
		})
	}
}

func annotationSiblingsDSL(annotation string, registered, annotatedFirst bool) func() {
	return func() {
		nested := func(annotated bool) expr.DataType {
			name := "PlainNested"
			if annotated {
				name = "AnnotatedNested"
			}
			if !registered {
				return annotationNestedType(annotation, name, annotated)
			}
			return dsl.Type(name, func() {
				dsl.Attribute("value", dsl.String, func() {
					if !annotated {
						return
					}
					switch annotation {
					case "default":
						dsl.Default("owned-default")
					case "description":
						dsl.Description("Owned nested description.")
					case "deprecated":
						dsl.Meta("openapi:deprecated", "true")
					}
				})
			})
		}
		order := []bool{false, true}
		if annotatedFirst {
			order = []bool{true, false}
		}
		types := make(map[bool]expr.DataType, 2)
		for _, annotated := range order {
			types[annotated] = nested(annotated)
		}
		dsl.Service("annotation-siblings", func() {
			dsl.Method("show", func() {
				dsl.Result(func() {
					for _, annotated := range order {
						field := "plain"
						if annotated {
							field = "annotated"
						}
						dsl.Attribute(field, types[annotated])
					}
				})
				dsl.HTTP(func() {
					dsl.GET("/show")
				})
			})
		})
	}
}

// annotationNestedType mirrors evaluated named occurrences without an explicit
// public schema name, such as wrappers introduced by service analysis.
func annotationNestedType(annotation, name string, annotated bool) *expr.UserTypeExpr {
	value := &expr.AttributeExpr{Type: expr.String}
	if annotated {
		switch annotation {
		case "default":
			value.DefaultValue = "owned-default"
		case "description":
			value.Description = "Owned nested description."
		case "deprecated":
			value.Meta = expr.MetaExpr{"openapi:deprecated": {"true"}}
		}
	}
	return &expr.UserTypeExpr{TypeName: name, AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{{Name: "value", Attribute: value}}}}
}

func unaryResultSchema(t *testing.T, doc *Document) *Schema {
	t.Helper()
	for _, item := range doc.Paths {
		for _, operation := range item.Operations {
			response := operation.Responses["200"]
			require.NotNil(t, response)
			value := response.Value
			if response.Ref != "" {
				value = doc.Components.Responses[response.Ref[len(ResponseComponentRefPrefix):]].Value
			}
			return value.Content["application/json"].Schema
		}
	}
	t.Fatal("no operation")
	return nil
}

func schemaAnnotation(schema *Schema, annotation string) (any, bool) {
	switch annotation {
	case "default":
		return schema.DefaultValue, schema.DefaultValue != nil
	case "description":
		return schema.Description, schema.Description != ""
	default:
		return schema.Deprecated, schema.Deprecated
	}
}

func deprecatedAnywhere(components map[string]*Schema, schema *Schema, seen map[string]bool) bool {
	if schema == nil {
		return false
	}
	if schema.Deprecated {
		return true
	}
	if name, ok := schemaComponentName(schema.Ref); ok && !seen[name] {
		seen[name] = true
		if deprecatedAnywhere(components, components[name], seen) {
			return true
		}
	}
	for _, branch := range append(append(append([]*Schema(nil), schema.AllOf...), schema.AnyOf...), schema.OneOf...) {
		if deprecatedAnywhere(components, branch, seen) {
			return true
		}
	}
	return false
}

func descriptionsAnywhere(components map[string]*Schema, schema *Schema, seen map[string]bool) []string {
	if schema == nil {
		return nil
	}
	var found []string
	if schema.Description != "" {
		found = append(found, schema.Description)
	}
	if name, ok := schemaComponentName(schema.Ref); ok && !seen[name] {
		seen[name] = true
		found = append(found, descriptionsAnywhere(components, components[name], seen)...)
	}
	for _, branch := range append(append(append([]*Schema(nil), schema.AllOf...), schema.AnyOf...), schema.OneOf...) {
		found = append(found, descriptionsAnywhere(components, branch, seen)...)
	}
	return found
}

func TestComponentAnnotationAcquisitionDoesNotRunCodec(t *testing.T) {
	codec := &fingerprintCodec{}
	attribute := &expr.AttributeExpr{Type: expr.Any, DefaultValue: codec}
	target, err := representation.PrepareStandaloneValues(attribute, expr.NewRandom("annotation-acquisition"), true)
	require.NoError(t, err)
	require.NotNil(t, target)
	before := codec.calls.Load()
	analyzer := NewAnalyzer(nil, false)
	analyzer.preparedExamples = target
	first := analyzer.componentAnnotationFingerprint(attribute, target.Plan.Root())
	second := analyzer.componentAnnotationFingerprint(attribute, target.Plan.Root())
	require.NoError(t, analyzer.Err())
	require.Equal(t, first, second)
	require.Equal(t, before, codec.calls.Load(), "component lookup must only consume retained checked annotations")
}
