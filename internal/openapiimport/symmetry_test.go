package openapiimport

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImportPreservesExporterMetadataSymmetry(t *testing.T) {
	source := []byte(`openapi: 3.2.0
info: {title: Symmetry, version: "1"}
tags:
  - name: root
  - name: catalog
    summary: Catalog
    description: Catalog operations.
    parent: root
    kind: navigation
    externalDocs:
      url: https://example.com/catalog
      description: Catalog guide.
    x-color: blue
paths:
  /items:
    get:
      operationId: listItems
      tags: [catalog]
      parameters:
        - name: q
          in: query
          style: form
          allowReserved: true
          schema: {type: string}
        - name: session
          in: cookie
          style: cookie
          schema: {type: string}
      responses:
        "200":
          summary: Item list
          description: Items.
          headers:
            X-Cursor:
              description: Cursor.
              allowReserved: true
              schema: {type: string}
`)

	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	require.Len(t, document.Tags, 2)
	require.Equal(t, "Catalog", document.TagMetadata[1].Summary)
	require.Equal(t, "Item list", document.Operations[0].Responses[0].Response.Summary)
	require.True(t, document.Operations[0].Parameters[0].AllowReserved)
	require.Equal(t, "form", document.Operations[0].Parameters[0].Style)
	require.True(t, document.Operations[0].Responses[0].Response.Headers[0].Header.AllowReserved)

	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	design := string(rendered)
	for _, want := range []string{
		`Meta("openapi:tag:catalog:summary", "Catalog")`,
		`Meta("openapi:tag:catalog:desc", "Catalog operations.")`,
		`Meta("openapi:tag:catalog:parent", "root")`,
		`Meta("openapi:tag:catalog:kind", "navigation")`,
		`Meta("openapi:tag:catalog:url", "https://example.com/catalog")`,
		`Meta("openapi:tag:catalog:url:desc", "Catalog guide.")`,
		`Meta("openapi:tag:catalog:extension:x-color", "\"blue\"")`,
		`Meta("openapi:style", "form")`,
		`Meta("openapi:style", "cookie")`,
		`Meta("openapi:allowReserved", "true")`,
		`Meta("openapi:summary", "Item list")`,
	} {
		require.Contains(t, design, want)
	}
	requireRenderedDesignEvaluates(t, rendered, 1)
}

func TestImportPreservesRepresentableSecuritySchemes(t *testing.T) {
	source := []byte(`openapi: 3.2.0
info: {title: Security symmetry, version: "1"}
security:
  - Basic: []
  - Bearer: []
  - OAuth: [read]
paths:
  /secure:
    get:
      operationId: secure
      responses: {"204": {description: done}}
components:
  securitySchemes:
    Basic:
      type: http
      scheme: basic
      description: User credentials.
    Bearer:
      type: http
      scheme: bearer
      deprecated: true
    OAuth:
      type: oauth2
      oauth2MetadataUrl: https://auth.example.com/.well-known/oauth-authorization-server
      flows:
        authorizationCode:
          authorizationUrl: https://auth.example.com/authorize
          tokenUrl: https://auth.example.com/token
          refreshUrl: https://auth.example.com/refresh
          scopes: {read: Read access.}
        clientCredentials:
          tokenUrl: https://auth.example.com/token
          scopes: {read: Read access.}
`)

	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	require.Len(t, document.Components.SecuritySchemes, 3)

	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	design := string(rendered)
	for _, want := range []string{
		`var ImportedBasicSecurity = BasicAuthSecurity("Basic", func() {`,
		`var ImportedBearerSecurity = JWTSecurity("Bearer", func() {`,
		`Meta("openapi:deprecated", "true")`,
		`var ImportedOAuthSecurity = OAuth2Security("OAuth", func() {`,
		`Meta("openapi:oauth2MetadataUrl", "https://auth.example.com/.well-known/oauth-authorization-server")`,
		`AuthorizationCodeFlow("https://auth.example.com/authorize", "https://auth.example.com/token", "https://auth.example.com/refresh")`,
		`ClientCredentialsFlow("https://auth.example.com/token", "")`,
		`Scope("read", "Read access.")`,
		`Security(ImportedOAuthSecurity, func() {`,
		`Scope("read")`,
		`Username("basicUsername", String)`,
		`Password("basicPassword", String)`,
		`Token("bearerToken", String)`,
		`AccessToken("oAuthAccessToken", String)`,
	} {
		require.Contains(t, design, want)
	}
	requireRenderedDesignEvaluates(t, rendered, 1)
}

func TestImportPreservesStructuredAndReusableExamples(t *testing.T) {
	source := []byte(`openapi: 3.2.0
info: {title: Example symmetry, version: "1"}
paths:
  /events:
    get:
      operationId: listEvents
      responses:
        "200":
          description: Events.
          content:
            application/json:
              schema:
                type: object
                required: [id]
                properties: {id: {type: string}}
              examples:
                shared:
                  $ref: '#/components/examples/Event'
                inline:
                  summary: Inline event
                  dataValue: {id: evt-2}
                  serializedValue: '{"id":"evt-2"}'
components:
  examples:
    Event:
      summary: Shared event
      description: A reusable event.
      value: {id: evt-1}
`)

	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	examples := document.Operations[0].Responses[0].Response.Examples
	require.Len(t, examples, 2)
	require.Equal(t, "Event", examples[0].ComponentName)
	require.True(t, examples[1].DataValue)
	require.Equal(t, `{"id":"evt-2"}`, examples[1].SerializedValue)

	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	design := string(rendered)
	for _, want := range []string{
		`Meta("openapi:component:example", "Event")`,
		`Meta("openapi:example:dataValue")`,
		`Meta("openapi:example:serializedValue", "{\"id\":\"evt-2\"}")`,
	} {
		require.Contains(t, design, want)
	}
	requireRenderedDesignEvaluates(t, rendered, 1)
}

func TestImportKeepsNonInvertibleExporterShapesRejected(t *testing.T) {
	tests := []struct {
		name     string
		fragment string
		code     string
	}{
		{name: "parameter explode", fragment: "parameters: [{name: q, in: query, explode: true, schema: {type: string}}]", code: "parameter-serialization"},
		{name: "bearer format", fragment: "security: [{Auth: []}]\n      x-test: true", code: "security-scheme"},
		{name: "device authorization", code: "security-scheme"},
		{name: "missing OAuth token URL", code: "security-scheme"},
		{name: "external example", fragment: "x-test: true", code: "examples"},
		{name: "value with serialized value", code: "examples"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var source []byte
			switch test.name {
			case "parameter explode":
				source = []byte("openapi: 3.2.0\ninfo: {title: Reject, version: '1'}\npaths:\n  /x:\n    get:\n      operationId: x\n      " + test.fragment + "\n      responses: {'204': {description: done}}\n")
			case "bearer format":
				source = []byte("openapi: 3.2.0\ninfo: {title: Reject, version: '1'}\npaths:\n  /x:\n    get:\n      operationId: x\n      security: [{Auth: []}]\n      responses: {'204': {description: done}}\ncomponents:\n  securitySchemes:\n    Auth: {type: http, scheme: bearer, bearerFormat: JWT}\n")
			case "device authorization":
				source = []byte("openapi: 3.2.0\ninfo: {title: Reject, version: '1'}\npaths: {/x: {get: {operationId: x, responses: {'204': {description: done}}}}}\ncomponents:\n  securitySchemes:\n    Auth:\n      type: oauth2\n      flows:\n        deviceAuthorization:\n          deviceAuthorizationUrl: https://auth.example.com/device\n          tokenUrl: https://auth.example.com/token\n          scopes: {}\n")
			case "missing OAuth token URL":
				source = []byte("openapi: 3.2.0\ninfo: {title: Reject, version: '1'}\npaths: {/x: {get: {operationId: x, responses: {'204': {description: done}}}}}\ncomponents:\n  securitySchemes:\n    Auth:\n      type: oauth2\n      flows:\n        clientCredentials:\n          scopes: {}\n")
			case "external example":
				source = []byte("openapi: 3.2.0\ninfo: {title: Reject, version: '1'}\npaths:\n  /x:\n    get:\n      operationId: x\n      responses:\n        '200':\n          description: x\n          content:\n            application/json:\n              schema: {type: string}\n              examples: {external: {externalValue: 'https://example.com/value'}}\n")
			case "value with serialized value":
				source = []byte("openapi: 3.2.0\ninfo: {title: Reject, version: '1'}\npaths:\n  /x:\n    get:\n      operationId: x\n      responses:\n        '200':\n          description: x\n          content:\n            text/plain:\n              schema: {type: string}\n              examples: {bad: {value: hello, serializedValue: hello}}\n")
			}
			_, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			requireDiagnosticCode(t, diagnostics, test.code)
		})
	}
}

func TestImportRejectsOpenAPI32FieldsInEarlierDocuments(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Version boundary, version: "1"}
tags:
  - name: items
    summary: Items
paths:
  /items:
    get:
      operationId: items
      parameters:
        - name: session
          in: cookie
          style: cookie
          allowReserved: true
          schema: {type: string}
      responses:
        "200":
          summary: Items
          description: Items.
          headers:
            X-Cursor:
              allowReserved: true
              schema: {type: string}
          content:
            application/json:
              schema: {type: string}
              examples:
                item:
                  dataValue: item-1
                  serializedValue: item-1
components:
  securitySchemes:
    Bearer:
      type: http
      scheme: bearer
      deprecated: true
`)

	_, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	requireDiagnosticCode(t, diagnostics, "versioned-field")
	require.GreaterOrEqual(t, len(diagnostics), 6)
}
