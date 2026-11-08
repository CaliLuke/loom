package openapiimport

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImportSSEReservedContract(t *testing.T) {
	valid := sseAsyncContract("['200', '201']", "{$ref: '#/components/schemas/Event'}")
	for _, tc := range []struct{ name, extension, code string }{
		{"valid reference", valid, ""},
		{"valid expansion", sseAsyncContract("['200', '201']", "{type: object, required: [message], properties: {message: {type: string}}}"), ""},
		{"legacy scalar", strings.Replace(valid, "statuses: ['200', '201']", "status: 200", 1), "sse-async"},
		{"wrong key", strings.Replace(valid, "'201'", "'202'", 1), "sse-async"},
		{"wrong order", strings.Replace(valid, "['200', '201']", "['201', '200']", 1), "sse-async"},
		{"duplicates", strings.Replace(valid, "['200', '201']", "['200', '200', '201']", 1), "sse-async"},
		{"route", strings.Replace(valid, "path: /events", "path: /other", 1), "sse-async"},
		{"method", strings.Replace(valid, "method: GET", "method: POST", 1), "sse-async"},
		{"mapping", strings.Replace(valid, "dataField: null", "dataField: message", 1), "sse-async"},
		{"unknown member", strings.Replace(valid, "transport: sse", "transport: sse\n        extra: true", 1), "sse-async"},
		{"payload mismatch", strings.Replace(valid, "{$ref: '#/components/schemas/Event'}", "{type: string}", 1), "sse-async"},
		{"sample mismatch", strings.Replace(valid, "{$ref: '#/components/schemas/Event'}", "{$ref: '#/components/schemas/Event', example: {message: other}}", 1), "sse-async"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc, diagnostics, err := Analyze(sseInverseSource([]string{"200", "201"}, "", tc.extension))
			require.NoError(t, err)
			if tc.code != "" {
				require.Contains(t, diagnostics.Error(), tc.code)
				return
			}
			require.Empty(t, diagnostics)
			out, err := Render(doc, Options{})
			require.NoError(t, err)
			require.NotContains(t, string(out), "x-loom-async")
		})
	}
}

func sseAsyncContract(statuses, schema string) string {
	return `      x-loom-async:
        transport: sse
        direction: server
        handshake:
          path: /events
          request: {method: GET}
          response: {statuses: ` + statuses + `, contentType: text/event-stream}
        messages:
          outbound:
            contentType: application/json
            schema: ` + schema + `
            sse: {requestIDField: null, dataField: null, idField: null, eventField: null, retryField: null}
`
}

func TestImportSSEOccurrenceContract(t *testing.T) {
	for _, tc := range []struct {
		name, first, second string
		valid               bool
	}{
		{"equal local sample", "example: {message: same}", "example: {message: same}", true},
		{"different local sample", "example: {message: first}", "example: {message: second}", false},
		{"sample presence", "example: {message: first}", "", false},
		{"empty examples presence", "examples: []", "", false},
		{"equal description", "description: local", "description: local", true},
		{"different description", "description: first", "description: second", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := string(sseInverseSource([]string{"200", "201"}, "", ""))
			needle := "$ref: '#/components/schemas/Event'"
			position := strings.Index(source, needle) + len(needle)
			source = source[:position] + "\n                      " + tc.first + source[position:]
			position = strings.LastIndex(source, needle) + len(needle)
			source = source[:position] + "\n                      " + tc.second + source[position:]
			doc, diagnostics, err := Analyze([]byte(source))
			require.NoError(t, err)
			if !tc.valid {
				require.Contains(t, diagnostics.Error(), "sse-event")
				return
			}
			require.Empty(t, diagnostics)
			out, err := Render(doc, Options{})
			require.NoError(t, err)
			require.Contains(t, string(out), "StreamingResult(")
		})
	}
}

func TestImportSSEExpandedOccurrenceAnnotations(t *testing.T) {
	for _, tc := range []struct{ name, local, extension string }{
		{"root example", "example: {message: same}", "{type: object, required: [message], properties: {message: {type: string}}, example: {message: same}}"},
		{"root description", "description: local", "{type: object, required: [message], properties: {message: {type: string}}, description: local}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := sseInverseSource([]string{"200", "201"}, "                      "+tc.local+"\n", sseAsyncContract("['200', '201']", tc.extension))
			_, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			require.Empty(t, diagnostics)
		})
	}
}

func TestImportSSEEmptyExamplesRequireRepresentation(t *testing.T) {
	for _, inherited := range []bool{false, true} {
		name := "without inherited examples"
		if inherited {
			name = "with inherited examples"
		}
		t.Run(name, func(t *testing.T) {
			source := string(sseInverseSource([]string{"201"}, "                      examples: []\n", ""))
			if inherited {
				source = strings.Replace(source, "      required: [message]", "      example: {message: declared}\n      required: [message]", 1)
			}
			_, diagnostics, err := Analyze([]byte(source))
			require.NoError(t, err)
			require.Contains(t, diagnostics.Error(), "schema-examples-presence")
		})
	}
}

func TestImportSSEStrictBoundaries(t *testing.T) {
	for _, tc := range []struct{ name, source, code string }{
		{"reserved outside stream", `openapi: 3.2.0
info: {title: Reserved, version: '1'}
paths:
  /events:
    get:
` + sseAsyncContract("['200']", "{type: string}") + `      responses:
        '200': {description: ordinary, content: {application/json: {schema: {type: string}}}}
`, "sse-async"},
		{"generic declaration content", string(sseInverseSource([]string{"201"}, "", "")) + "    Ordinary: {type: string, contentMediaType: application/json}\n", "schema-content"},
		{"payload content keyword", strings.Replace(string(sseInverseSource([]string{"201"}, "", "")), "message: {type: string}", "message: {type: string, contentEncoding: base64}", 1), "schema-content"},
		{"bodyless status", string(sseInverseSource([]string{"204"}, "", "")), "sse-response"},
		{"inbound", strings.Replace(string(sseInverseSource([]string{"201"}, "", sseAsyncContract("['201']", "{$ref: '#/components/schemas/Event'}"))), "        messages:\n", "        messages:\n          inbound: {contentType: application/json, schema: {type: string}}\n", 1), "sse-async"},
		{"projection", strings.Replace(string(sseInverseSource([]string{"201"}, "", sseAsyncContract("['201']", "{$ref: '#/components/schemas/Event'}"))), "sse: {requestIDField:", "sse: {projections: [], requestIDField:", 1), "sse-async"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, diagnostics, err := Analyze([]byte(tc.source))
			require.NoError(t, err)
			require.Contains(t, diagnostics.Error(), tc.code)
		})
	}
}

func TestImportSSESingleAndRecursivePayload(t *testing.T) {
	for _, tc := range []struct{ name, source string }{
		{"single inline", strings.Replace(string(sseInverseSource([]string{"200"}, "", "")), "$ref: '#/components/schemas/Event'", "type: string", 1)},
		{"recursive expansion", strings.Replace(string(sseInverseSource([]string{"201"}, "", sseAsyncContract("['201']", "{type: object, required: [message], properties: {message: {type: string}, next: {$ref: '#/components/schemas/Event'}}}"))), "message: {type: string}\n", "message: {type: string}\n        next: {$ref: '#/components/schemas/Event'}\n", 1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc, diagnostics, err := Analyze([]byte(tc.source))
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			out, err := Render(doc, Options{})
			require.NoError(t, err)
			checks := ""
			if tc.name == "single inline" {
				checks = `
 if expr.Root.API.HTTP.Services[0].HTTPEndpoints[0].Responses[0].Handshake != nil {
  t.Fatal("a literal status without metadata must not invent a handshake carrier")
 }
`
			}
			requireRenderedDesignEvaluates(t, out, 1, checks)
		})
	}
}

func TestImportSSEExactValuesAndChildAnnotations(t *testing.T) {
	for _, tc := range []struct {
		name, first, second string
		valid               bool
	}{
		{"same large default", "default: 9007199254740993", "default: 9007199254740993", true},
		{"different large default", "default: 9007199254740993", "default: 9007199254740992", false},
		{"different bound", "minimum: 9007199254740993", "minimum: 9007199254740992", false},
		{"same bound", "minimum: 9007199254740993", "minimum: 9007199254740993", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := string(sseInverseSource([]string{"200", "201"}, "", ""))
			source = strings.Replace(source, "      type: object\n      required: [message]\n      properties:\n        message: {type: string}", "      type: integer", 1)
			needle := "$ref: '#/components/schemas/Event'"
			at := strings.Index(source, needle) + len(needle)
			source = source[:at] + "\n                      " + tc.first + source[at:]
			at = strings.LastIndex(source, needle) + len(needle)
			source = source[:at] + "\n                      " + tc.second + source[at:]
			_, diagnostics, err := Analyze([]byte(source))
			require.NoError(t, err)
			if tc.valid {
				require.Empty(t, diagnostics)
			} else {
				require.Contains(t, diagnostics.Error(), "sse-event")
			}
		})
	}
	for _, example := range []string{"child", "different"} {
		t.Run("child "+example, func(t *testing.T) {
			extension := sseAsyncContract("['201']", "{type: object, required: [message], properties: {message: {type: string, example: "+example+"}}}")
			source := strings.Replace(string(sseInverseSource([]string{"201"}, "", extension)), "message: {type: string}\n", "message: {type: string, example: child}\n", 1)
			_, diagnostics, err := Analyze([]byte(source))
			require.NoError(t, err)
			if example == "child" {
				require.Empty(t, diagnostics)
			} else {
				require.Contains(t, diagnostics.Error(), "sse-async")
			}
		})
	}
}

func TestImportSSEReusableResponseSource(t *testing.T) {
	entry := `openapi: 3.2.0
info: {title: Reusable, version: '1'}
paths:
  /events:
    get:
      operationId: events
` + sseAsyncContract("['201']", "{$ref: 'event.yaml#/Payload'}") + `      responses:
        '201': {$ref: 'response.yaml#/Events'}
`
	source := resolverFiles(t, []byte(entry), map[string]string{
		"response.yaml": "Events: {$ref: 'chain.yaml#/Events'}",
		"chain.yaml":    "Events:\n" + sseResolverResponse("  "),
		"event.yaml":    "Payload: {type: object, properties: {message: {type: string}}}",
	})
	doc, diagnostics, _, err := AnalyzeSelected(source, Selection{})
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	out, err := Render(doc, Options{})
	require.NoError(t, err)
	requireRenderedDesignEvaluates(t, out, 1, "")
}

func TestImportSSEExplicitNullExample(t *testing.T) {
	source := string(sseInverseSource([]string{"201"}, "                      example: null\n", ""))
	source = strings.Replace(source, "      type: object\n      required: [message]\n      properties:\n        message: {type: string}", "      type: [string, 'null']", 1)
	document, diagnostics, err := Analyze([]byte(source))
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	rendered, err := Render(document, Options{})
	require.NoError(t, err)
	requireRenderedDesignEvaluates(t, rendered, 1, `
	value := expr.Root.Services[0].Methods[0].StreamingResult
	if len(value.UserExamples) == 0 || value.UserExamples[0].Value != nil {
		t.Fatal("explicit null event example was discarded")
	}
`)
}

func TestImportSSEOrderAndPartialProjection(t *testing.T) {
	extension := sseAsyncContract("['200', '201']", "{$ref: '#/components/schemas/Event'}")
	var expected []byte
	for _, keys := range [][]string{{"200", "201"}, {"201", "200"}} {
		t.Run(strings.Join(keys, "_"), func(t *testing.T) {
			input := sseInverseSource(keys, "", extension)
			document, diagnostics, err := Analyze(input)
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			rendered, err := Render(document, Options{})
			require.NoError(t, err)
			if expected == nil {
				expected = rendered
			} else {
				require.Equal(t, string(expected), string(rendered))
			}
			partial, _, err := AnalyzePartial(Source{Bytes: input}, Selection{}, false)
			require.NoError(t, err)
			require.Empty(t, partial.Blocked)
			require.Empty(t, partial.Skipped)
			require.Empty(t, partial.Warnings)
			projected, err := Render(partial.Document, Options{})
			require.NoError(t, err)
			require.Equal(t, string(rendered), string(projected))
		})
	}
}

func TestImportSSEReferenceFormatContract(t *testing.T) {
	for _, tc := range []struct {
		name, message string
		valid         bool
	}{
		{"same local format", "{$ref: '#/components/schemas/Event', format: uuid}", true},
		{"expanded declaration loses local format", "{type: string, format: email}", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := string(sseInverseSource([]string{"201"}, "                      format: uuid\n", sseAsyncContract("['201']", tc.message)))
			input = strings.Replace(input, "      type: object\n      required: [message]\n      properties:\n        message: {type: string}", "      type: string\n      format: email", 1)
			_, diagnostics, err := Analyze([]byte(input))
			require.NoError(t, err)
			if tc.valid {
				require.Empty(t, diagnostics)
			} else {
				require.Contains(t, diagnostics.Error(), "sse-async")
			}
		})
	}
}

func TestImportSSEContractDiagnostics(t *testing.T) {
	valid := sseAsyncContract("['201']", "{$ref: '#/components/schemas/Event'}")
	for _, tc := range []struct {
		name, extension, contains string
		regenerate                bool
	}{
		{"payload mismatch cites event", strings.Replace(valid, "{$ref: '#/components/schemas/Event'}", "{type: integer}", 1), "/responses/201/", false},
		{"current handshake mismatch", strings.Replace(valid, "contentType: text/event-stream", "contentType: application/json", 1), "handshake response", false},
		{"legacy scalar", strings.Replace(valid, "statuses: ['201']", "status: 201", 1), "scalar status", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, diagnostics, err := Analyze(sseInverseSource([]string{"201"}, "", tc.extension))
			require.NoError(t, err)
			require.Contains(t, diagnostics.Error(), "sse-async")
			require.Contains(t, diagnostics.Error(), tc.contains)
			if tc.regenerate {
				require.Contains(t, diagnostics.Error(), "regenerat")
			} else {
				require.NotContains(t, diagnostics.Error(), "regenerat")
			}
		})
	}
}
