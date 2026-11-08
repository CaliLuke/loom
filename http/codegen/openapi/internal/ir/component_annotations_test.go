package ir

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
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
			require.Len(t, root.Services[0].Methods[0].Payload.ExtractUserExamples(), 1, "one authored callback supplies one example")
			service, err := representation.PrepareService(root.API.HTTP.Services[0], nil, nil)
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
			original := a.schemas["Data"]
			before := copyComponentAnnotation(original.Example)
			initialOwners := snapshotAnnotationOwners(a)
			bodies := analyzeEndpointBodies(a, service.Endpoints[0])
			acquiredOwners := snapshotAnnotationOwners(a)
			firstCalls := calls
			analyzeEndpointBodies(a, service.Endpoints[0])
			require.Equal(t, firstCalls, calls, "exact acquired positions must not invoke the projector twice")
			a.finalizeRepresentations()
			// Final public-name allocation may choose another complete variant;
			// the acquired original schema and its annotation owner stay intact.
			require.Equal(t, before, original.Example, "a representation must preserve the acquired component annotation authority")
			require.Equal(t, firstCalls, calls, "final allocation must not invoke the projector")
			require.Equal(t, acquiredOwners, snapshotAnnotationOwners(a), "all acquired annotation positions remain unchanged")
			for owner, positions := range initialOwners {
				require.Equal(t, positions, snapshotAnnotationOwners(a)[owner], "the original standalone owner is preserved")
			}
			request := annotationObject(t, bodies.RequestBody, a.schemas)
			if omitted {
				require.Nil(t, request.Example)
			} else {
				require.Equal(t, map[string]any{"bytes": "aGk="}, request.Example, "the authored request value has its own annotation authority")
				require.Empty(t, request.Examples, "one authored example uses the singular annotation")
			}
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
	ordinaryRef, acquired := streamingAnnotationBinding(t, root)
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
	require.Equal(t, "Other", name)
	// Follow the actual ordinary body's acquired reference through final naming.
	// It has its own prepared root/child samples; its private suffix is not a contract.
	published := annotationObject(t, &Schema{Ref: ordinaryRef}, document.Components.Schemas)
	require.Equal(t, acquired, published, "the exact ordinary binding remains reachable through its renamed wrapper")
	require.NotNil(t, published.Example, "the retained streaming declaration owns its representative")
	require.NotNil(t, published.Properties["count"].Example,
		"the retained child occurrence owns its representative independently")
	t.Logf("async declaration %s; ordinary acquired body %s", schema.Ref, ordinaryRef)
}

func streamingAnnotationBinding(t *testing.T, root *expr.RootExpr) (string, *Schema) {
	t.Helper()
	a := NewAnalyzer(root.API.ExampleGenerator, false)
	a.initializeDiscriminatorHints(root.API, root.Types, root.ResultTypes)
	analyzeComponentTypes(a, root.Types, root.ResultTypes)
	a.recordDiscriminatorChildren(root.Types, root.ResultTypes)
	bodies := &BodyTypes{
		Services:  make(map[string]map[string]*EndpointBodies),
		prepared:  make(map[string]*transportir.Service),
		locations: make(map[*expr.AttributeExpr]*Schema),
	}
	analyzeServiceBodies(a, root.API, bodies)
	require.NoError(t, a.Err())
	stream := bodies.Services["streamcollision"]["other"]
	require.NotNil(t, stream)
	binding := stream.RequestBody
	require.NotNil(t, binding)
	owned := annotationObject(t, binding, a.schemas)
	var localName string
	for name, component := range a.schemas {
		if component == owned {
			localName = name
			break
		}
	}
	require.NotEmpty(t, localName)
	naming := a.schemaNames[localName]
	var owner componentAnnotationIdentity
	matches := 0
	for identity := range a.componentAnnotations {
		if identity.declaration == naming.logical && identity.baseline == naming.baseline && identity.context == exampleContext("component", naming.annotationName) {
			owner = identity
			matches++
		}
	}
	require.Equal(t, 1, matches, "the actual acquired component identifies one annotation owner")
	objectNode := owner.examplePlan.Underlying()
	target := stream.prepared.Request.StreamingValue
	require.NotNil(t, target)
	rootSet := target.ExampleSets[objectNode]
	require.NotNil(t, rootSet, "the owner must refer to this exact prepared streaming target")
	require.NotNil(t, rootSet.Representative)
	rootValue, ok := preparedOpenAPIExampleValue(*rootSet.Representative)
	require.True(t, ok)
	require.Equal(t, rootValue, owned.Example)
	var countNode expr.ValuePlanNode
	for _, member := range objectNode.Members() {
		if member.WireName == "count" {
			countNode = member.Node
		}
	}
	require.True(t, countNode.Valid())
	childSet := target.ExampleSets[countNode]
	require.NotNil(t, childSet)
	require.NotNil(t, childSet.Representative)
	childValue, ok := preparedOpenAPIExampleValue(*childSet.Representative)
	require.True(t, ok)
	require.Equal(t, childValue, owned.Properties["count"].Example)
	beforeRoot, beforeChild := copyComponentAnnotation(owned.Example), copyComponentAnnotation(owned.Properties["count"].Example)
	for _, service := range orderedStringKeys(bodies.Services) {
		for _, method := range orderedStringKeys(bodies.Services[service]) {
			analyzeAsyncSchemas(a, bodies.Services[service][method].prepared, bodies.Services[service][method].event)
		}
	}
	owners := snapshotAnnotationOwners(a)
	a.finalizeRepresentations()
	require.NoError(t, a.Err())
	require.Equal(t, owners, snapshotAnnotationOwners(a), "final naming cannot replace acquired annotation positions")
	published := annotationObject(t, binding, a.schemas)
	require.Equal(t, beforeRoot, published.Example)
	require.Equal(t, beforeChild, published.Properties["count"].Example)
	return binding.Ref, published
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
				prepared, err := representation.PrepareService(root.API.HTTP.Services[0], nil, nil)
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
				originalOwners := make(map[componentAnnotationIdentity]map[componentAnnotationPosition]componentExamples)
				for identity, positions := range analyzer.componentAnnotations {
					originalOwners[identity] = make(map[componentAnnotationPosition]componentExamples)
					for position, value := range positions {
						originalOwners[identity][position] = value.snapshot()
					}
				}
				for _, endpoint := range prepared.Endpoints {
					analyzeAsyncSchemas(analyzer, endpoint, analyzeEndpointBodies(analyzer, endpoint).event)
				}
				firstCalls := calls
				for _, endpoint := range prepared.Endpoints {
					analyzeAsyncSchemas(analyzer, endpoint, analyzeEndpointBodies(analyzer, endpoint).event)
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

// snapshotAnnotationOwners records detached samples at exact acquired positions.
func snapshotAnnotationOwners(a *Analyzer) map[componentAnnotationIdentity]map[componentAnnotationPosition]componentExamples {
	owners := make(map[componentAnnotationIdentity]map[componentAnnotationPosition]componentExamples, len(a.componentAnnotations))
	for identity, positions := range a.componentAnnotations {
		owner := make(map[componentAnnotationPosition]componentExamples, len(positions))
		for position, samples := range positions {
			owner[position] = samples.snapshot()
		}
		owners[identity] = owner
	}
	return owners
}
