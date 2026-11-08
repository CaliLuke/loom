package openapiimport

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImportStatusResponseBodyRoles(t *testing.T) {
	const raw = `{description: bytes, content: {application/octet-stream: {schema: {type: string, format: binary, description: RawPayload}}}}`
	const typed = `{description: json, content: {application/json: {schema: {type: object, properties: {value: {type: string}}, required: [value]}, example: {value: retained}}}}`
	const empty = `{description: empty}`
	for _, tc := range []struct {
		name, responses string
		raw             bool
	}{
		{"mixed", "'200': " + typed + "\n        '201': " + raw, true},
		{"mixed reverse", "'201': " + raw + "\n        '200': " + typed, true},
		{"raw bodyless", "'200': " + raw + "\n        '204': " + empty, true},
		{"raw bodyless reverse", "'204': " + empty + "\n        '200': " + raw, true},
		{"all raw", "'200': " + raw + "\n        '201': " + raw, true},
		{"all encoded", "'200': " + typed + "\n        '201': " + typed, false},
		{"encoded bodyless", "'200': " + typed + "\n        '204': " + empty, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte("openapi: 3.2.0\ninfo: {title: Body roles, version: '1'}\npaths:\n  /value:\n    post:\n      operationId: value\n      responses:\n        " + tc.responses + "\n")
			document, diags, err := Analyze(source)
			require.NoError(t, err)
			require.Empty(t, diags)
			rendered, err := Render(document, Options{})
			require.NoError(t, err)
			if tc.raw {
				require.Contains(t, string(rendered), "SkipResponseBodyEncodeDecode()")
				require.NotContains(t, string(rendered), "\n\t\t\t\tBody(")
			}
			if strings.Contains(tc.responses, "RawPayload") {
				require.Contains(t, string(rendered), `Description("RawPayload")`)
			}
			if strings.Contains(tc.responses, "retained") {
				require.Contains(t, string(rendered), `"retained"`)
			}
			requireRenderedDesignEvaluates(t, rendered, 1, fmt.Sprintf(`

	e := expr.Root.API.HTTP.Services[0].HTTPEndpoints[0]
	if e.SkipResponseBodyEncodeDecode != %t {
		t.Error("wrong endpoint body mode")
	}
	if e.SkipResponseBodyEncodeDecode {
		if len(*expr.AsObject(e.MethodExpr.Result.Type)) != 1 {
			t.Error("raw result retains body fields")
		}
		for _, r := range e.Responses {
			if r.Body.Type != expr.Empty {
				t.Error("raw success has typed body")
			}
			if r.Status.Code == 204 {
				if r.OpenAPIBody != nil {
					t.Error("bodyless gained documentation")
				}
			} else if r.OpenAPIBody == nil {
				t.Error("raw schema lost")
			}
			if r.Status.Code == 200 && r.ContentType == "application/json" && len(r.OpenAPIBody.UserExamples) != 1 {
				t.Error("raw JSON occurrence example lost")
			}
		}
	}

`, tc.raw))
		})
	}
}

func TestImportRawSuccessErrorBodyRoles(t *testing.T) {
	for _, tc := range []struct {
		name, content, source string
		rejected              bool
	}{
		{"single binary", "application/octet-stream: {schema: {type: string, format: binary}}", "", true},
		{"media union", "application/json: {schema: {type: object, properties: {reason: {type: string}}}}, application/octet-stream: {schema: {type: string, format: binary}}", "", false},
		{"typed JSON", "application/json: {schema: {type: object, properties: {reason: {type: string}}}}", "", false},
		{"single PDF exact error", "", `openapi: 3.1.1
info: {title: Binary media, version: "1.0"}
paths:
  /signatures:
    post:
      operationId: attachSignature
      responses:
        "200":
          description: signed document
          content:
            application/pdf:
              schema: {type: string, format: binary}
        "400":
          description: invalid signature
          content:
            application/pdf:
              schema: {type: string, format: binary}
`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte("openapi: 3.2.0\ninfo: {title: Error roles, version: '1'}\npaths:\n  /value:\n    post:\n      responses:\n        '200': {description: bytes, content: {application/octet-stream: {schema: {type: string, format: binary}}}}\n        '4XX': {description: failure, content: {" + tc.content + "}}\n")
			if tc.source != "" {
				source = []byte(tc.source)
			}
			document, diags, err := Analyze(source)
			require.NoError(t, err)
			if tc.rejected {
				require.Contains(t, diags.Error(), "encoded error body")
				return
			}
			require.Empty(t, diags)
			rendered, err := Render(document, Options{})
			require.NoError(t, err)
			requireRenderedDesignEvaluates(t, rendered, 1, `

	e := expr.Root.API.HTTP.Services[0].HTTPEndpoints[0]
	if len(e.HTTPErrors) != 1 {
		t.Fatal("error mapping lost")
	}
	r := e.HTTPErrors[0].Response
	if r.Body == nil || r.Body.Type == expr.Empty {
		t.Error("error body lost")
	}
	if r.OpenAPIBody != nil {
		t.Error("encoded error replaced by documentation")
	}

`)
		})
	}
}

func TestImportRawSuccessMultipleMediaBoundary(t *testing.T) {
	const multiple = `{description: choices, content: {application/json: {schema: {type: string}}, application/octet-stream: {schema: {type: string, format: binary}}}}`
	const raw = `{description: raw, content: {application/octet-stream: {schema: {type: string, format: binary}}}}`
	const encoded = `{description: json, content: {application/json: {schema: {type: string}}}}`
	for _, tc := range []struct {
		name, other string
		rejected    bool
	}{
		{"raw endpoint", raw, true},
		{"encoded endpoint", encoded, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := []byte("openapi: 3.2.0\ninfo: {title: Media roles, version: '1'}\npaths:\n  /value:\n    post:\n      responses:\n        '200': " + multiple + "\n        '201': " + tc.other + "\n")
			document, diagnostics, err := Analyze(input)
			require.NoError(t, err)
			if tc.rejected {
				// If planning regresses, retain the emitted contract in the failure log.
				if len(diagnostics) == 0 {
					rendered, renderErr := Render(document, Options{})
					require.NoError(t, renderErr)
					t.Log(string(rendered))
					requireRenderedDesignEvaluates(t, rendered, 1)
				}
				require.Contains(t, diagnostics.Error(), "multiple media contracts on a raw successful response")
				return
			}
			require.Empty(t, diagnostics)
			rendered, err := Render(document, Options{})
			require.NoError(t, err)
			requireRenderedDesignEvaluates(t, rendered, 1)
		})
	}
}
