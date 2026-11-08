package ir

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/openapi"
)

type sseSpecimen struct {
	// statuses lists typed-handshake success statuses in declaration order.
	// Empty selects the legacy result-derived exact 200 response.
	statuses  []int
	docs      int
	mapped    bool
	primitive bool
	recursive bool
	nullable  bool
	local     bool
	body      string
	mixed     bool
}

// observedProjection is the custom projector output. Its call number identifies
// the projector invocation that owns a rendered example.
type observedProjection struct {
	Call int
	Raw  any
}

func TestTypedHandshakeEventPayloadIsStatusIndependent(t *testing.T) {
	for _, specimen := range []sseSpecimen{
		{statuses: []int{201}},
		{statuses: []int{200, 201}},
		{statuses: []int{201, 200}},
		{statuses: []int{200, 201}, mapped: true},
		{statuses: []int{201, 200}, mapped: true},
	} {
		t.Run(specimen.name(), func(t *testing.T) {
			doc := buildSpecimenDocument(t, specimen)
			operation := doc.Paths["/events"].Operations["GET"]
			var first *Schema
			for _, status := range specimen.statuses {
				response := resolveSpecimenResponse(doc, operation, strconv.Itoa(status))
				require.NotNil(t, response, "status %d", status)
				media := response.Content["text/event-stream"]
				require.NotNil(t, media, "status %d documents the event stream", status)
				require.NotNil(t, media.Schema, "status %d documents the event payload", status)
				if first == nil {
					first = media.Schema
				}
				require.Same(t, first, media.Schema, "status %d consumes the one completed event outcome", status)
				require.Contains(t, response.Headers, "X-Session")
				require.NotContains(t, response.Headers, "status", "the handshake StatusField is never a header")
				event := resolveSpecimenSchema(doc.Components.Schemas, media.Schema)
				require.Equal(t, "string", event.Properties["status"].Type, "same-named handshake members never remove event members")
				require.Equal(t, "integer", event.Properties["session"].Type)
			}
		})
	}
}

func TestTypedHandshakeIndependentDocumentationOverride(t *testing.T) {
	for _, specimen := range []sseSpecimen{
		{statuses: []int{200, 201}, docs: 200},
		{statuses: []int{201, 200}, docs: 200},
		{statuses: []int{200, 201}, docs: 201},
		{statuses: []int{201, 200}, docs: 201},
	} {
		t.Run(specimen.name(), func(t *testing.T) {
			doc := buildSpecimenDocument(t, specimen)
			operation := doc.Paths["/events"].Operations["GET"]
			for _, status := range specimen.statuses {
				response := resolveSpecimenResponse(doc, operation, strconv.Itoa(status))
				media := response.Content["text/event-stream"]
				require.NotNil(t, media, "status %d", status)
				schema := resolveSpecimenSchema(doc.Components.Schemas, media.Schema)
				if status == specimen.docs {
					require.Contains(t, schema.Properties, "summary", "status %d retains its authored override", status)
					require.NotContains(t, schema.Properties, "session")
					continue
				}
				require.Contains(t, schema.Properties, "session", "status %d retains event documentation", status)
				require.NotContains(t, schema.Properties, "summary")
			}
			requireIndependentAsync(t, specimen)
		})
	}
}

func TestDerivedAsyncMessageReusesCompletedEventOutcome(t *testing.T) {
	for _, specimen := range []sseSpecimen{
		{},
		{local: true},
		{primitive: true},
		{body: "equal"},
		{statuses: []int{201}},
		{statuses: []int{200, 201}},
		{statuses: []int{201, 200}},
		{statuses: []int{200, 201}, nullable: true},
		{statuses: []int{200, 201}, recursive: true},
	} {
		t.Run(specimen.name(), func(t *testing.T) {
			bodies, endpoint, calls := buildSpecimenBodies(t, specimen, false)
			payload := specimenPayload(t, bodies, endpoint, specimen)
			before := *calls
			message := specimenOutboundMessage(t, bodies, endpoint)
			require.Equal(t, before, *calls, "the derived message invokes no projector")
			payloadRoot := resolveSpecimenSchema(bodies.asyncComponents, payload)
			require.NotNil(t, exampleGroup(message.Example, message.Examples), "the derived message retains the event example owner")
			require.Equal(t, rootExample(payload, payloadRoot), exampleGroup(message.Example, message.Examples), "message and payload share one projected outcome")
			if !specimen.primitive {
				require.Equal(t, payloadRoot.Properties["status"].Example, message.Properties["status"].Example)
			}
			if specimen.local {
				require.Equal(t, "Locally described event.", message.Description)
			}
			if specimen.nullable {
				note := message.Properties["note"]
				require.Len(t, note.AnyOf, 2, "the nullable wrapper is retained")
				require.Equal(t, openapi.Type(openapi.Null), note.AnyOf[1].Type)
			}
			if specimen.recursive {
				next := message.Properties["next"]
				require.NotNil(t, next)
				name, ok := schemaComponentName(next.Ref)
				require.True(t, ok, "the recursive edge stays a reference cut")
				require.Contains(t, bodies.asyncComponents, name, "the recursive closure is retained")
			}
		})
	}
}

func TestDerivedAsyncMessageFilterAllRetainsSuppression(t *testing.T) {
	for _, specimen := range []sseSpecimen{{}, {statuses: []int{200, 201}}} {
		t.Run(specimen.name(), func(t *testing.T) {
			bodies, endpoint, calls := buildSpecimenBodies(t, specimen, true)
			payload := specimenPayload(t, bodies, endpoint, specimen)
			require.NotZero(t, *calls, "the projector owns the event examples")
			before := *calls
			message := specimenOutboundMessage(t, bodies, endpoint)
			require.Equal(t, before, *calls, "filter-all cannot trigger fresh acquisition")
			payloadRoot := resolveSpecimenSchema(bodies.asyncComponents, payload)
			require.Nil(t, rootExample(payload, payloadRoot))
			require.Nil(t, message.Example)
			require.Empty(t, message.Examples)
			require.Nil(t, message.Properties["status"].Example)
		})
	}
}

func TestIneligibleAsyncMessageRetainsIndependentOwner(t *testing.T) {
	for _, specimen := range []sseSpecimen{
		{mixed: true},
		{body: "local"},
		{statuses: []int{200, 201}, mapped: true},
		{statuses: []int{201, 200}, docs: 201},
	} {
		t.Run(specimen.name(), func(t *testing.T) {
			requireIndependentAsync(t, specimen)
		})
	}
}

func TestMixedResultSSEPayloadKeepsResultDocumentation(t *testing.T) {
	doc := buildSpecimenDocument(t, sseSpecimen{mixed: true})
	response := resolveSpecimenResponse(doc, doc.Paths["/events"].Operations["GET"], "200")
	for contentType, media := range response.Content {
		schema := resolveSpecimenSchema(doc.Components.Schemas, media.Schema)
		require.Contains(t, schema.Properties, "summary", "mixed result %s retains the unary result", contentType)
	}
}

func TestAsyncHandshakeStatusesList(t *testing.T) {
	for _, test := range []struct {
		specimen sseSpecimen
		want     []string
	}{
		{sseSpecimen{}, []string{"200"}},
		{sseSpecimen{statuses: []int{201}}, []string{"201"}},
		{sseSpecimen{statuses: []int{200, 201}}, []string{"200", "201"}},
		{sseSpecimen{statuses: []int{201, 200}}, []string{"200", "201"}},
		{sseSpecimen{statuses: []int{201, 200}, docs: 200}, []string{"200", "201"}},
	} {
		t.Run(test.specimen.name(), func(t *testing.T) {
			doc := buildSpecimenDocument(t, test.specimen)
			response := specimenAsyncHandshakeResponse(t, doc, "/events")
			require.Equal(t, test.want, response["statuses"])
			require.NotContains(t, response, "status", "no scalar alias is retained")
		})
	}
	t.Run("websocket", func(t *testing.T) {
		root := codegen.RunDSL(t, func() {
			dsl.Service("socket", func() {
				dsl.Method("watch", func() {
					dsl.StreamingResult(dsl.String)
					dsl.HTTP(func() {
						dsl.GET("/socket")
					})
				})
			})
		})
		doc, err := BuildDocument(root.API, root.Types, root.ResultTypes)
		require.NoError(t, err)
		response := specimenAsyncHandshakeResponse(t, doc, "/socket")
		require.Equal(t, []string{"101"}, response["statuses"])
		require.NotContains(t, response, "status")
	})
}

// requireIndependentAsync checks that the async message does not copy the
// completed event outcome: its examples come from separate projector calls.
func requireIndependentAsync(t *testing.T, specimen sseSpecimen) {
	t.Helper()
	bodies, endpoint, calls := buildSpecimenBodies(t, specimen, false)
	before := *calls
	message := specimenOutboundMessage(t, bodies, endpoint)
	require.Greater(t, *calls, before, "the ineligible message keeps its independent acquisition")
	if message.Example != nil {
		observed, ok := message.Example.(observedProjection)
		require.True(t, ok)
		require.Greater(t, observed.Call, before)
	}
}

func buildSpecimenDocument(t *testing.T, specimen sseSpecimen) *Document {
	t.Helper()
	root := codegen.RunDSL(t, sseSpecimenDSL(specimen))
	doc, err := BuildDocument(root.API, root.Types, root.ResultTypes)
	require.NoError(t, err)
	return doc
}

func buildSpecimenBodies(t *testing.T, specimen sseSpecimen, filterAll bool) (*EndpointBodies, *EndpointBodies, *int) {
	t.Helper()
	root := codegen.RunDSL(t, sseSpecimenDSL(specimen))
	calls := 0
	bodies := BuildBodyTypes(root.API, root.Types, root.ResultTypes, WithExampleValue(func(_ *expr.AttributeExpr, raw any) (any, bool) {
		calls++
		if filterAll || raw == nil {
			return nil, false
		}
		return observedProjection{Call: calls, Raw: raw}, true
	}))
	require.NoError(t, bodies.Err)
	endpoint := bodies.Services["events"]["watch"]
	require.NotNil(t, endpoint)
	return endpoint, endpoint, &calls
}

func specimenPayload(t *testing.T, bodies, _ *EndpointBodies, specimen sseSpecimen) *Schema {
	t.Helper()
	statuses := specimen.statuses
	if len(statuses) == 0 {
		statuses = []int{200}
	}
	var payload *Schema
	for _, status := range statuses {
		schemas := bodies.ResponseBodies[strconv.Itoa(status)]
		require.Len(t, schemas, 1, "status %d", status)
		require.NotNil(t, schemas[0], "status %d", status)
		if payload == nil {
			payload = schemas[0]
		}
		require.Same(t, payload, schemas[0], "status %d", status)
	}
	return payload
}

func specimenOutboundMessage(t *testing.T, bodies, _ *EndpointBodies) *openapi.Schema {
	t.Helper()
	messages := buildAsyncMessages(bodies.prepared, "/events", bodies)
	message, ok := messages["outbound"].(map[string]any)
	require.True(t, ok)
	schema, ok := message["schema"].(*openapi.Schema)
	require.True(t, ok)
	return schema
}

// rootExample returns the payload root's complete example group: the local
// owner when present, otherwise the declaration owner.
func rootExample(payload, resolved *Schema) any {
	if group := exampleGroup(payload.Example, payload.Examples); group != nil {
		return group
	}
	return exampleGroup(resolved.Example, resolved.Examples)
}

func exampleGroup(example any, examples []any) any {
	if len(examples) > 0 {
		return examples
	}
	return example
}

func specimenAsyncHandshakeResponse(t *testing.T, doc *Document, path string) map[string]any {
	t.Helper()
	async, ok := doc.Paths[path].Operations["GET"].Extensions[asyncContractExtensionName].(map[string]any)
	require.True(t, ok)
	handshake, ok := async["handshake"].(map[string]any)
	require.True(t, ok)
	response, ok := handshake["response"].(map[string]any)
	require.True(t, ok)
	return response
}

func resolveSpecimenResponse(doc *Document, operation *Operation, status string) *Response {
	ref := operation.Responses[status]
	if ref == nil {
		return nil
	}
	if ref.Ref != "" {
		return doc.Components.Responses[ref.Ref[len(ResponseComponentRefPrefix):]].Value
	}
	return ref.Value
}

func resolveSpecimenSchema(components map[string]*Schema, schema *Schema) *Schema {
	for schema != nil {
		name, ok := schemaComponentName(schema.Ref)
		if !ok || components[name] == nil {
			return schema
		}
		schema = components[name]
	}
	return schema
}

func (s sseSpecimen) name() string {
	return fmt.Sprintf("statuses=%v/docs=%d/mapped=%t/primitive=%t/recursive=%t/nullable=%t/local=%t/body=%s/mixed=%t",
		s.statuses, s.docs, s.mapped, s.primitive, s.recursive, s.nullable, s.local, s.body, s.mixed)
}

func sseSpecimenDSL(s sseSpecimen) func() {
	return func() {
		var event expr.DataType = dsl.String
		if !s.primitive {
			event = dsl.Type("StreamEvent", func() {
				dsl.Attribute("status", dsl.String, func() {
					dsl.Example("ready")
				})
				dsl.Attribute("session", dsl.Int)
				if s.mapped {
					dsl.Attribute("payload", dsl.String)
				}
				if s.nullable {
					dsl.Attribute("note", dsl.String, func() {
						dsl.Nullable()
						dsl.Example("noted")
					})
				}
				if s.recursive {
					dsl.Attribute("next", "StreamEvent")
				}
				dsl.Required("status", "session")
				dsl.Example("declared", map[string]any{"status": "ready", "session": 1})
			})
		}
		opening := dsl.Type("Opening", func() {
			if len(s.statuses) > 1 {
				dsl.Attribute("status", dsl.Int)
				dsl.Required("status")
			}
			dsl.Attribute("session", dsl.String)
			dsl.Required("session")
		})
		summary := dsl.Type("EventSummary", func() {
			dsl.Attribute("status", dsl.String)
			dsl.Attribute("summary", dsl.String)
			dsl.Required("summary")
		})
		dsl.Service("events", func() {
			dsl.Method("watch", func() {
				if s.mixed {
					dsl.Result(summary)
				}
				dsl.StreamingResult(event, func() {
					if s.local {
						dsl.Description("Locally described event.")
						dsl.Example("local", map[string]any{"status": "local", "session": 2})
					}
				})
				dsl.HTTP(func() {
					dsl.GET("/events")
					dsl.ServerSentEvents(func() {
						if s.mapped {
							dsl.SSEEventData("payload")
						}
					})
					if len(s.statuses) == 0 {
						if s.body != "" {
							dsl.Response(dsl.StatusOK, func() {
								dsl.Body(event, func() {
									if s.body == "local" {
										dsl.Example("body", map[string]any{"status": "body", "session": 3})
									}
								})
							})
						}
						return
					}
					for _, status := range s.statuses {
						dsl.Response(status, opening, func() {
							if len(s.statuses) > 1 {
								dsl.StatusField("status")
							}
							dsl.Header("session:X-Session")
							if status == s.docs {
								dsl.OpenAPIBody(summary)
							}
						})
					}
				})
			})
		})
	}
}

func TestViewSelectedEventOutcomeUsesProjectedValues(t *testing.T) {
	root := codegen.RunDSL(t, func() {
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
	})
	doc, err := BuildDocument(root.API, root.Types, root.ResultTypes)
	require.NoError(t, err)
	operation := doc.Paths["/events"].Operations["GET"]
	media := resolveSpecimenResponse(doc, operation, "200").Content["text/event-stream"]
	payload := resolveSpecimenSchema(doc.Components.Schemas, media.Schema)
	require.NotContains(t, payload.Properties, "extra")
	for _, example := range []any{media.Schema.Example, payload.Example} {
		if values, ok := example.(map[string]any); ok {
			require.NotContains(t, values, "extra", "the projected event samples only view members")
		}
	}
}

// minimumWitnessDSL is the producer witness for the importer roundtrip: a named
// integer event with Minimum(0), a StreamingResult occurrence narrowed with
// Minimum(5) and exact 200/201 typed handshakes sharing one opening carrier.
func minimumWitnessDSL() {
	event := dsl.Type("Event", dsl.Int, func() {
		dsl.Minimum(0)
	})
	opening := dsl.Type("Opening", func() {
		dsl.Attribute("status", dsl.Int)
		dsl.Attribute("session", dsl.String)
		dsl.Required("status", "session")
	})
	dsl.Service("events", func() {
		dsl.Method("watch", func() {
			dsl.StreamingResult(event, func() {
				dsl.Minimum(5)
			})
			dsl.HTTP(func() {
				dsl.GET("/events")
				dsl.ServerSentEvents()
				for _, status := range []int{dsl.StatusOK, dsl.StatusCreated} {
					dsl.Response(status, opening, func() {
						dsl.StatusField("status")
						dsl.Header("session:X-Session")
					})
				}
			})
		})
	})
}

func TestMinimumWitnessRetainsOccurrenceConstraint(t *testing.T) {
	root := codegen.RunDSL(t, minimumWitnessDSL)
	doc, err := BuildDocument(root.API, root.Types, root.ResultTypes)
	require.NoError(t, err)
	operation := doc.Paths["/events"].Operations["GET"]
	var payload *Schema
	for _, status := range []string{"200", "201"} {
		media := resolveSpecimenResponse(doc, operation, status).Content["text/event-stream"]
		require.NotNil(t, media, status)
		if payload == nil {
			payload = media.Schema
		}
		require.Same(t, payload, media.Schema, status)
	}
	require.Equal(t, 5.0, effectiveMinimum(t, doc.Components.Schemas, payload), "the local occurrence narrows the payload")
	if name, ok := schemaComponentName(payload.Ref); ok {
		require.Equal(t, 0.0, effectiveMinimum(t, doc.Components.Schemas, doc.Components.Schemas[name]), "the declaration keeps its own bound")
	}
	async := operation.Extensions[asyncContractExtensionName].(map[string]any)
	message := async["messages"].(map[string]any)["outbound"].(map[string]any)["schema"].(*openapi.Schema)
	require.Equal(t, []string{"200", "201"}, async["handshake"].(map[string]any)["response"].(map[string]any)["statuses"])
	require.True(t, renderedMinimum(message, 5), "the derived message keeps the occurrence bound")
}

// effectiveMinimum returns the strongest minimum reachable through the local
// schema and its conjunctive references.
func effectiveMinimum(t *testing.T, components map[string]*Schema, schema *Schema) float64 {
	t.Helper()
	best := -1.0
	var visit func(*Schema)
	visit = func(node *Schema) {
		if node == nil {
			return
		}
		if node.Minimum != nil {
			value, err := strconv.ParseFloat(node.Minimum.String(), 64)
			require.NoError(t, err)
			best = max(best, value)
		}
		if name, ok := schemaComponentName(node.Ref); ok {
			visit(components[name])
		}
		for _, branch := range node.AllOf {
			visit(branch)
		}
	}
	visit(schema)
	return best
}

func renderedMinimum(schema *openapi.Schema, want float64) bool {
	if schema == nil {
		return false
	}
	if schema.Minimum != nil {
		if value, err := strconv.ParseFloat(schema.Minimum.String(), 64); err == nil && value == want {
			return true
		}
	}
	for _, branch := range schema.AllOf {
		if renderedMinimum(branch, want) {
			return true
		}
	}
	return false
}
