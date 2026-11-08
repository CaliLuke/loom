package openapiimport

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolverSSETypedSourceClosure(t *testing.T) {
	for _, tc := range []struct {
		name, response string
		files          map[string]string
	}{
		{"inline", "", nil},
		{"local response", "#/components/responses/Events", nil},
		{"external response", "response.yaml#/Events", map[string]string{"response.yaml": "Events:\n" + sseResolverResponse("  ")}},
		{"chained response", "response.yaml#/Events", map[string]string{"response.yaml": "Events: {$ref: 'chain.yaml#/Events'}", "chain.yaml": "Events:\n" + sseResolverResponse("  ")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			entry := `openapi: 3.2.0
info: {title: Closure, version: '1'}
paths:
  /events:
    get:
      x-loom-async:
        messages:
          outbound:
            schema:
              $ref: 'async.yaml#/Payload'
              example: {$ref: 'opaque.yaml#/Never'}
      responses:
        '200':
`
			if tc.response == "" {
				entry += sseResolverResponse("          ")
			} else {
				entry += "          $ref: '" + tc.response + "'\n"
			}
			if strings.HasPrefix(tc.response, "#") {
				entry += "components:\n  responses:\n    Events:\n" + sseResolverResponse("      ")
			}
			files := map[string]string{"event.yaml": "Payload: {type: string}", "async.yaml": "Payload: {type: string}"}
			for name, body := range tc.files {
				files[name] = body
			}
			resolved, diagnostics, err := resolveDescription(resolverFiles(t, []byte(entry), files))
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			found := make(map[string]bool)
			for _, edge := range resolved.edges {
				found[edge.authored] = edge.target != nil
			}
			require.True(t, found["event.yaml#/Payload"], "item payload must be a typed source edge")
			require.True(t, found["async.yaml#/Payload"], "guarded reserved payload must be a typed source edge")
			require.NotContains(t, found, "opaque.yaml#/Never")
		})
	}
}

func sseResolverResponse(indent string) string {
	return indent + strings.ReplaceAll(`description: events
content:
  text/event-stream:
    itemSchema:
      type: object
      additionalProperties: false
      required: [data]
      properties:
        data:
          type: string
          contentMediaType: application/json
          contentSchema: {$ref: 'event.yaml#/Payload'}
`, "\n", "\n"+indent) + "\n"
}

func TestResolverSSEGuardRemainsClosed(t *testing.T) {
	for _, response := range []string{"#/components/responses/Missing", "#/components/responses/Loop"} {
		t.Run(response, func(t *testing.T) {
			source := Source{Bytes: []byte(`openapi: 3.2.0
info: {title: Closed, version: '1'}
paths:
  /events:
    get:
      x-loom-async:
        messages: {outbound: {schema: {$ref: 'unadmitted.yaml#/Payload'}}}
      responses:
        '200': {$ref: '` + response + `'}
components:
  responses:
    Loop: {$ref: '#/components/responses/Loop'}
`)}
			resolved, _, err := resolveDescription(source)
			require.NoError(t, err)
			for _, edge := range resolved.edges {
				require.NotEqual(t, "unadmitted.yaml#/Payload", edge.authored)
			}
		})
	}
}
