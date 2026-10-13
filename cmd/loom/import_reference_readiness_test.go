package main

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/internal/openapiimport"
)

func TestGeneratedResponseInliningPreservesOccurrenceContract(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(*openapiimport.Schema)
	}{
		{"example", func(schema *openapiimport.Schema) {
			schema.Examples = []openapiimport.Example{{Value: "different"}}
		}},
		{"description", func(schema *openapiimport.Schema) {
			schema.Description = "local occurrence description"
		}},
		{"enum", func(schema *openapiimport.Schema) {
			schema.Enum = []any{"only-this-value"}
		}},
		{"maximum length", func(schema *openapiimport.Schema) {
			limit := int64(1)
			schema.MaxLength = &limit
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expected := &openapiimport.Document{OpenAPIVersion: "3.2.0", Title: "Comparison", Operations: []openapiimport.Operation{{
				Method: "GET", Path: "/value", Responses: []openapiimport.StatusResponse{{
					Status: "200", Response: openapiimport.Response{Schema: &openapiimport.Schema{Type: "string"}},
				}},
			}}}
			actual := cloneOpenAPIContract(t, expected)
			actual.Components.Schemas = []openapiimport.NamedSchema{{Name: "Generated", GoName: "Generated", Schema: &openapiimport.Schema{Type: "string"}}}
			actual.Operations[0].Responses[0].Response.Schema = &openapiimport.Schema{Ref: "#/components/schemas/Generated"}
			require.NoError(t, compareOpenAPIContracts(expected, actual), "pure generated references are equivalent")
			tc.edit(actual.Operations[0].Responses[0].Response.Schema)
			require.Error(t, compareOpenAPIContracts(expected, actual), "normalizing a generated component must not erase the occurrence contract")
		})
	}
}

func TestGeneratedResponseInliningRetainsSharedReferenceTarget(t *testing.T) {
	expected := &openapiimport.Document{OpenAPIVersion: "3.2.0", Title: "Shared comparison"}
	for _, path := range []string{"/first", "/second"} {
		expected.Operations = append(expected.Operations, openapiimport.Operation{
			Method: "GET", Path: path, Responses: []openapiimport.StatusResponse{{
				Status: "200", Response: openapiimport.Response{Schema: &openapiimport.Schema{Type: "string"}},
			}},
		})
	}
	actual := cloneOpenAPIContract(t, expected)
	actual.Components.Schemas = []openapiimport.NamedSchema{{Name: "Generated", GoName: "Generated", Schema: &openapiimport.Schema{Type: "string"}}}
	for index := range actual.Operations {
		actual.Operations[index].Responses[0].Response.Schema = &openapiimport.Schema{Ref: "#/components/schemas/Generated"}
	}
	require.NoError(t, compareOpenAPIContracts(expected, actual), "each response occurrence must resolve before an unreferenced generated component is removed")
}

func TestGeneratedDocumentPreparationPreservesNumericTokens(t *testing.T) {
	source := []byte(`{
		"openapi":"3.2.0","info":{"title":"Exact","version":"1"},
		"jsonSchemaDialect":"https://spec.openapis.org/oas/3.1/dialect/base",
		"servers":[{"url":"http://localhost:80","description":"Default server for Exact","name":"Exact"}],
		"components":{"schemas":{"Count":{"type":"integer","maximum":18446744073709551615}}},
		"paths":{}
	}`)
	normalized, err := prepareGeneratedOpenAPIContract(source, "Exact")
	require.NoError(t, err)
	require.Contains(t, string(normalized), `"maximum":18446744073709551615`, "preparing generated input must preserve exact contract numbers")
}

func TestComparisonLiteralNumbersRemainDistinct(t *testing.T) {
	for _, tc := range []struct {
		name string
		set  func(*openapiimport.Schema, uint64)
	}{
		{"default", func(schema *openapiimport.Schema, value uint64) {
			schema.Default = &openapiimport.SchemaDefault{Value: value}
		}},
		{"enum", func(schema *openapiimport.Schema, value uint64) {
			schema.Enum = []any{value}
		}},
		{"example", func(schema *openapiimport.Schema, value uint64) {
			schema.Examples = []openapiimport.Example{{Value: value}}
		}},
		{"nested extension", func(schema *openapiimport.Schema, value uint64) {
			schema.Extensions = map[string]any{"x-data": map[string]any{"values": []any{value}}}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			schema := &openapiimport.Schema{Type: "integer"}
			tc.set(schema, 18446744073709551615)
			expected := &openapiimport.Document{Operations: []openapiimport.Operation{{
				Responses: []openapiimport.StatusResponse{{Response: openapiimport.Response{Schema: schema}}},
			}}}
			actual := cloneOpenAPIContract(t, expected)
			require.NoError(t, compareOpenAPIContracts(expected, actual))
			tc.set(actual.Operations[0].Responses[0].Response.Schema, 18446744073709551614)
			require.Error(t, compareOpenAPIContracts(expected, actual))
		})
	}
}
