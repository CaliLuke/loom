package ir

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
)

// A typed mixed success documents both chosen representations under its one
// status: the whole unary Result and the SSE event, with shared metadata.
// Same-named handshake members never subtract Result fields.
func TestMixedTypedResponseDocumentsUnaryAndEvent(t *testing.T) {
	for _, kind := range []string{"object", "primitive", "collection"} {
		t.Run(kind, func(t *testing.T) {
			root := codegen.RunDSL(t, mixedTypedResponseDSL(kind, true))
			doc, err := BuildDocument(root.API, root.Types, root.ResultTypes)
			require.NoError(t, err)
			operation := doc.Paths["/"].Operations["GET"]
			require.ElementsMatch(t, []string{"201"}, mapKeys(operation.Responses))
			response := resolveSpecimenResponse(doc, operation, "201")
			require.ElementsMatch(t, []string{"application/json", "text/event-stream"}, mapKeys(response.Content))
			require.ElementsMatch(t, []string{"X-Session"}, mapKeys(response.Headers), "metadata is shared and never a body member")

			unary := resolveSpecimenSchema(doc.Components.Schemas, response.Content["application/json"].Schema)
			switch kind {
			case "object":
				require.ElementsMatch(t, []string{"status", "session", "total"}, mapKeys(unary.Properties), "the whole Result keeps same-named fields")
				require.Equal(t, "string", unary.Properties["status"].Type)
			case "primitive":
				require.Equal(t, "integer", unary.Type)
			case "collection":
				require.Equal(t, "array", unary.Type)
				require.Equal(t, "string", unary.Items.Type)
			}
			event := resolveSpecimenSchema(doc.Components.Schemas, response.Content["text/event-stream"].Schema)
			require.Equal(t, "object", event.Type)
			require.Contains(t, event.Properties, "tick", "the event comes from StreamingResult")
		})
	}
}

// Controls: a non-mixed typed handshake documents only its event stream, and an
// ordinary unary endpoint documents only its body.
func TestMixedTypedResponseControls(t *testing.T) {
	root := codegen.RunDSL(t, mixedTypedResponseDSL("object", false))
	doc, err := BuildDocument(root.API, root.Types, root.ResultTypes)
	require.NoError(t, err)
	response := resolveSpecimenResponse(doc, doc.Paths["/"].Operations["GET"], "201")
	require.ElementsMatch(t, []string{"text/event-stream"}, mapKeys(response.Content))

	root = codegen.RunDSL(t, func() {
		dsl.Service("unary", func() {
			dsl.Method("read", func() {
				dsl.Result(func() {
					dsl.Attribute("status", dsl.String)
				})
				dsl.HTTP(func() {
					dsl.GET("/")
				})
			})
		})
	})
	doc, err = BuildDocument(root.API, root.Types, root.ResultTypes)
	require.NoError(t, err)
	response = resolveSpecimenResponse(doc, doc.Paths["/"].Operations["GET"], "200")
	require.ElementsMatch(t, []string{"application/json"}, mapKeys(response.Content))
}

func mixedTypedResponseDSL(kind string, mixed bool) func() {
	return func() {
		opening := dsl.Type("Opening", func() {
			dsl.Attribute("status", dsl.Int)
			dsl.Attribute("session", dsl.String)
			dsl.Required("status", "session")
		})
		event := dsl.Type("TickEvent", func() {
			dsl.Attribute("tick", dsl.Int)
			dsl.Required("tick")
		})
		dsl.Service("mixed", func() {
			dsl.Method("read", func() {
				if mixed {
					switch kind {
					case "object":
						dsl.Result(func() {
							dsl.Attribute("status", dsl.String)
							dsl.Attribute("session", dsl.String)
							dsl.Attribute("total", dsl.Int)
							dsl.Required("status", "session", "total")
						})
					case "primitive":
						dsl.Result(dsl.Int)
					case "collection":
						dsl.Result(dsl.ArrayOf(dsl.String))
					}
				}
				dsl.StreamingResult(event)
				dsl.HTTP(func() {
					dsl.GET("/")
					dsl.ServerSentEvents()
					dsl.Response(dsl.StatusCreated, opening, func() {
						dsl.StatusField("status")
						dsl.Header("session:X-Session")
					})
				})
			})
		})
	}
}
