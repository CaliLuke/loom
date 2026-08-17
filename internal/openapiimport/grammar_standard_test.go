package openapiimport

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestOfficialOpenAPIGrammarFieldsHaveParserRepresentation compares the fixed
// fields from the published OpenAPI 3.0.4, 3.1.1, and 3.2.0 schemas with the
// parser models used by the importer. Fields that the parser does not expose
// need an explicit raw-source guard in the grammar coverage ledger.
func TestOfficialOpenAPIGrammarFieldsHaveParserRepresentation(t *testing.T) {
	coverage := make(map[string]grammarObjectCoverage)
	for _, object := range openAPIGrammarCoverage() {
		coverage[object.name] = object
	}

	for objectName, standardFields := range officialOpenAPIFixedFields() {
		t.Run(objectName, func(t *testing.T) {
			object, ok := coverage[objectName]
			require.True(t, ok, "official object needs a grammar coverage entry")

			available := parserJSONFields(object.model)
			for _, field := range object.parserGapFields {
				_, exposed := available[field]
				require.False(t, exposed, "parser gap %s is now exposed and should move into the field ledger", field)
				available[field] = struct{}{}
			}

			var missing []string
			for _, field := range standardFields {
				if _, ok := available[field]; !ok {
					missing = append(missing, field)
				}
			}
			sort.Strings(missing)
			require.Empty(t, missing, "official fields need parser representation or an explicit raw-source guard")
		})
	}
}

func parserJSONFields(model any) map[string]struct{} {
	typeOf := reflect.TypeOf(model)
	fields := make(map[string]struct{}, typeOf.NumField())
	for index := 0; index < typeOf.NumField(); index++ {
		field := strings.Split(typeOf.Field(index).Tag.Get("json"), ",")[0]
		if field == "" || field == "-" {
			continue
		}
		fields[field] = struct{}{}
	}
	return fields
}

// officialOpenAPIFixedFields is the union of fixed fields in the official
// OpenAPI 3.0.4, 3.1.1, and 3.2.0 schemas. Dynamic map keys, specification
// extensions, and Schema Object vocabulary are covered by dedicated tests.
func officialOpenAPIFixedFields() map[string][]string {
	return map[string][]string{
		"OpenAPI Object": {
			"$self", "components", "externalDocs", "info", "jsonSchemaDialect",
			"openapi", "paths", "security", "servers", "tags", "webhooks",
		},
		"Info Object": {
			"contact", "description", "license", "summary", "termsOfService", "title", "version",
		},
		"Tag Object": {
			"description", "externalDocs", "kind", "name", "parent", "summary",
		},
		"External Documentation Object": {
			"description", "url",
		},
		"Path Item Object": {
			"$ref", "additionalOperations", "delete", "description", "get", "head", "options",
			"parameters", "patch", "post", "put", "query", "servers", "summary", "trace",
		},
		"Operation Object": {
			"callbacks", "deprecated", "description", "externalDocs", "operationId", "parameters",
			"requestBody", "responses", "security", "servers", "summary", "tags",
		},
		"Components Object": {
			"callbacks", "examples", "headers", "links", "mediaTypes", "parameters", "pathItems",
			"requestBodies", "responses", "schemas", "securitySchemes",
		},
		"Parameter Object": {
			"allowEmptyValue", "allowReserved", "content", "deprecated", "description", "example",
			"examples", "explode", "in", "name", "required", "schema", "style",
		},
		"Request Body Object": {
			"content", "description", "required",
		},
		"Media Type Object": {
			"description", "encoding", "example", "examples", "itemEncoding", "itemSchema",
			"prefixEncoding", "schema",
		},
		"Responses Object": {
			"default",
		},
		"Response Object": {
			"content", "description", "headers", "links", "summary",
		},
		"Header Object": {
			"allowEmptyValue", "allowReserved", "content", "deprecated", "description", "example",
			"examples", "explode", "required", "schema", "style",
		},
		"Security Scheme Object": {
			"bearerFormat", "deprecated", "description", "flows", "in", "name",
			"oauth2MetadataUrl", "openIdConnectUrl", "scheme", "type",
		},
		"OAuth Flows Object": {
			"authorizationCode", "clientCredentials", "deviceAuthorization", "implicit", "password",
		},
		"OAuth Flow Object": {
			"authorizationUrl", "deviceAuthorizationUrl", "refreshUrl", "scopes", "tokenUrl",
		},
		"Example Object": {
			"dataValue", "description", "externalValue", "serializedValue", "summary", "value",
		},
	}
}
