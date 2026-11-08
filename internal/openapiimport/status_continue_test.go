package openapiimport

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImportContinueDeclaration(t *testing.T) {
	for _, headers := range []bool{false, true} {
		t.Run(fmt.Sprint(headers), func(t *testing.T) {
			metadata := ""
			if headers {
				metadata = `, headers: {Trace: {description: continuation metadata, schema: {type: string}}}`
			}
			source := []byte(fmt.Sprintf(`openapi: 3.2.0
info: {title: Continue, version: "1"}
paths:
  /read:
    get:
      operationId: read
      responses:
        "100": {description: automatic continuation%s}
        "204": {description: done}
`, metadata))
			doc, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			rendered, err := Render(doc, Options{})
			require.NoError(t, err)
			require.Contains(t, string(rendered), `Informational("ReadStatus100", 100`)
			require.Contains(t, string(rendered), `Description("automatic continuation")`)
			if headers {
				require.Contains(t, string(rendered), `continuation metadata`)
			}
			requireRenderedDesignEvaluates(t, rendered, 1, `
				endpoint := expr.Root.API.HTTP.Services[0].HTTPEndpoints[0]
				if len(endpoint.Informational) != 1 || endpoint.Informational[0].Status.Code != 100 {
					t.Error("exact100 declaration lost")
				}
				if len(endpoint.Informational[0].StatusDomain()) != 0 {
					t.Error("exact100 emission unexpectedly admitted")
				}
			`)
		})
	}
}

func TestImportContinueContentRejected(t *testing.T) {
	source := `openapi: 3.2.0
info: {title: Continue, version: "1"}
paths:
  /read:
    get:
      responses:
        "100": {description: continue, content: {application/json: {schema: {type: string}}}}
        "204": {description: done}
`
	_, diagnostics, err := Analyze([]byte(source))
	require.NoError(t, err)
	require.Contains(t, strings.ToLower(diagnostics.Error()), "1xx responses cannot contain content")
}
