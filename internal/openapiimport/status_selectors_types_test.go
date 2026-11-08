package openapiimport

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImportStatusSelectorCarriers(t *testing.T) {
	source := []byte(`openapi: 3.2.0
info: {title: Native carriers, version: "1"}
paths:
  /types:
    get:
      operationId: types
      responses:
        '200': {description: ordinary, content: {application/json: {schema: {$ref: '#/components/schemas/Problem'}}}}
        '4XX': {description: union, content: {application/json: {schema: {$ref: '#/components/schemas/Variant'}}}}
        default: {description: any, content: {application/json: {schema: {$ref: '#/components/schemas/Anything'}}}}
  /problem:
    get:
      operationId: problem
      responses:
        '2XX': {description: empty}
        '4XX':
          description: problem
          headers:
            HTTP_Status: {required: true, schema: {type: string}}
            Body: {schema: {type: string}}
          content:
            application/json: {schema: {$ref: '#/components/schemas/Problem'}}
  /progress:
    get:
      operationId: progress
      responses:
        '1XX':
          description: progress
          headers:
            HTTP_Status: {schema: {type: string}}
        '204': {description: no content}
components:
  schemas:
    ProgressStatus1XX: {type: string}
    Variant:
      oneOf: [{type: string}, {type: integer}]
    Anything:
      description: Authored unconstrained schema
    Problem:
      type: object
      properties:
        http_status: {type: string}
        body: {type: string}
        detail: {type: string}
      required: [detail]
      examples: [{detail: preserved}]
`)
	doc, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	rendered, err := Render(doc, Options{})
	require.NoError(t, err)
	require.Contains(t, string(rendered), `StatusField("http_status")`)
	require.Contains(t, string(rendered), `Body("body2")`)
	require.Contains(t, string(rendered), `Informational("ProgressStatus1XX2"`)
	require.Contains(t, string(rendered), `ImportedAnything`)
	requireRenderedDesignEvaluates(t, rendered, 3, `
 for _, e := range expr.Root.API.HTTP.Services[0].HTTPEndpoints {
  for _, failure := range e.HTTPErrors {
   response := failure.Response
   if response.Status.IsExact() { continue }
   if response.StatusField != "http_status" { t.Errorf("status field = %q", response.StatusField) }
   designError := e.MethodExpr.Error(failure.Name)
   if designError == nil { t.Fatal("missing error") }
   if expr.AsObject(designError.AttributeExpr.Type) == nil { t.Error("status error is not a native object") }
   _, status := designError.AttributeExpr.FindAttribute("http_status")
   if status == nil || status.Type != expr.Int { t.Error("missing native status member") }
   if response.Body == nil || response.Body.Type == expr.Empty { t.Error("selected error body lost") }
  }
 }
 `)
}

func TestImportStatusSelectorBodyMode(t *testing.T) {
	tests := []struct {
		name     string
		statuses []string
		mode     responseBodyMode
	}{
		{"exact file", []string{"200"}, responseBodyFile},
		{"class stream", []string{"2XX"}, responseBodyStream},
		{"fallback stream", []string{"default"}, responseBodyStream},
		{"multiple stream", []string{"200", "201"}, responseBodyStream},
		{"file protocol stream", []string{"200", "416"}, responseBodyStream},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			operation := Operation{Method: "GET"}
			for _, key := range tc.statuses {
				operation.Responses = append(operation.Responses, StatusResponse{Status: key, Response: Response{ContentType: "application/octet-stream", Schema: &Schema{Type: "string", Format: "binary"}}})
			}
			plan, diagnostics := classifyOperationResponses(&operation, "#/operation", Components{})
			require.Empty(t, diagnostics)
			require.Equal(t, tc.mode, plan.bodyMode(&operation))
		})
	}
}

// Empty informational values still describe an observable, headerless 103.
func TestImportStatusSelectorHeaderlessInformational(t *testing.T) {
	source := []byte(`openapi: 3.2.0
info: {title: Headerless informational, version: "1"}
paths:
  /hints:
    get:
      responses:
        '103': {description: no headers}
        '204': {description: final}
`)
	doc, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	rendered, err := Render(doc, Options{})
	require.NoError(t, err)
	requireRenderedDesignEvaluates(t, rendered, 1)
}

func TestImportStatusSelectorMediaCarriers(t *testing.T) {
	source := []byte(`openapi: 3.2.0
info: {title: Status media, version: "1"}
paths:
  /download:
    get:
      responses:
        '2XX':
          description: native streaming bytes
          content:
            application/octet-stream:
              schema: {type: string, format: binary}
  /errors:
    get:
      responses:
        '204': {description: no content}
        '4XX':
          description: distinct source representations
          content:
            application/json:
              schema: {type: object, properties: {reason: {type: string}}, required: [reason]}
            application/octet-stream:
              schema: {type: string, format: binary}
`)
	doc, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	rendered, err := Render(doc, Options{})
	require.NoError(t, err)
	require.Contains(t, string(rendered), `SkipResponseBodyEncodeDecode()`)
	require.NotContains(t, string(rendered), `FileResponse()`)
	require.Contains(t, string(rendered), `MediaType("application/octet-stream"`)
	requireRenderedDesignEvaluates(t, rendered, 2, `
 for _, e := range expr.Root.API.HTTP.Services[0].HTTPEndpoints {
  for _, failure := range e.HTTPErrors {
   if len(failure.Response.MediaTypes) != 2 { t.Error("selected error media contracts lost") }
  }
 }
 `)
}
