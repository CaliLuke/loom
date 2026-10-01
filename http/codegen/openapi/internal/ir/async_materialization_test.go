package ir

import (
	"encoding/json/v2"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/CaliLuke/loom/http/codegen/openapi"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

// A repeated declaration hash is a legacy source-inlining cut even when the
// resulting component graph is acyclic. Constraint analysis must still see the
// complete graph; only late materialization and example sampling honor the cut.
func TestAsyncMaterializationDeclaredCuts(t *testing.T) {
	for _, shape := range []string{"root", "property", "array", "map", "untagged union"} {
		for _, cut := range []bool{false, true} {
			name := shape + "/inline"
			if cut {
				name = shape + "/reference"
			}
			t.Run(name, func(t *testing.T) {
				inner := &expr.UserTypeExpr{TypeName: "MaterializedMessage", AttributeExpr: materializationByteObject()}
				attribute := &expr.AttributeExpr{Type: &expr.UserTypeExpr{
					TypeName: "MessageAlias", AttributeExpr: &expr.AttributeExpr{Type: inner},
				}}
				attribute = materializationContainer(shape, attribute)
				calls := 0
				analyzer := materializationAnalyzer(&calls)
				endpoint := inlineAsyncTestEndpoint(t, attribute)
				sampler := asyncSamplerAttribute(attribute)
				if cut {
					// Retained named nodes are the sampler's reference-cut input.
					// Analyze the intact representation independently of this policy.
					sampler = materializationContainer(shape, &expr.AttributeExpr{Type: inner})
				}
				target := representation.PrepareStreamSchema(endpoint, attribute, false)
				prepared := analyzer.acquireAsyncBaseline(attribute, representationRoot(attribute, target), sampler, "materialization")
				require.Zero(t, calls)
				analyzer.finalizeRepresentations()
				result := materializationMessage(t, analyzer, endpoint, prepared)
				targetSchema := materializationChild(t, shape, result)
				if cut {
					wantCalls := 1
					if shape == "root" {
						wantCalls = 0
					}
					require.Equal(t, wantCalls, calls, "reference cuts must not sample or materialize component examples")
					require.NotEmpty(t, targetSchema.Ref)
					componentName, ok := schemaComponentName(targetSchema.Ref)
					require.True(t, ok)
					component := materializeAsyncSchema(&asyncSchema{schema: analyzer.schemas[componentName]}, analyzer.schemas).schema
					materializationByteBounds(t, RenderSchema(component.Properties["data"]))
				} else {
					wantCalls := 3
					if shape == "root" {
						wantCalls = 2
					}
					require.Equal(t, wantCalls, calls)
					require.Empty(t, targetSchema.Ref)
					materializationByteBounds(t, targetSchema.Properties["data"])
				}
			})
		}
	}
}

func TestAsyncMaterializationFinalizedDeclaredCut(t *testing.T) {
	root := codegen.RunDSL(t, testdata.TypeIdentityDSL)
	document, err := BuildDocument(root.API, root.Types, root.ResultTypes)
	require.NoError(t, err)
	contract := document.Paths["/stream"].Operations["GET"].Extensions[asyncContractExtensionName].(map[string]any)
	inbound := contract["messages"].(map[string]any)["inbound"].(map[string]any)
	schema := materializationDecode(t, inbound["schema"])
	require.NotEmpty(t, schema.Ref)
	example, ok := schema.Example.(map[string]any)
	require.True(t, ok, "the retained target-plan position owns the referenced schema example")
	require.Contains(t, example, "count")
	name, ok := schemaComponentName(schema.Ref)
	require.True(t, ok)
	require.Contains(t, document.Components.Schemas, name)
}

func TestAsyncMaterializationTaggedEnvelopes(t *testing.T) {
	attribute := &expr.AttributeExpr{Type: &expr.Union{TypeName: "Message", Values: []*expr.NamedAttributeExpr{
		{Name: "bytes", Attribute: materializationByteObject()},
		{Name: "text", Attribute: &expr.AttributeExpr{Type: expr.String}},
	}}}
	calls := 0
	analyzer := materializationAnalyzer(&calls)
	endpoint := inlineAsyncTestEndpoint(t, attribute)
	prepared := analyzeAsyncSchema(analyzer, attribute, endpoint, false, "materialization")
	require.Zero(t, calls)
	analyzer.finalizeRepresentations()
	result := materializationMessage(t, analyzer, endpoint, prepared)
	require.Equal(t, 1, calls, "the union is sampled once; synthetic envelopes keep component authority")
	require.Len(t, result.OneOf, 2)
	for _, arm := range result.OneOf {
		require.NotEmpty(t, arm.Ref)
		name, ok := schemaComponentName(arm.Ref)
		require.True(t, ok)
		envelope := analyzer.schemas[name]
		tag := envelope.Properties["type"].Enum[0].(string)
		require.Equal(t, arm.Ref, result.Discriminator.Mapping[tag])
		if tag == "bytes" {
			materializationByteBounds(t, RenderSchema(envelope.Properties["value"].Properties["data"]))
		}
	}
}

func TestAsyncMaterializationSSEEnvelopeReferences(t *testing.T) {
	root := codegen.RunDSL(t, testdata.SSEVariantProjectionDSL)
	document, err := BuildDocument(root.API, root.Types, root.ResultTypes)
	require.NoError(t, err)
	contract := document.Paths["/events"].Operations["GET"].Extensions[asyncContractExtensionName].(map[string]any)
	outbound := contract["messages"].(map[string]any)["outbound"].(map[string]any)
	schema := outbound["schema"].(*openapi.Schema)
	require.Len(t, schema.OneOf, 2)
	event := schema.OneOf[1].Properties["event"]
	require.Len(t, event.OneOf, 2)
	for _, arm := range event.OneOf {
		require.NotEmpty(t, arm.Ref)
		name, ok := schemaComponentName(arm.Ref)
		require.True(t, ok)
		require.Contains(t, document.Components.Schemas, name)
	}
}

func TestAsyncMaterializationReferenceSiblings(t *testing.T) {
	for _, assertion := range []bool{false, true} {
		name := "annotations"
		if assertion {
			name = "assertions"
		}
		t.Run(name, func(t *testing.T) {
			calls := 0
			analyzer := materializationAnalyzer(&calls)
			attribute := materializationByteObject()
			object := attribute.Type.(*expr.Object)
			*object = append(*object, &expr.NamedAttributeExpr{Name: "count", Attribute: &expr.AttributeExpr{Type: expr.Int}})
			endpoint := inlineAsyncTestEndpoint(t, attribute)
			prepared := analyzeAsyncSchema(analyzer, attribute, endpoint, false, "materialization")
			require.Zero(t, calls)
			analyzer.finalizeRepresentations()
			// This is the exact post-analysis vocabulary: a component reference
			// with local annotations, optionally with an active assertion sibling.
			component := prepared.schema
			analyzer.schemas["AnnotationBase"] = component
			prepared.schema = &Schema{Ref: toRef("AnnotationBase"), Description: "local description", Example: map[string]any{"data": "aGk="}}
			if assertion {
				prepared.schema.Not = &Schema{Required: []string{"forbidden"}}
			}
			result := materializationMessage(t, analyzer, endpoint, prepared)
			require.Equal(t, "local description", result.Description)
			if assertion {
				require.Equal(t, 3, calls, "assertion wrappers must preserve child sampling correspondence")
				require.NotNil(t, result.Not, "active siblings must remain conjoined")
				require.Equal(t, []string{"forbidden"}, result.Not.Required)
				require.Len(t, result.AllOf, 1)
				require.Equal(t, "base64", result.AllOf[0].Properties["data"].ContentEncoding)
			} else {
				require.Equal(t, 3, calls, "annotation-only overlays must not hide child sampling paths")
				require.Empty(t, result.AllOf)
				materializationByteBounds(t, result.Properties["data"])
				require.Equal(t, "aGk=", result.Properties["data"].Example)
				legacyCalls := 0
				legacy := materializationAnalyzer(&legacyCalls).AnalyzeSchemaWithContext(attribute, "materialization")
				require.Equal(t, 3, legacyCalls)
				require.Equal(t, materializationDecode(t, RenderSchema(legacy)).Properties["count"].Example, result.Properties["count"].Example, "inline occurrence seed must not become a component seed")
			}
			require.Empty(t, component.Description, "materialization must not mutate registered components")
			require.Nil(t, component.Example)
		})
	}
}

func TestAsyncMaterializationCutKeepsAssertionSiblings(t *testing.T) {
	calls := 0
	analyzer := materializationAnalyzer(&calls)
	attribute := &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "CutAssertions", AttributeExpr: materializationByteObject()}}
	endpoint := inlineAsyncTestEndpoint(t, attribute)
	target := representation.PrepareStreamSchema(endpoint, attribute, false)
	prepared := analyzer.acquireAsyncBaseline(attribute, representationRoot(attribute, target), attribute, "materialization")
	analyzer.finalizeRepresentations()
	require.NotEmpty(t, prepared.schema.Ref)
	prepared.schema.Not = &Schema{Required: []string{"forbidden"}}
	prepared.schema.Description = "An assertion-bearing reference."
	result := materializationMessage(t, analyzer, endpoint, prepared)
	require.Zero(t, calls)
	require.Equal(t, prepared.schema.Ref, result.Ref)
	require.Equal(t, prepared.schema.Description, result.Description)
	require.NotNil(t, result.Not, "a retained reference must not discard its active siblings")
	require.Equal(t, []string{"forbidden"}, result.Not.Required)
	name, ok := schemaComponentName(result.Ref)
	require.True(t, ok)
	materializationByteBounds(t, RenderSchema(analyzer.schemas[name].Properties["data"]))
}

func materializationByteBounds(t *testing.T, schema *openapi.Schema) {
	t.Helper()
	require.Equal(t, "base64", schema.ContentEncoding)
	require.NotNil(t, schema.Not)
	require.Equal(t, "[^A-Za-z0-9+/=]", schema.Not.Pattern)
	require.Len(t, schema.AnyOf, 3)
	for index, maximum := range []int{0, 4, 4} {
		require.NotNil(t, schema.AnyOf[index].MaxLength)
		require.Equal(t, maximum, *schema.AnyOf[index].MaxLength)
	}
}

func materializationAnalyzer(calls *int) *Analyzer {
	return NewAnalyzer(expr.NewRandom("materialization"), false, WithExampleValue(func(_ *expr.AttributeExpr, raw any) (any, bool) {
		*calls++
		return raw, true
	}))
}

func materializationByteObject() *expr.AttributeExpr {
	maximum := 2
	return &expr.AttributeExpr{Type: &expr.Object{
		{Name: "data", Attribute: &expr.AttributeExpr{
			Type: expr.Bytes, Validation: &expr.ValidationExpr{MaxLength: &maximum},
			UserExamples: []*expr.ExampleExpr{{Value: []byte("hi")}},
		}},
	}, UserExamples: []*expr.ExampleExpr{{Value: map[string]any{"data": []byte("hi")}}}}
}

func materializationContainer(shape string, child *expr.AttributeExpr) *expr.AttributeExpr {
	switch shape {
	case "property":
		return &expr.AttributeExpr{Type: &expr.Object{{Name: "child", Attribute: child}}}
	case "array":
		return &expr.AttributeExpr{Type: &expr.Array{ElemType: child}}
	case "map":
		return &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: child}}
	case "untagged union":
		return &expr.AttributeExpr{Type: &expr.Union{TypeName: "Choice", Untagged: true, Values: []*expr.NamedAttributeExpr{{Name: "child", Attribute: child}}}}
	default:
		return child
	}
}

func materializationChild(t *testing.T, shape string, schema *openapi.Schema) *openapi.Schema {
	t.Helper()
	switch shape {
	case "property":
		return schema.Properties["child"]
	case "array":
		return schema.Items
	case "map":
		child, ok := schema.AdditionalProperties.(map[string]any)
		require.True(t, ok)
		return materializationDecode(t, child)
	case "untagged union":
		require.Len(t, schema.OneOf, 1)
		return schema.OneOf[0]
	default:
		return schema
	}
}

func materializationMessage(t *testing.T, analyzer *Analyzer, endpoint *transportir.Endpoint, prepared *asyncSchema) *openapi.Schema {
	t.Helper()
	bodies := &EndpointBodies{
		async:           map[string]map[string]*asyncSchema{"/probe": {"outbound": prepared}},
		asyncComponents: analyzer.schemas, asyncAnalyzer: analyzer,
	}
	message := buildAsyncMessages(endpoint, "/probe", bodies)["outbound"].(map[string]any)
	return materializationDecode(t, message["schema"])
}

func materializationDecode(t *testing.T, value any) *openapi.Schema {
	t.Helper()
	encoded, err := json.Marshal(value)
	require.NoError(t, err)
	var result openapi.Schema
	require.NoError(t, json.Unmarshal(encoded, &result))
	return &result
}

func TestAsyncMaterializationRequiredOverlayKeepsChildSampling(t *testing.T) {
	for _, nullable := range []bool{false, true} {
		name := "concrete"
		if nullable {
			name = "nullable"
		}
		t.Run(name, func(t *testing.T) {
			inner := &expr.UserTypeExpr{TypeName: "WithRequiredOverlay", AttributeExpr: materializationByteObject()}
			attribute := &expr.AttributeExpr{Type: inner, Validation: &expr.ValidationExpr{Required: []string{"data"}}, Nullable: nullable}
			calls := 0
			analyzer := materializationAnalyzer(&calls)
			endpoint := inlineAsyncTestEndpoint(t, attribute)
			prepared := analyzeAsyncSchema(analyzer, attribute, endpoint, false, "materialization")
			require.Zero(t, calls)
			legacyCalls := 0
			legacy := materializationAnalyzer(&legacyCalls).AnalyzeSchemaWithContext(prepared.sampler, "materialization")
			require.Equal(t, 2, legacyCalls)
			require.NotNil(t, legacy.Properties["data"].Example)
			analyzer.finalizeRepresentations()
			result := materializationMessage(t, analyzer, endpoint, prepared)
			require.Equal(t, 2, calls, "inline object and child each retain their existing example callback")
			require.Empty(t, result.AllOf, "the parent inline owner merges occurrence validation before construction")
			require.Empty(t, result.AnyOf, "the parent named inline occurrence does not add an outer null branch")
			require.Equal(t, []string{"data"}, result.Required)
			require.NotNil(t, result.Properties["data"].Example, "required fields keep their inline child example")
		})
	}
}

func TestAsyncMaterializationWrapperComposition(t *testing.T) {
	for _, shape := range []string{"property", "array", "map", "untagged union"} {
		for _, nullability := range []string{"concrete", "nullable"} {
			for _, overlay := range []string{"annotations", "assertions"} {
				for _, order := range []string{"outer", "inner"} {
					t.Run(shape+"/"+nullability+"/"+overlay+"/"+order, func(t *testing.T) {
						checkAsyncWrapperComposition(t, shape, nullability == "nullable", overlay == "assertions", order == "outer")
					})
				}
			}
		}
	}
}

func checkAsyncWrapperComposition(t *testing.T, shape string, nullable, assertion, outer bool) {
	t.Helper()
	attribute := materializationContainer(shape, materializationByteObject())
	attribute.Nullable = nullable
	calls := 0
	analyzer := materializationAnalyzer(&calls)
	endpoint := inlineAsyncTestEndpoint(t, attribute)
	prepared := analyzeAsyncSchema(analyzer, attribute, endpoint, false, "composition")
	require.Zero(t, calls)
	analyzer.finalizeRepresentations()
	root := &prepared.schema
	if nullable && !outer {
		require.True(t, asyncNullableWrapper(*root))
		root = &(*root).AnyOf[0]
	}
	analyzer.schemas["ComposedBase"] = *root
	*root = &Schema{Ref: toRef("ComposedBase"), Description: "Occurrence annotation"}
	if assertion {
		(*root).Not = &Schema{Required: []string{"forbidden"}}
	}
	materialized := materializeAsyncSchema(prepared, analyzer.schemas)
	applyPreparedAsyncExamples(analyzer, materialized.schema, prepared, materialized.structures)
	require.Equal(t, 3, calls, "container, child object and byte field each have one callback")
	structural := materializationConcrete(t, materialized.schema)
	var child *Schema
	switch shape {
	case "property":
		child = structural.Properties["child"]
	case "array":
		child = structural.Items
	case "map":
		child = structural.AdditionalProperties.Schema
	case "untagged union":
		child = structural.OneOf[0]
	}
	require.NotNil(t, child)
	require.NotNil(t, child.Example)
	require.NotNil(t, child.Properties["data"].Example)
	materializationByteBounds(t, RenderSchema(child.Properties["data"]))
	// The wrapper is retained independently from the structure used for samples.
	wrapper := materialized.schema
	if nullable && !outer {
		wrapper = wrapper.AnyOf[0]
	}
	require.Equal(t, "Occurrence annotation", wrapper.Description)
	if assertion {
		require.Equal(t, []string{"forbidden"}, wrapper.Not.Required)
	}
	if nullable {
		encoded, err := json.Marshal(RenderSchema(materialized.schema))
		require.NoError(t, err)
		require.Contains(t, string(encoded), `"type":"null"`)
	}
}

// This test oracle follows the explicitly built single-assertion and nullable
// wrappers, independently of the materializer's pointer correspondence map.
func materializationConcrete(t *testing.T, schema *Schema) *Schema {
	t.Helper()
	for range 3 {
		if len(schema.AllOf) == 1 {
			schema = schema.AllOf[0]
			continue
		}
		if len(schema.AnyOf) == 2 && schema.AnyOf[1].Type == "null" {
			schema = schema.AnyOf[0]
			continue
		}
		return schema
	}
	t.Fatal("unexpected wrapper depth")
	return nil
}

func TestAsyncMaterializationDoesNotAssignArbitraryAlternatives(t *testing.T) {
	for _, schema := range []*Schema{
		{AnyOf: []*Schema{{Type: "object"}, {Type: "string"}}},
		{AllOf: []*Schema{{Required: []string{"one"}}, {Required: []string{"two"}}}},
		{AnyOf: []*Schema{{Type: "object"}, {Type: "null", Ref: toRef("ConstrainedNull")}}},
	} {
		result := materializeAsyncSchema(&asyncSchema{schema: schema, sampler: materializationByteObject()}, nil)
		require.Empty(t, result.structures, "arbitrary alternatives and assertions have no inferred source-child authority")
	}
}
