package openapiimport

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImportParameterCredentialNamespace(t *testing.T) {
	for _, location := range []string{"query", "cookie"} {
		for _, property := range []string{"token", "other"} {
			t.Run(location+"/"+property, func(t *testing.T) {
				style := "form"
				if location == "cookie" {
					style = "cookie"
				}
				source := []byte(fmt.Sprintf(`openapi: 3.2.0
info: {title: credentials, version: '1'}
security: [{Key: []}]
components:
  securitySchemes:
    Key: {type: apiKey, in: %s, name: token}
paths:
  /values:
    get:
      parameters:
        - name: fields
          in: %s
          style: %s
          explode: true
          schema: {type: object, properties: {%s: {type: string}}, additionalProperties: false}
      responses: {'204': {description: done}}
`, location, location, style, property))
				_, diagnostics, err := Analyze(source)
				require.NoError(t, err)
				if property == "other" {
					require.Empty(t, diagnostics)
				} else {
					require.Contains(t, diagnostics.Error(), "parameter-namespace")
				}
			})
		}
	}
}

func TestImportMediaFormNamespaces(t *testing.T) {
	source := []byte(`openapi: 3.2.0
info: {title: media, version: '1'}
paths:
  /values:
    post:
      requestBody:
        content:
          application/json:
            schema: {type: string}
          application/x-www-form-urlencoded:
            schema:
              type: object
              properties:
                left: {type: object, additionalProperties: {type: string}}
                right: {type: object, additionalProperties: {type: string}}
            encoding:
              left: {style: form}
              right: {style: form}
      responses: {'204': {description: done}}
`)
	_, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Contains(t, diagnostics.Error(), "overlapping parameter namespaces")
	partial, _, err := AnalyzePartial(Source{Bytes: source}, Selection{}, true)
	require.NoError(t, err)
	require.Empty(t, partial.Document.Operations)
}
