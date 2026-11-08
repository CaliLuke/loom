package ir

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/http/codegen/openapi"
)

// The derived async branch requires the completed event target, the response
// source and the finalized response target to correspond node by node. Guards
// on bindings and selection alone are not sufficient.
func TestEventCorrespondenceComparesFinalizedTargetGraph(t *testing.T) {
	for _, test := range []struct {
		name     string
		design   func()
		eligible bool
	}{
		{"implicit named event", sseSpecimenDSL(sseSpecimen{}), true},
		{"implicit local example and description", sseSpecimenDSL(sseSpecimen{local: true}), true},
		{"implicit primitive event", sseSpecimenDSL(sseSpecimen{primitive: true}), true},
		{"equal explicit body", sseSpecimenDSL(sseSpecimen{body: "equal"}), true},
		{"explicit body with local example", sseSpecimenDSL(sseSpecimen{body: "local"}), false},
		{"mixed results", sseSpecimenDSL(sseSpecimen{mixed: true}), false},
		{"annotation-distinct nested declarations", correspondenceSiblingsDSL, true},
		{"alias chain with inherited annotation", correspondenceAliasDSL(false), true},
		{"alias chain with authored override", correspondenceAliasDSL(true), true},
		{"alias chain refinement", correspondenceRefinementDSL, true},
		{"nullable occurrence refinement and examples", correspondenceNullableRefinementDSL, true},
		{"selected result view", correspondenceViewDSL, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := codegen.RunDSL(t, test.design)
			bodies := BuildBodyTypes(root.API, root.Types, root.ResultTypes)
			require.NoError(t, bodies.Err)
			endpoint := bodies.Services["events"]["watch"]
			require.NotNil(t, endpoint)
			require.Equal(t, test.eligible, endpoint.event != nil && endpoint.event.eligible)
		})
	}
}

// The consumer contract: an eligible message and its payload document the same
// completed event, including annotations of same-shaped nested declarations.
func TestDerivedMessageKeepsAnnotationDistinctDeclarations(t *testing.T) {
	root := codegen.RunDSL(t, correspondenceSiblingsDSL)
	doc, err := BuildDocument(root.API, root.Types, root.ResultTypes)
	require.NoError(t, err)
	operation := doc.Paths["/events"].Operations["GET"]
	media := resolveSpecimenResponse(doc, operation, "200").Content["text/event-stream"]
	payload := resolveSpecimenSchema(doc.Components.Schemas, media.Schema)
	message := operation.Extensions[asyncContractExtensionName].(map[string]any)["messages"].(map[string]any)["outbound"].(map[string]any)
	for _, field := range []string{"plain", "annotated"} {
		member := resolveSpecimenSchema(doc.Components.Schemas, payload.Properties[field])
		want := ""
		if field == "annotated" {
			want = "Owned nested description."
		}
		require.Equal(t, want, member.Properties["value"].Description, "payload %s", field)
	}
	schema, ok := message["schema"].(*openapi.Schema)
	require.True(t, ok)
	require.Equal(t, "Owned nested description.", schema.Properties["annotated"].Properties["value"].Description)
	require.Empty(t, schema.Properties["plain"].Properties["value"].Description)
	require.Equal(t, rootExample(media.Schema, payload), exampleGroup(schema.Example, schema.Examples), "message and payload share one outcome")
}

func correspondenceSiblingsDSL() {
	dsl.Service("events", func() {
		dsl.Method("watch", func() {
			dsl.StreamingResult(func() {
				dsl.Attribute("plain", annotationNestedType("description", "PlainNested", false))
				dsl.Attribute("annotated", annotationNestedType("description", "AnnotatedNested", true))
			})
			dsl.HTTP(func() {
				dsl.GET("/events")
				dsl.ServerSentEvents()
			})
		})
	})
}

func correspondenceAliasDSL(override bool) func() {
	return func() {
		base := dsl.Type("DeprecatedValue", dsl.String, func() {
			dsl.Meta("openapi:deprecated", "true")
		})
		alias := dsl.Type("ValueAlias", base, func() {
			if override {
				dsl.Meta("openapi:deprecated", "false")
			}
		})
		correspondenceEventService(func() {
			dsl.Attribute("value", alias)
		})
	}
}

func correspondenceRefinementDSL() {
	inner := dsl.Type("Inner", dsl.Int, func() {
		dsl.Enum(1, 2)
	})
	outer := dsl.Type("Outer", inner, func() {
		dsl.Enum(2)
	})
	correspondenceEventService(func() {
		dsl.Attribute("value", outer)
	})
}

func correspondenceEventService(members func()) {
	dsl.Service("events", func() {
		dsl.Method("watch", func() {
			dsl.StreamingResult(members)
			dsl.HTTP(func() {
				dsl.GET("/events")
				dsl.ServerSentEvents()
			})
		})
	})
}

func correspondenceViewDSL() {
	result := dsl.ResultType("ViewedEvent", func() {
		dsl.Attributes(func() {
			dsl.Attribute("value", dsl.Int)
			dsl.Attribute("extra", dsl.String)
		})
		dsl.View("tiny", func() {
			dsl.Attribute("value")
		})
	})
	dsl.Service("events", func() {
		dsl.Method("watch", func() {
			dsl.StreamingResult(result, func() {
				dsl.View("tiny")
			})
			dsl.HTTP(func() {
				dsl.GET("/events")
				dsl.ServerSentEvents()
			})
		})
	})
}

// A producer roundtrip can turn a viewed alias into an ordinary named event
// with local nullability, enum refinement and distinct declaration examples.
func correspondenceNullableRefinementDSL() {
	base := dsl.Type("EventValueBase", dsl.Int64, func() {
		dsl.Enum(1, 2)
		dsl.Example(1)
	})
	event := dsl.Type("Event", func() {
		dsl.Attribute("value", base, func() {
			dsl.Nullable()
			dsl.Enum(2)
			dsl.Example(2)
		})
		dsl.Example(dsl.Val{"value": 2})
	})
	dsl.Service("events", func() {
		dsl.Method("watch", func() {
			dsl.StreamingResult(event, func() {
				dsl.Example(dsl.Val{"value": 2})
			})
			dsl.HTTP(func() {
				dsl.GET("/events")
				dsl.ServerSentEvents()
				dsl.Response(200)
			})
		})
	})
}

func TestDerivedMessagePreservesAuthoredOccurrenceExample(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		dsl.API("examples", func() {
			dsl.Meta("openapi:example", "false")
		})
		base := dsl.Type("Base", dsl.String, func() {
			dsl.Example("default", func() {
				dsl.Meta("openapi:example:scope", "schema")
				dsl.Value("declaration")
			})
		})
		opening := dsl.Type("Opening", func() {
			dsl.Attribute("status", dsl.Int)
			dsl.Required("status")
		})
		event := dsl.Type("Event", func() {
			dsl.Attribute("value", base, func() {
				dsl.Example("default", func() {
					dsl.Meta("openapi:example:scope", "schema")
					dsl.Value("occurrence")
				})
			})
		})
		dsl.Service("events", func() {
			dsl.Method("watch", func() {
				dsl.StreamingResult(event)
				dsl.HTTP(func() {
					dsl.GET("/events")
					dsl.ServerSentEvents()
					for _, status := range []int{200, 201} {
						dsl.Response(status, opening, func() {
							dsl.StatusField("status")
						})
					}
				})
			})
		})
	})
	doc, err := BuildDocument(root.API, root.Types, root.ResultTypes)
	require.NoError(t, err)
	operation := doc.Paths["/events"].Operations["GET"]
	for _, status := range []string{"200", "201"} {
		media := resolveSpecimenResponse(doc, operation, status).Content["text/event-stream"]
		payload := resolveSpecimenSchema(doc.Components.Schemas, media.Schema)
		require.Equal(t, "occurrence", payload.Properties["value"].Example, "status %s", status)
	}
	message := operation.Extensions[asyncContractExtensionName].(map[string]any)["messages"].(map[string]any)["outbound"].(map[string]any)
	schema, ok := message["schema"].(*openapi.Schema)
	require.True(t, ok)
	require.Equal(t, "occurrence", schema.Properties["value"].Example, "async retains authored local value, not merely agreement with response")
}
