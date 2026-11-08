package openapiimport

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImportStatusSelectorRawResponsesAvoidFileOwnership(t *testing.T) {
	source := []byte(`openapi: 3.2.0
info: {title: File ownership, version: "1"}
paths:
  /class:
    get:
      operationId: class
      responses:
        '200': {description: bytes, content: {application/octet-stream: {schema: {type: string, format: binary}}}}
        '4XX': {description: failure}
  /fallback:
    get:
      operationId: fallback
      responses:
        '200': {description: bytes, content: {application/octet-stream: {schema: {type: string, format: binary}}}}
        default: {description: failure}
  /progress:
    get:
      operationId: progress
      responses:
        '200': {description: bytes, content: {application/octet-stream: {schema: {type: string, format: binary}}}}
        '1XX': {description: informational}
  /exact:
    get:
      operationId: exact
      responses:
        '200': {description: bytes, content: {application/octet-stream: {schema: {type: string, format: binary}}}}
        '404': {description: missing}
`)
	doc, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	rendered, err := Render(doc, Options{})
	require.NoError(t, err)
	requireRenderedDesignEvaluates(t, rendered, 4, `
 for _, e := range expr.Root.API.HTTP.Services[0].HTTPEndpoints {
  if e.MethodExpr.Name == "Exact" {
   if !e.FileResponse || e.SkipResponseBodyEncodeDecode { t.Error("exact-only file contract changed") }
  } else if e.FileResponse || !e.SkipResponseBodyEncodeDecode { t.Error("non-exact selector entered file protocol ownership") }
 }
 `)
}
