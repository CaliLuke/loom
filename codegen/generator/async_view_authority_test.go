package generator

import (
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

// These assertions preserve the inline consumer shape while using the shared
// alias refinement and occurrence-nullability semantics.
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
			require.NotContains(t, value, "allOf")
			require.NotContains(t, value, "anyOf")
			if policy == "enum refinement" {
				require.Equal(t, []any{float64(2)}, value["enum"])
				require.Equal(t, "integer", value["type"])
			} else {
				require.Equal(t, "string", value["type"])
			}
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
	require.NotContains(t, schema, "description")
	properties := asyncAnnotationMap(t, schema, "properties")
	require.Contains(t, properties, "extra", "inline shape does not inherit the ordinary tiny-view policy")
	value := asyncAnnotationMap(t, properties, "value")
	require.Equal(t, "integer", value["type"])
	require.Equal(t, []any{float64(2)}, value["enum"])
	require.NotContains(t, value, "allOf")
	require.NotContains(t, value, "anyOf")
}
