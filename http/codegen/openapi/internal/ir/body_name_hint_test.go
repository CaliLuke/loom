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

type bodyHintRandomizer struct {
	expr.Randomizer
	calls int
}

func TestInlineBodyCompatibleNameHint(t *testing.T) {
	for _, mode := range []string{"authored", "suppressed", "generated", "custom"} {
		t.Run(mode, func(t *testing.T) {
			root := bodyHintRoot(t, mode == "authored" || mode == "custom", false, true)
			options := []AnalyzerOption{}
			callbacks := 0
			if mode == "suppressed" {
				options = append(options, WithExampleSuppression(func(*expr.AttributeExpr, bool) bool {
					return true
				}))
			}
			if mode == "custom" {
				options = append(options, WithExampleValue(func(attribute *expr.AttributeExpr, value any) (any, bool) {
					callbacks++
					return OpenAPIExampleValue(attribute, value)
				}))
			}
			bodies := BuildBodyTypes(root.API, root.Types, root.ResultTypes, options...)
			require.NoError(t, bodies.Err)
			responses := bodies.Services["Svc"]["do"].ResponseBodies
			explicit := responses["200"][0].Ref
			if mode == "generated" {
				require.NotEqual(t, toRef("RT"), explicit, "method samples do not substitute declaration samples")
				name, ok := schemaComponentName(explicit)
				require.True(t, ok)
				require.NotEqual(t, bodies.Components["RT"].Example, bodies.Components[name].Example)
				implicitRoot := bodyHintRoot(t, false, false, false)
				implicit := BuildBodyTypes(implicitRoot.API, implicitRoot.Types, implicitRoot.ResultTypes)
				require.NoError(t, implicit.Err)
				require.Equal(t, implicit.Services["Svc"]["do"].ResponseBodies["200"][0].Ref, explicit, "separate identical seeded designs preserve full-selection naming")
				require.Equal(t, implicit.Components[name], bodies.Components[name])
			} else {
				require.Equal(t, toRef("RT"), explicit, "equal complete annotated schemas reuse the hinted declaration")
			}
			if mode == "custom" {
				require.Positive(t, callbacks, "the custom projector participates in annotation materialization")
			}
		})
	}
}

func TestInlineBodyHintForeignAndPartial(t *testing.T) {
	for _, mode := range []string{"foreign", "partial"} {
		t.Run(mode, func(t *testing.T) {
			root := codegen.RunDSL(t, func() {
				first := dsl.ResultType("application/vnd.first", "First", func() {
					dsl.Attribute("a", dsl.String)
					dsl.Attribute("b", dsl.String)
				})
				second := dsl.ResultType("application/vnd.second", "Second", func() {
					dsl.Attribute("a", dsl.String)
					dsl.Attribute("b", dsl.String)
				})
				dsl.Service("Svc", func() {
					for index, result := range []expr.UserType{first, second} {
						dsl.Method(fmt.Sprintf("method%d", index), func() {
							dsl.Result(result)
							dsl.HTTP(func() {
								dsl.GET(fmt.Sprintf("/%d", index))
								dsl.Response(dsl.StatusOK, func() {
									dsl.Body(func() {
										dsl.Attribute("a")
										if mode != "partial" {
											dsl.Attribute("b")
										}
									})
								})
							})
						})
					}
				})
			})
			bodies := BuildBodyTypes(root.API, root.Types, root.ResultTypes, WithExampleSuppression(func(*expr.AttributeExpr, bool) bool {
				return true
			}))
			require.NoError(t, bodies.Err)
			first := bodies.Services["Svc"]["method0"].ResponseBodies["200"][0].Ref
			second := bodies.Services["Svc"]["method1"].ResponseBodies["200"][0].Ref
			require.NotEqual(t, first, second, "different authored hints retain independent families")
			if mode == "foreign" {
				require.Equal(t, toRef("First"), first)
				require.Equal(t, toRef("Second"), second)
			} else {
				require.NotEqual(t, toRef("First"), first)
				name, ok := schemaComponentName(first)
				require.True(t, ok)
				require.Len(t, bodies.Components[name].Properties, 1)
			}
		})
	}
}

func TestInlineBodyHintRetainsMethodSample(t *testing.T) {
	root := bodyHintRoot(t, false, true, true)
	random := &bodyHintRandomizer{Randomizer: expr.NewFakerRandomizer("body-hint")}
	generator := &expr.ExampleGenerator{Randomizer: random}
	prepared, err := representation.PrepareService(root.API.HTTP.Services[0], nil, nil)
	require.NoError(t, err)
	require.NoError(t, representation.PrepareServiceExamples(prepared, root.API.HTTP.Services[0], generator))
	endpoint := prepared.Endpoints[0]
	first, second := endpoint.Response.Responses[0].DocumentValue, endpoint.Response.Responses[1].DocumentValue
	require.Same(t, first.Source, second.Source, "both bodies belong to this exact method occurrence")
	require.NoError(t, representation.PrepareEndpointExamples(endpoint, generator))
	require.Same(t, first.Source, first.Representative.Source)
	require.Same(t, second.Source, second.Representative.Source)
	value, present := first.Source.Example.DeclaredJSONValue()
	require.True(t, present)
	retained, ok := first.Source.Example.DeclaredJSONValue()
	require.True(t, ok)
	require.Equal(t, value, retained)
	calls := random.calls
	require.Positive(t, calls)
	require.NoError(t, representation.PrepareServiceExamples(prepared, root.API.HTTP.Services[0], generator))
	require.NoError(t, representation.PrepareEndpointExamples(endpoint, generator))
	a := NewAnalyzer(generator, false)
	bodies := analyzeEndpointBodies(a, endpoint)
	repeated := analyzeEndpointBodies(a, endpoint)
	// The first compatible acquisition owns the component's exact prepared
	// position. A second response still retains its own unused sample records.
	response := endpoint.Response.Responses[0]
	attribute := attributeForSchemaUsage(response.DocumentBody, schemaUsageResponse)
	node := representationRoot(attribute, response.DocumentValue).Underlying()
	set := response.DocumentValue.ExampleSets[node]
	require.NotNil(t, set)
	require.NotNil(t, set.Representative)
	expected, emitted := preparedOpenAPIExampleValue(*set.Representative)
	require.True(t, emitted)
	require.Equal(t, bodies.ResponseBodies["200"][0].Ref, bodies.ResponseBodies["201"][0].Ref)
	name, local := schemaComponentName(bodies.ResponseBodies["200"][0].Ref)
	require.True(t, local)
	require.Equal(t, expected, a.schemas[name].Example)
	a.finalizeRepresentations()
	require.NoError(t, a.Err())
	require.Equal(t, calls, random.calls, "schema registration/finalization must not sample retained values again")
	for _, status := range []string{"200", "201"} {
		name, ok := schemaComponentName(bodies.ResponseBodies[status][0].Ref)
		require.True(t, ok)
		require.Equal(t, expected, a.schemas[name].Example)
		require.Equal(t, bodies.ResponseBodies[status][0].Ref, repeated.ResponseBodies[status][0].Ref)
	}
}

func TestInlineBodyHintCustomCallbacks(t *testing.T) {
	root := bodyHintRoot(t, true, false, true)
	prepared, err := representation.PrepareService(root.API.HTTP.Services[0], nil, nil)
	require.NoError(t, err)
	componentCalls, occurrenceCalls := 0, 0
	var a *Analyzer
	a = NewAnalyzer(root.API.ExampleGenerator, false, WithExampleValue(func(attribute *expr.AttributeExpr, raw any) (any, bool) {
		if a.annotationOwner != nil {
			componentCalls++
		} else {
			occurrenceCalls++
		}
		return OpenAPIExampleValue(attribute, raw)
	}))
	analyzeComponentTypes(a, root.Types, root.ResultTypes)
	first := analyzeEndpointBodies(a, prepared.Endpoints[0])
	components, occurrences := componentCalls, occurrenceCalls
	require.Positive(t, components)
	repeated := analyzeEndpointBodies(a, prepared.Endpoints[0])
	require.Equal(t, components, componentCalls, "reusing acquired component positions does not replay callbacks")
	require.Equal(t, occurrences+1, occurrenceCalls, "direct occurrence annotations remain outside the component memo")
	a.finalizeRepresentations()
	require.Equal(t, components, componentCalls, "complete graph comparison does not call custom projectors")
	require.Equal(t, occurrences+1, occurrenceCalls)
	require.Equal(t, first.ResponseBodies["200"][0].Ref, repeated.ResponseBodies["200"][0].Ref)
	require.Equal(t, toRef("RT"), first.ResponseBodies["200"][0].Ref)
}

// The expected values are the independently acquired component's retained
// samples before compatible-name alignment, not the declaration's samples.
func TestInlineBodyHintCustomGeneratedSampling(t *testing.T) {
	root := bodyHintRoot(t, false, false, true)
	bodies := BuildBodyTypes(root.API, root.Types, root.ResultTypes, WithExampleValue(OpenAPIExampleValue))
	require.NoError(t, bodies.Err)
	ref := bodies.Services["Svc"]["do"].ResponseBodies["200"][0].Ref
	name, local := schemaComponentName(ref)
	require.True(t, local)
	component := bodies.Components[name]
	require.Equal(t, map[string]any{
		"a": "Molestias non voluptas vero molestiae aut quidem.",
		"b": "Vel quia sed dolore ut natus.",
	}, component.Example)
	require.Equal(t, "Ipsam quia ut cupiditate.", component.Properties["a"].Example)
	require.Equal(t, "Voluptas aut vero.", component.Properties["b"].Example)
}

func bodyHintRoot(t *testing.T, authored, paired, explicit bool) *expr.RootExpr {
	t.Helper()
	return codegen.RunDSL(t, func() {
		result := dsl.ResultType("application/vnd.rt", "RT", func() {
			dsl.Attributes(func() {
				for _, name := range []string{"a", "b"} {
					dsl.Attribute(name, dsl.String, func() {
						if authored {
							dsl.Example(name)
						}
					})
				}
				if authored {
					dsl.Example(map[string]any{"a": "a", "b": "b"})
				}
			})
		})
		dsl.Service("Svc", func() {
			dsl.Method("do", func() {
				dsl.Result(result)
				dsl.HTTP(func() {
					dsl.POST("/do")
					dsl.Response(dsl.StatusOK, func() {
						if !explicit {
							return
						}
						dsl.Body(func() {
							dsl.Attribute("a")
							dsl.Attribute("b")
						})
					})
					if paired {
						dsl.Response(dsl.StatusCreated, func() {
							dsl.Tag("a", "created")
						})
					}
				})
			})
		})
	})
}

func (r *bodyHintRandomizer) String() string {
	r.calls++
	return fmt.Sprintf("sample-%d", r.calls)
}
