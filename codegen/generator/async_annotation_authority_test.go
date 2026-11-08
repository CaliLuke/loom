package generator

import (
	"bytes"
	"encoding/json/v2"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestAsyncAnnotationsAfterServiceGeneration(t *testing.T) {
	for _, serviceFirst := range []bool{false, true} {
		name := "openapi-only"
		if serviceFirst {
			name = "service-then-openapi"
		}
		t.Run(name, func(t *testing.T) {
			spec := asyncAnnotationDocument(t, testdata.SSEFieldPresenceDSL, serviceFirst)
			for path, expected := range map[string]map[string]any{
				"/optional":  {},
				"/defaulted": {"event": "default-event", "id": "default-id", "retry": float64(1500)},
			} {
				properties := asyncAnnotationMap(t, asyncAnnotationMessage(t, spec, path), "properties")
				for _, field := range []string{"event", "id", "retry"} {
					annotation := asyncAnnotationFields(asyncAnnotationMap(t, properties, field))
					value, exists := annotation["default"]
					want, present := expected[field]
					assert.Equal(t, present, exists, "%s.%s default presence", path, field)
					assert.Equal(t, want, value, "%s.%s default value", path, field)
				}
			}
		})
	}
}

func TestAsyncAnnotationSiblingAuthority(t *testing.T) {
	for _, annotation := range []string{"default", "description", "deprecated"} {
		t.Run(annotation, func(t *testing.T) {
			fixture := func() {
				dsl.Service("annotation-siblings", func() {
					dsl.Method("watch", func() {
						dsl.StreamingResult(func() {
							for _, field := range []string{"plain", "annotated"} {
								dsl.Attribute(field, asyncAnnotationNestedType(annotation, field == "annotated"))
							}
						})
						dsl.HTTP(func() {
							dsl.GET("/watch")
							dsl.ServerSentEvents()
						})
					})
				})
			}
			spec := asyncAnnotationDocument(t, fixture, true)
			properties := asyncAnnotationMap(t, asyncAnnotationMessage(t, spec, "/watch"), "properties")
			for _, field := range []string{"plain", "annotated"} {
				child := asyncAnnotationMap(t, properties, field)
				value := asyncAnnotationMap(t, asyncAnnotationMap(t, child, "properties"), "value")
				annotations := asyncAnnotationFields(value)
				actual, present := annotations[annotation]
				assert.Equal(t, field == "annotated", present, "%s annotation presence", field)
				if field == "annotated" {
					assert.Equal(t, map[string]any{"default": "owned-default", "description": "Owned nested description.", "deprecated": true}[annotation], actual)
				}
			}
		})
	}
}

// Evaluated named occurrences intentionally have no explicit public schema-name
// override, like the named wrappers introduced by service analysis. They have
// equal validation shapes but different authored annotation ownership.
func asyncAnnotationNestedType(annotation string, annotated bool) *expr.UserTypeExpr {
	value := &expr.AttributeExpr{Type: expr.String}
	name := "PlainNested"
	if annotated {
		name = "AnnotatedNested"
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

func TestAsyncAnnotationFalseOverride(t *testing.T) {
	for _, override := range []bool{false, true} {
		name := "inherited"
		if override {
			name = "explicit-false"
		}
		t.Run(name, func(t *testing.T) {
			fixture := func() {
				base := dsl.Type("DeprecatedValue", dsl.String, func() {
					dsl.Meta("openapi:deprecated", "true")
				})
				alias := dsl.Type("ValueAlias", base, func() {
					if override {
						dsl.Meta("openapi:deprecated", "false")
					}
				})
				dsl.Service("annotation-override", func() {
					dsl.Method("watch", func() {
						dsl.StreamingResult(func() {
							dsl.Attribute("value", alias)
						})
						dsl.HTTP(func() {
							dsl.GET("/watch")
							dsl.ServerSentEvents()
						})
					})
				})
			}
			spec := asyncAnnotationDocument(t, fixture, true)
			properties := asyncAnnotationMap(t, asyncAnnotationMessage(t, spec, "/watch"), "properties")
			value := asyncAnnotationMap(t, properties, "value")
			if override {
				assert.NotContains(t, asyncAnnotationFields(value), "deprecated")
				for _, layer := range asyncConjunction(t, value) {
					assert.NotContains(t, layer, "deprecated", "the authored false leaves no inherited deprecated layer")
				}
			} else {
				assert.Equal(t, true, value["deprecated"])
			}
			requireAsyncMessageMatchesPayload(t, spec, "/watch")
		})
	}
}

func TestAsyncRetainedDeclarationBindings(t *testing.T) {
	identity := asyncAnnotationDocument(t, testdata.TypeIdentityDSL, true)
	inbound := asyncAnnotationMap(t, asyncAnnotationMap(t, asyncAnnotationMap(t, asyncAnnotationMap(t, asyncAnnotationMap(t, asyncAnnotationMap(t, identity, "paths"), "/stream"), "get"), "x-loom-async"), "messages"), "inbound")
	require.Equal(t, "#/components/schemas/Other", asyncAnnotationMap(t, inbound, "schema")["$ref"])
	for _, test := range []struct {
		name     string
		authored bool
		mapped   bool
	}{
		{name: "derived"},
		{name: "derived-authored", authored: true},
		{name: "mapped", mapped: true},
		{name: "mapped-authored", authored: true, mapped: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			spec := asyncAnnotationDocument(t, recursiveAuthorityDSL(test.authored, test.mapped), true)
			message := asyncAnnotationMessage(t, spec, "/recursive")
			properties := asyncAnnotationMap(t, message, "properties")
			child := asyncAnnotationMap(t, properties, "child")
			ref, ok := child["$ref"].(string)
			require.True(t, ok)
			require.True(t, strings.HasPrefix(ref, "#/components/schemas/"))
			if !test.mapped {
				require.Equal(t, "#/components/schemas/RecursiveAuthority", ref)
			}
			require.NotContains(t, child, "properties", "the recursive cut remains a reference")
			components := asyncAnnotationMap(t, asyncAnnotationMap(t, spec, "components"), "schemas")
			component := asyncAnnotationMap(t, components, strings.TrimPrefix(ref, "#/components/schemas/"))
			label := asyncAnnotationMap(t, asyncAnnotationMap(t, component, "properties"), "label")
			require.Equal(t, "string", label["type"])
			require.Equal(t, "Authored label.", label["description"])
			switch {
			case test.authored:
				require.Equal(t, map[string]any{"label": "child-owned"}, child["example"])
				require.Equal(t, map[string]any{"label": "declaration-owned"}, component["example"])
			case test.mapped:
				// Ineligible async documentation retains its independent sampler.
				example := asyncAnnotationMap(t, child, "example")
				require.IsType(t, "", example["label"])
				require.NotContains(t, example, "child", "the cut must not synthesize another recursive value")
			default:
				// Eligible async documentation reuses the prepared event, including
				// the absence of a synthetic reference-local example at this cut.
				require.NotContains(t, child, "example")
			}
			if !test.mapped {
				if test.authored {
					require.Equal(t, map[string]any{"label": "declaration-owned"}, message["example"])
				}
				requireAsyncMessageMatchesPayload(t, spec, "/recursive")
				require.Equal(t, component["properties"], properties, "expanded children retain acquired declaration annotations")
			}
		})
	}
}

func recursiveAuthorityDSL(authored, mapped bool) func() {
	return func() {
		node := dsl.Type("RecursiveAuthority", func() {
			dsl.Attribute("label", dsl.String, "Authored label.")
			dsl.Attribute("child", "RecursiveAuthority", func() {
				if authored {
					dsl.Example(map[string]any{"label": "child-owned"})
				}
			})
			if authored {
				dsl.Example(map[string]any{"label": "declaration-owned"})
			}
		})
		dsl.Service("cut", func() {
			dsl.Method("watch", func() {
				dsl.StreamingResult(node)
				dsl.HTTP(func() {
					dsl.GET("/recursive")
					dsl.ServerSentEvents(func() {
						if mapped {
							dsl.SSEEventID("label")
						}
					})
				})
			})
		})
	}
}

func asyncAnnotationDocument(t *testing.T, fixture func(), serviceFirst bool) map[string]any {
	t.Helper()
	root := codegen.RunDSL(t, fixture)
	roots := []eval.Root{root}
	if serviceFirst {
		_, err := Service("example.com/annotation/gen", roots)
		require.NoError(t, err)
	}
	files, err := OpenAPI("example.com/annotation/gen", roots)
	require.NoError(t, err)
	for _, file := range files {
		if !strings.HasSuffix(file.Path, "openapi.json") {
			continue
		}
		var raw bytes.Buffer
		for _, section := range file.AllSections() {
			require.NoError(t, section.Write(&raw))
		}
		var result map[string]any
		require.NoError(t, json.Unmarshal(raw.Bytes(), &result))
		return result
	}
	t.Fatal("OpenAPI JSON file missing")
	return nil
}

func asyncAnnotationMessage(t *testing.T, spec map[string]any, path string) map[string]any {
	t.Helper()
	current := spec
	for _, key := range []string{"paths", path, "get", "x-loom-async", "messages", "outbound", "schema"} {
		current = asyncAnnotationMap(t, current, key)
	}
	return current
}

func asyncAnnotationMap(t *testing.T, object map[string]any, key string) map[string]any {
	t.Helper()
	value, ok := object[key].(map[string]any)
	require.True(t, ok, "missing object %q in %#v", key, object)
	return value
}

// Preserve key presence as well as value; an absent annotation is distinct from
// an authored false, zero, empty string, or JSON null. Extension data stays data.
func asyncAnnotationFields(schema map[string]any) map[string]any {
	result := make(map[string]any)
	for key, value := range schema {
		switch key {
		case "default", "example", "examples", "title", "description", "readOnly", "writeOnly", "deprecated":
			result[key] = value
		default:
			if strings.HasPrefix(key, "x-") {
				result[key] = value
			}
		}
	}
	return result
}
