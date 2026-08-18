package openapiimport

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/stretchr/testify/require"
)

type grammarDisposition string

const (
	grammarPreserved   grammarDisposition = "preserved"
	grammarConditional grammarDisposition = "conditional"
	grammarLossy       grammarDisposition = "lossy"
	grammarRejected    grammarDisposition = "rejected"
	grammarLibrary     grammarDisposition = "library-only"
)

type grammarFieldGroup struct {
	disposition grammarDisposition
	evidence    string
	fields      []string
}

type grammarObjectCoverage struct {
	name              string
	model             any
	groups            []grammarFieldGroup
	parserGapFields   []string
	parserGapEvidence string
}

func TestOpenAPIGrammarCoverageLedgerIsExhaustive(t *testing.T) {
	for _, object := range openAPIGrammarCoverage() {
		t.Run(object.name, func(t *testing.T) {
			covered := make(map[string]grammarFieldGroup)
			for _, group := range object.groups {
				require.NotEmpty(t, group.evidence)
				for _, field := range group.fields {
					_, duplicate := covered[field]
					require.False(t, duplicate, "field %s has multiple grammar classifications", field)
					covered[field] = group
				}
			}
			if len(object.parserGapFields) > 0 {
				require.NotEmpty(t, object.parserGapEvidence)
			}

			typeOf := reflect.TypeOf(object.model)
			var missing []string
			for index := 0; index < typeOf.NumField(); index++ {
				field := typeOf.Field(index)
				if !field.IsExported() {
					continue
				}
				if _, ok := covered[field.Name]; !ok {
					missing = append(missing, field.Name)
				}
				delete(covered, field.Name)
			}
			sort.Strings(missing)
			require.Empty(t, missing, "new parser fields need an explicit preserved, conditional, lossy, rejected, or library-only classification")

			unknown := make([]string, 0, len(covered))
			for field := range covered {
				unknown = append(unknown, field)
			}
			sort.Strings(unknown)
			require.Empty(t, unknown, "coverage ledger names fields no longer exposed by the parser")
		})
	}
}

func TestSchemaGrammarKeywordsAreNeverSilentlyIgnored(t *testing.T) {
	cases := map[string]string{
		"$anchor":               `$anchor: subject`,
		"$comment":              `$comment: retained nowhere`,
		"$defs":                 `$defs: {nested: {type: string}}`,
		"$dynamicAnchor":        `$dynamicAnchor: subject`,
		"$dynamicRef":           `$dynamicRef: '#subject'`,
		"$id":                   `$id: 'https://example.com/subject'`,
		"$schema":               `$schema: 'https://json-schema.org/draft/2020-12/schema'`,
		"$vocabulary":           `$vocabulary: {'https://json-schema.org/draft/2020-12/vocab/core': true}`,
		"const":                 `const: fixed`,
		"contains":              `contains: {type: string}`,
		"contentEncoding":       `contentEncoding: base64`,
		"contentMediaType":      `contentMediaType: application/json`,
		"contentSchema":         `contentSchema: {type: object}`,
		"dependentRequired":     `dependentRequired: {credit_card: [billing_address]}`,
		"dependentSchemas":      `dependentSchemas: {credit_card: {type: object}}`,
		"discriminator":         `type: object\ndiscriminator: {propertyName: kind}`,
		"else":                  `else: {type: string}`,
		"externalDocs":          `externalDocs: {url: 'https://example.com/schema'}`,
		"if":                    `if: {type: string}`,
		"maxContains":           `maxContains: 2`,
		"maxProperties":         `type: object\nmaxProperties: 2`,
		"minContains":           `minContains: 1`,
		"minProperties":         `type: object\nminProperties: 1`,
		"multipleOf":            `type: number\nmultipleOf: 0.5`,
		"not":                   `not: {type: string}`,
		"oneOf":                 `oneOf: [{type: string}, {type: integer}]`,
		"patternProperties":     `type: object\npatternProperties: {'^x-': {type: string}}`,
		"prefixItems":           `type: array\nprefixItems: [{type: string}]`,
		"propertyNames":         `type: object\npropertyNames: {pattern: '^[a-z]+$'}`,
		"then":                  `then: {type: string}`,
		"unevaluatedItems":      `type: array\nunevaluatedItems: false`,
		"unevaluatedProperties": `type: object\nunevaluatedProperties: false`,
		"uniqueItems":           `type: array\nitems: {type: string}\nuniqueItems: true`,
		"xml":                   `type: string\nxml: {name: subject}`,
	}

	names := make([]string, 0, len(cases))
	for name := range cases {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			source := []byte(fmt.Sprintf(`openapi: 3.1.1
info: {title: Grammar, version: "1"}
paths:
  /subject:
    get:
      operationId: getSubject
      responses:
        "200":
          description: subject
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Subject'}
components:
  schemas:
    Subject:
%s
`, indentYAML(cases[name], 6)))

			_, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			require.NotEmpty(t, diagnostics, "%s must be preserved or rejected explicitly", name)
		})
	}
}

func TestSuccessResponseClassificationMatrix(t *testing.T) {
	tests := []struct {
		name      string
		statuses  []string
		wantClass byte
		wantCount int
	}{
		{name: "one 2xx", statuses: []string{"200", "400"}, wantClass: '2', wantCount: 1},
		{name: "redirect only", statuses: []string{"302", "400"}, wantClass: '3', wantCount: 1},
		{name: "2xx preferred over redirect", statuses: []string{"200", "302"}, wantClass: '2', wantCount: 1},
		{name: "multiple 2xx", statuses: []string{"200", "204", "302"}, wantClass: '2', wantCount: 2},
		{name: "multiple redirect", statuses: []string{"301", "302", "400"}, wantClass: '3', wantCount: 2},
		{name: "no success", statuses: []string{"400", "500"}, wantClass: '3', wantCount: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			responses := make([]StatusResponse, len(test.statuses))
			for index, status := range test.statuses {
				responses[index].Status = status
			}
			class := successResponseClass(responses)
			require.Equal(t, test.wantClass, class)
			count := 0
			for _, response := range responses {
				if isSuccessResponseStatus(response.Status, class) {
					count++
				}
			}
			require.Equal(t, test.wantCount, count)
		})
	}
}

func TestSingleReferenceAllOfCoversEveryRenderableSchemaKind(t *testing.T) {
	tests := map[string]string{
		"array":        "type: array\n      items: {type: string}",
		"boolean":      "type: boolean",
		"integer enum": "type: integer\n      enum: [1, 2]",
		"number":       "type: number",
		"object":       "type: object\n      properties: {id: {type: string}}",
		"string":       "type: string",
	}

	names := make([]string, 0, len(tests))
	for name := range tests {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			source := []byte(fmt.Sprintf(`openapi: 3.1.1
info: {title: Wrapper, version: "1"}
paths:
  /value:
    get:
      operationId: getValue
      responses:
        "200":
          description: value
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Wrapped'}
components:
  schemas:
    Value:
      %s
    Wrapped:
      allOf:
        - $ref: '#/components/schemas/Value'
`, strings.ReplaceAll(tests[name], `\n`, "\n      ")))

			document, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			var wrapped *Schema
			for _, schema := range document.Components.Schemas {
				if schema.Name == "Wrapped" {
					wrapped = schema.Schema
				}
			}
			require.NotNil(t, wrapped)
			require.Equal(t, "#/components/schemas/Value", wrapped.Ref)
			_, err = Render(document, Options{PackageName: "design"})
			require.NoError(t, err)
		})
	}
}

func TestPathParameterIdentityMatrix(t *testing.T) {
	for _, name := range []string{"asset_id", "accountID", "v2_id", "_tenant"} {
		t.Run(name, func(t *testing.T) {
			source := []byte(fmt.Sprintf(`openapi: 3.1.1
info: {title: Paths, version: "1"}
paths:
  /values/{%[1]s}:
    get:
      operationId: getValue
      parameters:
        - {name: %[1]s, in: path, required: true, schema: {type: string}}
      responses: {"204": {description: found}}
`, name))

			document, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			rendered, err := Render(document, Options{PackageName: "design"})
			require.NoError(t, err)
			design := string(rendered)
			require.Contains(t, design, fmt.Sprintf(`Attribute(%q, String)`, name))
			require.Contains(t, design, fmt.Sprintf(`GET("/values/{%s}")`, name))
			require.Contains(t, design, fmt.Sprintf(`Param(%q)`, name))
		})
	}
}

func TestInlineObjectArrayPromotionCoversEverySchemaLocation(t *testing.T) {
	inlineArray := func() *Schema {
		return &Schema{Type: "array", Items: &Schema{
			Type:       "object",
			Properties: []NamedProperty{{Name: "id", Schema: &Schema{Type: "string"}}},
		}}
	}
	document := &Document{
		Components: Components{
			Schemas:       []NamedSchema{{Name: "Values", Schema: inlineArray()}},
			Parameters:    []NamedParameter{{Name: "Values", Parameter: Parameter{Schema: inlineArray()}}},
			RequestBodies: []NamedRequestBody{{Name: "Values", RequestBody: RequestBody{Schema: inlineArray()}}},
			Responses:     []NamedResponse{{Name: "Values", Response: Response{Schema: inlineArray()}}},
			Headers:       []NamedHeader{{Name: "Values", Header: Header{Schema: inlineArray()}}},
		},
		Operations: []Operation{{
			Method:      "POST",
			Path:        "/values",
			GoName:      "CreateValues",
			Parameters:  []Parameter{{In: "query", Name: "values", Schema: inlineArray()}},
			RequestBody: &RequestBody{Schema: inlineArray()},
			Responses:   []StatusResponse{{Status: "200", Response: Response{Schema: inlineArray()}}},
		}},
	}

	analyzer := analyzer{}
	analyzer.promoteInlineArrayItems(document)
	require.Len(t, analyzer.diagnostics, 8)
	for _, diagnostic := range analyzer.diagnostics {
		require.Equal(t, "schema-inline-array-item-promoted", diagnostic.Code)
	}

	roots := []*Schema{
		document.Components.Schemas[0].Schema,
		document.Components.Parameters[0].Parameter.Schema,
		document.Components.RequestBodies[0].RequestBody.Schema,
		document.Components.Responses[0].Response.Schema,
		document.Components.Headers[0].Header.Schema,
		document.Operations[0].Parameters[0].Schema,
		document.Operations[0].RequestBody.Schema,
		document.Operations[0].Responses[0].Response.Schema,
	}
	for _, schema := range roots {
		require.NotNil(t, schema.Items)
		require.True(t, strings.HasPrefix(schema.Items.Ref, "#/components/schemas/"))
	}
}

func TestParserGapMediaTypeFieldsAreDiagnosedFromSource(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		wantCode  string
		wantLossy bool
	}{
		{
			name:      "description",
			field:     "description: A JSON representation.",
			wantCode:  "media-type-description",
			wantLossy: true,
		},
		{
			name:     "prefix encoding",
			field:    "prefixEncoding: []",
			wantCode: "media-prefix-encoding",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := []byte(fmt.Sprintf(`openapi: 3.2.0
info: {title: Media, version: "1"}
paths:
  /value:
    get:
      operationId: getValue
      responses:
        "200":
          description: value
          content:
            application/json:
              %s
              schema: {type: string}
`, test.field))

			_, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			requireDiagnosticCode(t, diagnostics, test.wantCode)
			fatal, warnings := diagnostics.Classify(true)
			if test.wantLossy {
				require.Empty(t, fatal)
				require.NotEmpty(t, warnings)
				return
			}
			require.NotEmpty(t, fatal)
		})
	}
}

func TestSchemaReferenceSiblingsAreExplicitlyRejected(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: References, version: "1"}
paths:
  /value:
    get:
      operationId: getValue
      responses:
        "200":
          description: value
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Value'
                maxLength: 8
components:
  schemas:
    Value: {type: string}
`)

	_, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	requireDiagnosticCode(t, diagnostics, "schema-reference-siblings")
	fatal, _ := diagnostics.Classify(true)
	require.NotEmpty(t, fatal)
}

func TestUnknownSchemaVocabularyIsExplicitlyRejected(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Vocabulary, version: "1"}
paths:
  /value:
    get:
      operationId: getValue
      responses:
        "200":
          description: value
          content:
            application/json:
              schema:
                type: string
                customAssertion: true
                x-documentation-hint: retained
`)

	_, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	requireDiagnosticCode(t, diagnostics, "schema-keyword-unknown")
	require.Len(t, diagnostics, 1, "x-* schema extensions remain supported")
}

func indentYAML(source string, spaces int) string {
	source = strings.ReplaceAll(source, `\n`, "\n")
	prefix := strings.Repeat(" ", spaces)
	return prefix + strings.ReplaceAll(source, "\n", "\n"+prefix)
}

func openAPIGrammarCoverage() []grammarObjectCoverage {
	group := func(disposition grammarDisposition, evidence string, fields ...string) grammarFieldGroup {
		return grammarFieldGroup{disposition: disposition, evidence: evidence, fields: fields}
	}
	return []grammarObjectCoverage{
		{
			name:  "OpenAPI Object",
			model: v3.Document{},
			groups: []grammarFieldGroup{
				group(grammarPreserved, "analyzer.document and TestAnalyzeJSONYAMLParity", "Version", "Info", "Paths", "Components", "Security", "Tags", "Extensions"),
				group(grammarRejected, "analyzer.document diagnostics", "Servers", "JsonSchemaDialect", "Self", "Webhooks"),
				group(grammarLossy, "external-docs diagnostic", "ExternalDocs"),
				group(grammarLibrary, "libopenapi parser state", "Index", "Rolodex"),
			},
		},
		{
			name:  "Info Object",
			model: base.Info{},
			groups: []grammarFieldGroup{
				group(grammarPreserved, "analyzer.document", "Title", "Description", "Version", "Extensions"),
				group(grammarLossy, "info-metadata diagnostic", "Summary", "TermsOfService", "Contact", "License"),
			},
		},
		{
			name:  "Tag Object",
			model: base.Tag{},
			groups: []grammarFieldGroup{
				group(grammarPreserved, "analyzer.tags and TestImportPreservesExporterMetadataSymmetry", "Name", "Summary", "Description", "Parent", "Kind", "Extensions"),
				group(grammarConditional, "analyzer.tags rejects nested external-doc extensions", "ExternalDocs"),
			},
		},
		{
			name:  "External Documentation Object",
			model: base.ExternalDoc{},
			groups: []grammarFieldGroup{
				group(grammarConditional, "analyzer.tags preserves fields and rejects nested extensions", "Description", "URL", "Extensions"),
			},
		},
		{
			name:  "Paths Object",
			model: v3.Paths{},
			groups: []grammarFieldGroup{
				group(grammarPreserved, "analyzer.operations", "PathItems"),
				group(grammarRejected, "paths extension diagnostic", "Extensions"),
			},
		},
		{
			name:  "Path Item Object",
			model: v3.PathItem{},
			groups: []grammarFieldGroup{
				group(grammarPreserved, "analyzer.operations", "Get", "Put", "Post", "Delete", "Options", "Head", "Patch", "Trace", "Query", "AdditionalOperations", "Parameters"),
				group(grammarLossy, "path-metadata diagnostic", "Description", "Summary"),
				group(grammarRejected, "path-reference, servers, and extension diagnostics", "Reference", "Servers", "Extensions"),
			},
		},
		{
			name:  "Operation Object",
			model: v3.Operation{},
			groups: []grammarFieldGroup{
				group(grammarPreserved, "analyzer.operation", "Tags", "Summary", "Description", "OperationId", "Parameters", "RequestBody", "Responses", "Deprecated", "Security", "Extensions"),
				group(grammarLossy, "external-docs diagnostic", "ExternalDocs"),
				group(grammarRejected, "callbacks and servers diagnostics", "Callbacks", "Servers"),
			},
		},
		{
			name:  "Components Object",
			model: v3.Components{},
			groups: []grammarFieldGroup{
				group(grammarConditional, "analyzer.components and render planner", "Schemas", "Responses", "Parameters", "RequestBodies", "Headers", "SecuritySchemes", "Examples"),
				group(grammarRejected, "component-kind and extension diagnostics", "Links", "Callbacks", "PathItems", "MediaTypes", "Extensions"),
			},
		},
		{
			name:  "Parameter Object",
			model: v3.Parameter{},
			groups: []grammarFieldGroup{
				group(grammarConditional, "analyzer.parameter and render planner", "Reference", "Name", "In", "Required", "AllowEmptyValue", "Style", "AllowReserved", "Schema", "Extensions"),
				group(grammarPreserved, "analyzer.parameter", "Description"),
				group(grammarLossy, "parameter-deprecated and examples diagnostics", "Deprecated", "Example", "Examples"),
				group(grammarRejected, "parameter serialization and content diagnostics", "Explode", "Content"),
			},
		},
		{
			name:  "Request Body Object",
			model: v3.RequestBody{},
			groups: []grammarFieldGroup{
				group(grammarPreserved, "analyzer.requestBody", "Description", "Content", "Required", "Extensions"),
				group(grammarRejected, "request-body-reference planner diagnostic", "Reference"),
			},
		},
		{
			name:              "Media Type Object",
			model:             v3.MediaType{},
			parserGapFields:   []string{"description", "prefixEncoding"},
			parserGapEvidence: "mediaTypeParserGapDiagnostics covers official OpenAPI 3.2 fields absent from libopenapi v0.38.7",
			groups: []grammarFieldGroup{
				group(grammarConditional, "analyzer.content and requestContent", "Schema", "Example", "Examples"),
				group(grammarRejected, "media item, encoding, and extension diagnostics", "ItemSchema", "Encoding", "ItemEncoding", "Extensions"),
			},
		},
		{
			name:  "Responses Object",
			model: v3.Responses{},
			groups: []grammarFieldGroup{
				group(grammarConditional, "analyzer.operation concrete response validation", "Codes"),
				group(grammarRejected, "default-response and extensions diagnostics", "Default", "Extensions"),
			},
		},
		{
			name:  "Response Object",
			model: v3.Response{},
			groups: []grammarFieldGroup{
				group(grammarPreserved, "analyzer.response and TestImportPreservesExporterMetadataSymmetry", "Description", "Summary", "Headers", "Content", "Extensions"),
				group(grammarRejected, "response reference and links diagnostics", "Reference", "Links"),
			},
		},
		{
			name:  "Header Object",
			model: v3.Header{},
			groups: []grammarFieldGroup{
				group(grammarPreserved, "analyzer.header", "Description", "Required", "Schema"),
				group(grammarConditional, "analyzer.header and TestImportPreservesExporterMetadataSymmetry", "Style", "AllowReserved"),
				group(grammarLossy, "header-deprecated and examples diagnostics", "Deprecated", "Example", "Examples"),
				group(grammarRejected, "header reference, serialization, content, and extension diagnostics", "Reference", "AllowEmptyValue", "Explode", "Content", "Extensions"),
			},
		},
		{
			name:  "Security Scheme Object",
			model: v3.SecurityScheme{},
			groups: []grammarFieldGroup{
				group(grammarConditional, "analyzer.securityScheme and TestImportPreservesRepresentableSecuritySchemes", "Reference", "Type", "Description", "Name", "In", "Scheme", "BearerFormat", "Flows", "OpenIdConnectUrl", "OAuth2MetadataUrl", "Deprecated", "Extensions"),
			},
		},
		{
			name:              "OAuth Flows Object",
			model:             v3.OAuthFlows{},
			parserGapFields:   []string{"deviceAuthorization"},
			parserGapEvidence: "oauthDeviceAuthorizationPresent rejects the official OpenAPI 3.2 field that libopenapi v0.38.7 does not expose correctly",
			groups: []grammarFieldGroup{
				group(grammarConditional, "analyzer.oauthFlows", "Implicit", "Password", "ClientCredentials", "AuthorizationCode"),
				group(grammarRejected, "device authorization and extension diagnostics", "Device", "Extensions"),
			},
		},
		{
			name:              "OAuth Flow Object",
			model:             v3.OAuthFlow{},
			parserGapFields:   []string{"deviceAuthorizationUrl"},
			parserGapEvidence: "device authorization is rejected from the raw OAuth Flows Object before URL fields can be discarded",
			groups: []grammarFieldGroup{
				group(grammarConditional, "analyzer.validOAuthFlow and analyzer.oauthFlows", "AuthorizationUrl", "TokenUrl", "RefreshUrl", "Scopes"),
				group(grammarRejected, "OAuth flow extension diagnostic", "Extensions"),
			},
		},
		{
			name:  "Example Object",
			model: base.Example{},
			groups: []grammarFieldGroup{
				group(grammarConditional, "analyzer.mediaExamples and TestImportPreservesStructuredAndReusableExamples", "Reference", "Summary", "Description", "Value", "ExternalValue", "DataValue", "SerializedValue", "Extensions"),
			},
		},
		{
			name:  "Schema Object",
			model: base.Schema{},
			groups: []grammarFieldGroup{
				group(grammarPreserved, "analyzer.schema normalization", "ExclusiveMaximum", "ExclusiveMinimum", "Properties", "Title", "Maximum", "Minimum", "MaxLength", "MinLength", "Pattern", "MaxItems", "MinItems", "Required", "Description", "Nullable", "ReadOnly", "WriteOnly", "Deprecated", "Extensions"),
				group(grammarConditional, "schema normalization and render planning", "Type", "AllOf", "AnyOf", "OneOf", "Examples", "Items", "Format", "Enum", "AdditionalProperties", "Default", "Example"),
				group(grammarRejected, "schemaUnsupportedKeywords", "SchemaTypeRef", "Discriminator", "PrefixItems", "Contains", "MinContains", "MaxContains", "If", "Else", "Then", "DependentSchemas", "DependentRequired", "PatternProperties", "Defs", "PropertyNames", "UnevaluatedItems", "UnevaluatedProperties", "Id", "Anchor", "DynamicAnchor", "DynamicRef", "Comment", "ContentSchema", "Vocabulary", "Not", "MultipleOf", "UniqueItems", "MaxProperties", "MinProperties", "ContentEncoding", "ContentMediaType", "Const", "XML", "ExternalDocs"),
				group(grammarLibrary, "libopenapi schema proxy state", "ParentProxy"),
			},
		},
	}
}
