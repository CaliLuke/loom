package openapiimport

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImportSSEInverseRequired(t *testing.T) {
	for _, tc := range []struct {
		name, keys string
		count      int
	}{
		{"class", "2XX", 1},
		{"created header", "201", 1},
		{"shared exact", "200,201", 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := sseInverseSource(strings.Split(tc.keys, ","), "", "")
			document, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			rendered, err := Render(document, Options{})
			require.NoError(t, err)
			require.Contains(t, string(rendered), "StreamingResult(")
			require.Contains(t, string(rendered), "ServerSentEvents()")
			require.NotContains(t, string(rendered), "SkipResponseBodyEncodeDecode()")
			require.NotContains(t, string(rendered), "OpenAPIBody(")
			requireRenderedDesignEvaluates(t, rendered, 1, fmt.Sprintf(`
	e := expr.Root.API.HTTP.Services[0].HTTPEndpoints[0]
	if e.SSE == nil || e.MethodExpr.StreamingResult == nil || e.MethodExpr.StreamingResult.Find("message") == nil {
		t.Fatal("stream event contract lost")
	}
	if len(e.Responses) != %d {
		t.Fatal("success occurrences lost")
	}
	for _, r := range e.Responses {
		if r.Handshake == nil || r.Body.Type != expr.Empty {
			t.Error("metadata/event ownership conflated")
		}
		if r.Status.Code == 201 && r.Headers.Find(r.Headers.KeyName("X-Trace")) == nil {
			t.Error("handshake header lost")
		}
		for _, code := range r.StatusDomain() {
			if code == 204 || code == 304 {
				t.Error("stream owns bodyless status")
			}
		}
	}
	if len(e.Responses) == 2 && e.Responses[0].Handshake.Type.(expr.UserType).ID() != e.Responses[1].Handshake.Type.(expr.UserType).ID() {
		t.Error("successes use distinct handshake declarations")
	}
`, tc.count))
		})
	}
}

func sseInverseSource(keys []string, local, extension string) []byte {
	var responses strings.Builder
	for _, key := range keys {
		fmt.Fprintf(&responses, "        %q:\n          description: events\n", key)
		if key == "201" {
			responses.WriteString("          headers:\n            X-Trace: {required: true, schema: {type: string}}\n")
		}
		responses.WriteString(`          content:
            text/event-stream:
              itemSchema:
                type: object
                additionalProperties: false
                required: [data]
                properties:
                  data:
                    type: string
                    contentMediaType: application/json
                    contentSchema:
                      $ref: '#/components/schemas/Event'
`)
		responses.WriteString(local)
	}
	return []byte(`openapi: 3.2.0
info: {title: SSE inverse, version: "1"}
paths:
  /events:
    get:
      operationId: events
` + extension + "      responses:\n" + responses.String() + `components:
  schemas:
    Event:
      type: object
      required: [message]
      properties:
        message: {type: string}
`)
}

func TestImportSSEEnvelopeRejections(t *testing.T) {
	for _, tc := range []struct {
		name, old, replacement, code string
	}{
		{"open envelope", "additionalProperties: false", "additionalProperties: true", "sse-envelope"},
		{"optional data", "required: [data]", "required: []", "sse-envelope"},
		{"envelope annotation", "                type: object", "                type: object\n                description: ignored", "sse-envelope"},
		{"data constraint", "                    type: string", "                    type: string\n                    minLength: 1", "sse-envelope"},
		{"named envelope", "              itemSchema:\n                type: object", "              itemSchema:\n                $ref: '#/components/schemas/Event'\n                type: object", "sse-envelope"},
		{"whole and item", "              itemSchema:", "              schema: {type: string}\n              itemSchema:", "sse-envelope"},
		{"wrong media", "text/event-stream:", "application/json:", "media-item-schema"},
		{"old dialect", "openapi: 3.2.0", "openapi: 3.1.1", "versioned-field"},
		{"error occurrence", "\"201\":", "\"400\":", "sse-response"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := string(sseInverseSource([]string{"201"}, "", ""))
			require.Contains(t, source, tc.old)
			source = strings.Replace(source, tc.old, tc.replacement, 1)
			_, diagnostics, err := Analyze([]byte(source))
			require.NoError(t, err)
			var codes []string
			for _, diagnostic := range diagnostics {
				codes = append(codes, diagnostic.Code)
			}
			require.Contains(t, codes, tc.code)
		})
	}
}
