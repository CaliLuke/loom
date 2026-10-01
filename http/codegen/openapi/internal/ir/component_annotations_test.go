package ir

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestComponentAnnotationsSurviveRepresentationRegistration(t *testing.T) {
	for _, omitted := range []bool{false, true} {
		t.Run(map[bool]string{false: "present", true: "omitted"}[omitted], func(t *testing.T) {
			root := codegen.RunDSL(t, func() {
				data := dsl.Type("Data", func() {
					dsl.Attribute("bytes", dsl.Bytes)
					dsl.Required("bytes")
				})
				dsl.Service("svc", func() {
					dsl.Method("send", func() {
						dsl.Payload(data, func() {
							dsl.Example(map[string]any{"bytes": "hi"})
						})
						dsl.Result(data)
						dsl.HTTP(func() {
							dsl.POST("/send")
						})
					})
				})
			})
			service, err := representation.PrepareService(root.API.HTTP.Services[0], nil)
			require.NoError(t, err)
			calls := 0
			a := NewAnalyzer(root.API.ExampleGenerator, false, WithExampleValue(func(attribute *expr.AttributeExpr, value any) (any, bool) {
				calls++
				if omitted {
					return nil, false
				}
				return OpenAPIExampleValue(attribute, value)
			}))
			analyzeComponentTypes(a, root.Types, root.ResultTypes)
			before := a.schemas["Data"].Example
			initialCalls := calls
			analyzeEndpointBodies(a, service.Endpoints[0])
			a.finalizeRepresentations()
			require.Equal(t, before, a.schemas["Data"].Example, "a representation must preserve the existing component annotation authority")
			require.Equal(t, initialCalls, calls, "neither present nor omitted component annotations may be resampled")
			require.Equal(t, "base64", a.schemas["Data"].Properties["bytes"].ContentEncoding)
		})
	}
}

func TestComponentAnnotationAuthorityIsolation(t *testing.T) {
	attr := &expr.AttributeExpr{Type: expr.String, UserExamples: []*expr.ExampleExpr{{Value: "source"}}}
	calls := 0
	a := NewAnalyzer(nil, false, WithExampleValue(func(_ *expr.AttributeExpr, _ any) (any, bool) {
		calls++
		return map[string]any{"items": []string{"owned"}}, true
	}))
	apply := func(declaration, baseline, ownerContext, childContext string) *Schema {
		restore := a.componentAnnotationScope(declaration, baseline, ownerContext)
		defer restore()
		schema := &Schema{}
		a.applySchemaExample(schema, attr, childContext)
		return schema
	}
	first := apply("First", "shape", "component", "field")
	first.Example.(map[string]any)["items"].([]string)[0] = "mutated"
	second := apply("First", "shape", "component", "field")
	require.Equal(t, []string{"owned"}, second.Example.(map[string]any)["items"])
	require.Equal(t, 1, calls)
	second.Example.(map[string]any)["items"].([]string)[0] = "again"
	require.Equal(t, []string{"owned"}, apply("First", "shape", "component", "field").Example.(map[string]any)["items"])
	for _, identity := range [][4]string{
		{"Second", "shape", "component", "field"},
		{"First", "other-shape", "component", "field"},
		{"First", "shape", "other-component", "field"},
		{"First", "shape", "component", "other-field"},
	} {
		apply(identity[0], identity[1], identity[2], identity[3])
	}
	require.Equal(t, 5, calls, "logical declarations, structural baselines and contexts remain independent")
	for range 2 {
		a.applySchemaExample(&Schema{}, attr, "field")
	}
	require.Equal(t, 7, calls, "direct occurrence annotations do not enter the component memo")
}

func TestComponentAnnotationCachesSuppression(t *testing.T) {
	checks := 0
	a := NewAnalyzer(nil, false, WithExampleSuppression(func(_ *expr.AttributeExpr, _ bool) bool {
		checks++
		return true
	}))
	attr := &expr.AttributeExpr{Type: expr.String}
	for range 2 {
		restore := a.componentAnnotationScope("Declaration", "baseline", "component")
		schema := &Schema{}
		a.applySchemaExample(schema, attr, "field")
		restore()
		require.Nil(t, schema.Example)
	}
	require.Equal(t, 1, checks, "an absent annotation is a completed decision")
}

func TestComponentAnnotationStreamingDeclarationAuthority(t *testing.T) {
	root := codegen.RunDSL(t, testdata.TypeIdentityDSL)
	document, err := BuildDocument(root.API, root.Types, root.ResultTypes)
	require.NoError(t, err)
	contract := document.Paths["/stream"].Operations["GET"].Extensions[asyncContractExtensionName].(map[string]any)
	inbound := contract["messages"].(map[string]any)["inbound"].(map[string]any)
	schema := materializationDecode(t, inbound["schema"])
	require.NotEmpty(t, schema.Ref)
	name, ok := schemaComponentName(schema.Ref)
	require.True(t, ok)
	actual := materializeAsyncSchema(&asyncSchema{schema: document.Components.Schemas[name]}, document.Components.Schemas).schema
	expected := materializeAsyncSchema(&asyncSchema{schema: document.Components.Schemas["Other"]}, document.Components.Schemas).schema
	require.Equal(t, expected, actual, "controlled streaming wrappers retain the original declaration's component and sample context")
	// The parent IR already allocates this ordinary streaming-body component.
	// The async retained cut still binds Other; public pruning removes unused
	// components later. Preserve each allocation's independent sample context.
	require.Equal(t, "Other", name)
	allocated := document.Components.Schemas["Other_e59b0fbb558592ed"]
	require.NotNil(t, allocated)
	require.NotNil(t, allocated.Example, "the retained streaming declaration owns its representative")
	require.NotNil(t, allocated.Properties["count"].Example,
		"the retained child occurrence owns its representative independently")
	require.Equal(t, toRef("Other_e59b0fbb558592ed"), document.Components.Schemas["OtherStreamingBody"].Ref)
}

func TestComponentAnnotationControlledWrapperReuse(t *testing.T) {
	for _, kind := range []expr.Primitive{expr.Int, expr.Bytes} {
		for _, aliases := range []int{0, 1, 2} {
			t.Run(fmt.Sprintf("%s/aliases-%d", kind.Name(), aliases), func(t *testing.T) {
				root := codegen.RunDSL(t, func() {
					source := dsl.Type("SampleData", func() {
						dsl.Attribute("data", kind)
						dsl.Required("data")
					})
					for index := range aliases {
						source = dsl.Type(fmt.Sprintf("SampleAlias%d", index), source)
					}
					dsl.Service("samples", func() {
						dsl.Method("ordinary", func() {
							dsl.Payload(source)
							dsl.Result(source)
							dsl.HTTP(func() {
								dsl.POST("/ordinary")
							})
						})
						dsl.Method("stream", func() {
							dsl.StreamingPayload(source)
							dsl.StreamingResult(source)
							dsl.HTTP(func() {
								dsl.GET("/stream")
							})
						})
					})
				})
				prepared, err := representation.PrepareService(root.API.HTTP.Services[0], nil)
				require.NoError(t, err)
				calls := 0
				analyzer := NewAnalyzer(root.API.ExampleGenerator, false, WithExampleValue(func(attribute *expr.AttributeExpr, raw any) (any, bool) {
					calls++
					return OpenAPIExampleValue(attribute, raw)
				}))
				analyzeComponentTypes(analyzer, root.Types, root.ResultTypes)
				initialCalls := calls
				require.Positive(t, initialCalls)
				originalSchema := analyzer.schemas["SampleData"]
				original := copyComponentAnnotation(originalSchema.Example)
				originalChild := copyComponentAnnotation(analyzer.schemas["SampleData"].Properties["data"].Example)
				originalOwners := make(map[componentAnnotationIdentity]map[componentAnnotationPosition]any)
				for identity, positions := range analyzer.componentAnnotations {
					originalOwners[identity] = make(map[componentAnnotationPosition]any)
					for position, value := range positions {
						originalOwners[identity][position] = copyComponentAnnotation(value)
					}
				}
				for _, endpoint := range prepared.Endpoints {
					analyzeEndpointBodies(analyzer, endpoint)
					analyzeAsyncSchemas(analyzer, endpoint)
				}
				firstCalls := calls
				for _, endpoint := range prepared.Endpoints {
					analyzeEndpointBodies(analyzer, endpoint)
					analyzeAsyncSchemas(analyzer, endpoint)
				}
				require.Equal(t, firstCalls, calls, "the same authority and context never invokes callbacks twice")
				analyzer.finalizeRepresentations()
				for identity, positions := range originalOwners {
					for position, value := range positions {
						require.Equal(t, value, analyzer.componentAnnotations[identity][position], "every original declaration/context position remains unchanged")
					}
				}
				total := 0
				for _, positions := range analyzer.componentAnnotations {
					total += len(positions)
				}
				require.Equal(t, total, calls, "each occupied sample position invokes its callback exactly once")
				require.Equal(t, original, originalSchema.Example)
				require.Equal(t, originalChild, originalSchema.Properties["data"].Example)
			})
		}
	}
}
