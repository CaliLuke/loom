package generator

import (
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

// Async inline contracts have an existing consumer-owned shape. View metadata
// still controls runtime response projection; this test does not change it.
func TestAsyncConsumerViewAuthority(t *testing.T) {
	cases := []struct {
		name    string
		fixture func()
		path    string
		kind    string
	}{
		{"scalar", testdata.StreamingResultPrimitiveDSL, "/", "string"},
		{"result object", testdata.StreamingResultWithExplicitViewDSL, "/{x}", "object"},
		{"result collection", testdata.StreamingResultCollectionWithExplicitViewDSL, "/{x}", "array"},
		{"payload object", testdata.StreamingPayloadResultWithExplicitViewDSL, "/", "object"},
		{"payload collection", testdata.StreamingPayloadResultCollectionWithExplicitViewDSL, "/", "array"},
		{"bidirectional object", testdata.BidirectionalStreamingResultWithExplicitViewDSL, "/", "object"},
		{"bidirectional collection", testdata.BidirectionalStreamingResultCollectionWithExplicitViewDSL, "/", "array"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			spec := asyncAnnotationDocument(t, test.fixture, true)
			schema := asyncAnnotationMessage(t, spec, test.path)
			require.Equal(t, test.kind, schema["type"])
			require.NotContains(t, schema, "description", "ordinary projection must not add generated view annotations to inline contracts")
			if test.kind == "string" {
				return
			}
			if test.kind == "array" {
				schema = asyncAnnotationMap(t, schema, "items")
			}
			require.NotContains(t, schema, "description")
			properties := asyncAnnotationMap(t, schema, "properties")
			require.Len(t, properties, 3)
			for _, field := range []string{"a", "b", "c"} {
				require.Contains(t, properties, field)
				require.Contains(t, asyncAnnotationMap(t, properties, field), "example")
			}
			example := asyncAnnotationMap(t, schema, "example")
			require.Contains(t, example, "b", "the parent inline sampler's complete shape remains authoritative")
			require.Contains(t, example, "c")
		})
	}
}

// An eligible SSE message is the completed common event outcome (#976 R7), so
// it keeps the shared alias-refinement conjunction and nullable wrapper. These
// assertions check the effective contract, not the superseded flat shape.
func TestAsyncConsumerOwnsAliasPolicy(t *testing.T) {
	for _, policy := range []string{"enum refinement", "occurrence nullable"} {
		t.Run(policy, func(t *testing.T) {
			fixture := func() {
				primitive := dsl.String
				if policy == "enum refinement" {
					primitive = dsl.Int
				}
				inner := dsl.Type("Inner", primitive, func() {
					if policy == "enum refinement" {
						dsl.Enum(1, 2)
					}
				})
				outer := dsl.Type("Outer", inner, func() {
					if policy == "enum refinement" {
						dsl.Enum(2)
					}
				})
				dsl.Service("consumer-alias-policy", func() {
					dsl.Method("watch", func() {
						dsl.StreamingResult(func() {
							dsl.Attribute("value", outer, func() {
								if policy == "occurrence nullable" {
									dsl.Nullable()
								}
							})
						})
						dsl.HTTP(func() {
							dsl.GET("/")
							dsl.ServerSentEvents()
						})
					})
				})
			}
			spec := asyncAnnotationDocument(t, fixture, true)
			properties := asyncAnnotationMap(t, asyncAnnotationMessage(t, spec, "/"), "properties")
			value := asyncAnnotationMap(t, properties, "value")
			if policy == "enum refinement" {
				require.False(t, asyncNullable(value))
				require.Equal(t, []any{float64(2)}, asyncEffectiveEnum(t, value))
				require.Equal(t, "integer", asyncConjunctionType(t, value))
			} else {
				require.True(t, asyncNullable(value), "the occurrence nullability is retained")
				require.Equal(t, "string", asyncConjunctionType(t, asyncNonNull(t, value)))
			}
			requireAsyncMessageMatchesPayload(t, spec, "/")
		})
	}
}

func TestAsyncConsumerCombinedPolicies(t *testing.T) {
	fixture := func() {
		inner := dsl.Type("Inner", dsl.Int, func() {
			dsl.Enum(1, 2)
		})
		outer := dsl.Type("Outer", inner, func() {
			dsl.Enum(2)
		})
		result := dsl.ResultType("CombinedPolicyResult", func() {
			dsl.Attributes(func() {
				dsl.Attribute("value", outer, func() {
					dsl.Nullable()
				})
				dsl.Attribute("extra", dsl.String)
			})
			dsl.View("tiny", func() {
				dsl.Attribute("value")
			})
		})
		dsl.Service("combined-policy", func() {
			dsl.Method("watch", func() {
				dsl.StreamingResult(result, func() {
					dsl.View("tiny")
				})
				dsl.HTTP(func() {
					dsl.GET("/")
					dsl.ServerSentEvents()
				})
			})
		})
	}
	spec := asyncAnnotationDocument(t, fixture, true)
	schema := asyncAnnotationMessage(t, spec, "/")
	properties := asyncAnnotationMap(t, schema, "properties")
	require.NotContains(t, properties, "extra", "the selected tiny view owns the event members")
	require.NotContains(t, asyncAnnotationMap(t, schema, "example"), "extra", "examples sample only view members")
	value := asyncAnnotationMap(t, properties, "value")
	require.True(t, asyncNullable(value), "the occurrence nullability is retained")
	refined := asyncNonNull(t, value)
	require.Equal(t, "integer", asyncConjunctionType(t, refined))
	require.Equal(t, []any{float64(2)}, asyncEffectiveEnum(t, refined))
	requireAsyncMessageMatchesPayload(t, spec, "/")
}

// requireAsyncMessageMatchesPayload checks that the derived outbound message
// documents the same completed event as the SSE payload: equal root examples,
// description and member set.
func requireAsyncMessageMatchesPayload(t *testing.T, spec map[string]any, path string) {
	t.Helper()
	message := asyncAnnotationMessage(t, spec, path)
	payload := spec
	for _, key := range []string{"paths", path, "get", "responses", "200", "content", "text/event-stream", "itemSchema", "properties", "data", "contentSchema"} {
		payload = asyncAnnotationMap(t, payload, key)
	}
	resolved := payload
	if ref, ok := payload["$ref"].(string); ok {
		resolved = asyncAnnotationMap(t, asyncAnnotationMap(t, asyncAnnotationMap(t, spec, "components"), "schemas"), strings.TrimPrefix(ref, "#/components/schemas/"))
	}
	example := payload["example"]
	if example == nil {
		example = resolved["example"]
	}
	require.Equal(t, example, message["example"], "message and payload share one example outcome")
	require.Equal(t, resolved["description"], message["description"])
	require.ElementsMatch(t, slices.Collect(maps.Keys(asyncAnnotationMap(t, resolved, "properties"))), slices.Collect(maps.Keys(asyncAnnotationMap(t, message, "properties"))))
}

// asyncConjunction returns schema and every schema it conjoins through allOf.
func asyncConjunction(t *testing.T, schema map[string]any) []map[string]any {
	t.Helper()
	result := []map[string]any{schema}
	if members, ok := schema["allOf"].([]any); ok {
		for _, member := range members {
			result = append(result, asyncConjunction(t, member.(map[string]any))...)
		}
	}
	return result
}

// asyncEffectiveEnum intersects every enum asserted by the conjunction.
func asyncEffectiveEnum(t *testing.T, schema map[string]any) []any {
	t.Helper()
	var effective []any
	for _, layer := range asyncConjunction(t, schema) {
		values, ok := layer["enum"].([]any)
		if !ok {
			continue
		}
		if effective == nil {
			effective = values
			continue
		}
		effective = slices.DeleteFunc(slices.Clone(effective), func(value any) bool {
			return !slices.Contains(values, value)
		})
	}
	return effective
}

func asyncConjunctionType(t *testing.T, schema map[string]any) string {
	t.Helper()
	for _, layer := range asyncConjunction(t, schema) {
		if kind, ok := layer["type"].(string); ok {
			return kind
		}
	}
	return ""
}

func asyncNullable(schema map[string]any) bool {
	alternatives, _ := schema["anyOf"].([]any)
	for _, alternative := range alternatives {
		if arm, ok := alternative.(map[string]any); ok && arm["type"] == "null" {
			return true
		}
	}
	return false
}

func asyncNonNull(t *testing.T, schema map[string]any) map[string]any {
	t.Helper()
	alternatives, _ := schema["anyOf"].([]any)
	for _, alternative := range alternatives {
		if arm, ok := alternative.(map[string]any); ok && arm["type"] != "null" {
			return arm
		}
	}
	t.Fatalf("schema has no non-null alternative: %v", schema)
	return nil
}
