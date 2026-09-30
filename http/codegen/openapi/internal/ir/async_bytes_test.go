package ir

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/openapi"
)

func TestAsyncByteSchemasUsePreparedRepresentations(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		message := dsl.Type("ByteMessage", func() {
			dsl.Attribute("data", dsl.Bytes, func() {
				dsl.MaxLength(2)
			})
			dsl.Attribute("external", dsl.Bytes, func() {
				dsl.Meta("struct:field:type", "custom.Blob", "example.com/custom")
				dsl.MaxLength(2)
			})
		})
		dsl.Service("byte-stream", func() {
			dsl.Method("ordinary", func() {
				dsl.Payload(message)
				dsl.Result(message)
				dsl.HTTP(func() {
					dsl.POST("/ordinary")
				})
			})
			dsl.Method("socket", func() {
				dsl.StreamingPayload(message)
				dsl.StreamingResult(message)
				dsl.HTTP(func() {
					dsl.GET("/socket")
				})
			})
		})
	})
	doc, err := BuildDocument(root.API, root.Types, root.ResultTypes)
	require.NoError(t, err)
	async := doc.Paths["/socket"].Operations["GET"].Extensions[asyncContractExtensionName].(map[string]any)
	messages := async["messages"].(map[string]any)
	for _, direction := range []string{"inbound", "outbound"} {
		t.Run(direction, func(t *testing.T) {
			message := messages[direction].(map[string]any)
			schema := message["schema"].(*openapi.Schema)
			require.Equal(t, "base64", schema.Properties["data"].ContentEncoding)
			require.Empty(t, schema.Properties["data"].Format)
			require.Nil(t, schema.Properties["data"].MaxLength)
			require.Empty(t, schema.Properties["external"].ContentEncoding)
			require.Equal(t, 2, *schema.Properties["external"].MaxLength)
		})
	}
}

func TestSSEByteProjectionSchemasUsePreparedRepresentations(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		event := dsl.ResultType("application/vnd.byte-event", func() {
			dsl.Attributes(func() {
				dsl.Attribute("event", dsl.String)
				dsl.Attribute("data", dsl.Bytes, func() {
					dsl.MaxLength(2)
				})
				dsl.Attribute("extra", dsl.String)
				dsl.Required("event", "data")
			})
			dsl.View("small", func() {
				dsl.Attribute("event")
				dsl.Attribute("data")
			})
			dsl.View("full", func() {
				dsl.Attribute("event")
				dsl.Attribute("data")
				dsl.Attribute("extra")
			})
		})
		dsl.Service("byte-events", func() {
			dsl.Method("watch", func() {
				dsl.StreamingResult(event)
				dsl.HTTP(func() {
					dsl.GET("/events")
					dsl.ServerSentEvents(func() {
						dsl.SSEEventType("event")
						dsl.SSEProjection("small", "small")
						dsl.SSEProjection("full", "full")
					})
				})
			})
		})
	})
	doc, err := BuildDocument(root.API, root.Types, root.ResultTypes)
	require.NoError(t, err)
	operation := doc.Paths["/events"].Operations["GET"]
	async := operation.Extensions[asyncContractExtensionName].(map[string]any)
	message := async["messages"].(map[string]any)["outbound"].(map[string]any)
	inline := message["schema"].(*openapi.Schema)
	require.Len(t, inline.OneOf, 2)
	for _, branch := range inline.OneOf {
		require.Equal(t, "base64", branch.Properties["data"].ContentEncoding)
	}
	response := operation.Responses["200"]
	if response.Ref != "" {
		response = doc.Components.Responses[response.Ref[len(ResponseComponentRefPrefix):]]
	}
	body := response.Value.Content["text/event-stream"].Schema
	require.Len(t, body.OneOf, 2)
	for _, branch := range body.OneOf {
		projected := representationSchema(t, doc, branch)
		require.Equal(t, "base64", projected.Properties["data"].ContentEncoding)
	}
}

func TestAsyncByteAliasBoundsRemainConjoined(t *testing.T) {
	minimum, maximum := 3, 2
	base := &expr.UserTypeExpr{TypeName: "AsyncBytes", AttributeExpr: &expr.AttributeExpr{Type: expr.Bytes, Validation: &expr.ValidationExpr{MaxLength: &maximum}}}
	attr := &expr.AttributeExpr{Type: base, Validation: &expr.ValidationExpr{MinLength: &minimum}}
	analyzer := NewAnalyzer(expr.NewRandom("async-alias"), false)
	prepared := analyzeAsyncSchema(analyzer, attr, inlineAsyncTestEndpoint(t, attr), false, "async-alias")
	analyzer.finalizeRepresentations()
	materialized := materializeAsyncSchema(prepared, analyzer.schemas)
	result := materialized.schema
	require.Nil(t, result.MinLength, "remove the old occurrence length gate")
	require.Empty(t, result.AllOf, "the parent inline consumer has a flat alias baseline")
	require.Equal(t, "base64", result.ContentEncoding)
	require.Equal(t, &Schema{}, result.Not)
	require.Nil(t, result.MinLength)
	require.Nil(t, result.MaxLength)
}

func TestAsyncSchemaInliningKeepsRecursiveReferencesAndSiblings(t *testing.T) {
	source := &Schema{Ref: toRef("Node"), Not: &Schema{Ref: toRef("Forbidden")}}
	components := map[string]*Schema{
		"Node":      {Type: "object", Properties: map[string]*Schema{"next": {Ref: toRef("Node")}}},
		"Forbidden": {Type: "string", Pattern: "bad"},
	}
	result := materializeAsyncSchema(&asyncSchema{schema: source}, components).schema
	require.Empty(t, result.Ref)
	require.Len(t, result.AllOf, 1)
	require.Equal(t, toRef("Node"), result.AllOf[0].Properties["next"].Ref)
	require.Equal(t, "bad", result.Not.Pattern)
	require.Equal(t, toRef("Forbidden"), source.Not.Ref)
	result.AllOf[0].Properties["other"] = &Schema{}
	require.NotContains(t, components["Node"].Properties, "other")
}

func TestAsyncSchemaSamplesOnceAfterConstraintAnalysis(t *testing.T) {
	calls := 0
	attribute := &expr.AttributeExpr{Type: &expr.Object{
		{Name: "name", Attribute: &expr.AttributeExpr{Type: expr.String}},
	}}
	analyzer := NewAnalyzer(expr.NewRandom("async-once"), false, WithExampleValue(func(_ *expr.AttributeExpr, raw any) (any, bool) {
		calls++
		return raw, true
	}))
	prepared := analyzeAsyncSchema(analyzer, attribute, inlineAsyncTestEndpoint(t, attribute), false, "async-once")
	require.Zero(t, calls, "constraint analysis must not sample or invoke projection callbacks")
	analyzer.finalizeRepresentations()
	materialized := materializeAsyncSchema(prepared, analyzer.schemas)
	result := materialized.schema
	applyPreparedAsyncExamples(analyzer, result, prepared, materialized.structures)
	require.Equal(t, 2, calls, "one callback per inline object/scalar occurrence")
	require.NotNil(t, result.Example)
	require.NotNil(t, result.Properties["name"].Example)
}
