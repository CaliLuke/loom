package main

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/internal/openapiimport"
)

func TestComparisonSchemaOverlayOwnership(t *testing.T) {
	for _, tc := range []struct {
		name      string
		base      *openapiimport.Schema
		local     *openapiimport.Schema
		wantError bool
	}{
		{"disjoint default", &openapiimport.Schema{Type: "string"}, &openapiimport.Schema{Default: &openapiimport.SchemaDefault{Value: "value"}}, false},
		{"matching description", &openapiimport.Schema{Type: "string", Description: "same"}, &openapiimport.Schema{Description: "same"}, false},
		{"conflicting description", &openapiimport.Schema{Type: "string", Description: "base"}, &openapiimport.Schema{Description: "local"}, true},
		{"conflicting enum", &openapiimport.Schema{Type: "string", Enum: []any{"one", "two"}}, &openapiimport.Schema{Enum: []any{"two"}}, true},
		{"matching structured annotations", &openapiimport.Schema{Type: "string", Extensions: map[string]any{"x-data": map[string]any{"a": "one", "b": "two"}}}, &openapiimport.Schema{Extensions: map[string]any{"x-data": map[string]any{"b": "two", "a": "one"}}}, false},
		{"vendor annotation", &openapiimport.Schema{Type: "string"}, &openapiimport.Schema{Extensions: map[string]any{"x-local": "value"}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.local.Ref = "#/components/schemas/Value"
			baseBefore, err := cloneOpenAPIContractValue(tc.base)
			require.NoError(t, err)
			localBefore, err := cloneOpenAPIContractValue(tc.local)
			require.NoError(t, err)
			r := newComparisonSchemaResolver(&openapiimport.Document{}, &openapiimport.Document{
				Components: openapiimport.Components{Schemas: []openapiimport.NamedSchema{{Name: "Value", Schema: tc.base}}},
			})
			got, err := r.resolve(tc.local, make(map[string]bool))
			if tc.wantError {
				require.ErrorContains(t, err, "conflicting")
			} else {
				require.NoError(t, err)
				require.Equal(t, "string", got.Type)
				local := reflect.ValueOf(tc.local).Elem()
				result := reflect.ValueOf(got).Elem()
				for index := 0; index < local.NumField(); index++ {
					field := local.Field(index)
					if field.CanInterface() && !field.IsZero() && local.Type().Field(index).Name != "Ref" {
						require.Equal(t, field.Interface(), result.Field(index).Interface())
					}
				}
				got.Description = "mutation"
				if got.Default != nil {
					got.Default.Value = "mutation"
				}
			}
			require.Equal(t, baseBefore, tc.base)
			require.Equal(t, localBefore, tc.local)
		})
	}
}

func TestComparisonSchemaResolutionRejectsMissingAndCyclicTargets(t *testing.T) {
	for _, cyclic := range []bool{false, true} {
		document := &openapiimport.Document{}
		if cyclic {
			document.Components.Schemas = []openapiimport.NamedSchema{{Name: "Loop", Schema: &openapiimport.Schema{Ref: "#/components/schemas/Loop"}}}
		}
		r := newComparisonSchemaResolver(&openapiimport.Document{}, document)
		_, err := r.resolve(&openapiimport.Schema{Ref: "#/components/schemas/Loop"}, make(map[string]bool))
		require.Error(t, err)
	}
}

func TestComparisonSchemaPruningRetainsAuthoredAndNestedTargets(t *testing.T) {
	for _, authored := range []bool{false, true} {
		expected := &openapiimport.Document{}
		document := &openapiimport.Document{Components: openapiimport.Components{Schemas: []openapiimport.NamedSchema{
			{Name: "Value", Schema: &openapiimport.Schema{Type: "string"}},
			{Name: "Nested", Schema: &openapiimport.Schema{Type: "object", Properties: []openapiimport.NamedProperty{{Name: "value", Schema: &openapiimport.Schema{Ref: "#/components/schemas/Value"}}}}},
		}}}
		if authored {
			document.Components.Schemas = document.Components.Schemas[:1]
			expected.Components.Schemas = document.Components.Schemas
		}
		r := newComparisonSchemaResolver(expected, document)
		_, err := r.resolve(&openapiimport.Schema{Ref: "#/components/schemas/Value"}, make(map[string]bool))
		require.NoError(t, err)
		r.prune(document)
		require.Equal(t, "Value", document.Components.Schemas[0].Name)
	}
}

func TestComparisonDuplicateExamplesKeepUniqueAnnotations(t *testing.T) {
	for _, unique := range []bool{false, true} {
		expected := &openapiimport.Response{Schema: &openapiimport.Schema{Type: "string"}}
		actual := &openapiimport.Response{Schema: &openapiimport.Schema{Type: "string", Examples: []openapiimport.Example{{Name: "example-1", Value: "same"}}}, Examples: []openapiimport.Example{{Name: "named", Value: "same"}}}
		if unique {
			actual.Schema.Examples[0].Description = "unique annotation"
		}
		removeDuplicatedResponseSchemaExamples(expected, actual)
		require.Equal(t, unique, len(actual.Schema.Examples) > 0)
	}
}

func TestComparisonSharedReferencesAreOrderIndependent(t *testing.T) {
	for _, paths := range [][]string{{"/first", "/second"}, {"/second", "/first"}} {
		expected := &openapiimport.Document{}
		for _, path := range paths {
			expected.Operations = append(expected.Operations, openapiimport.Operation{Path: path, Responses: []openapiimport.StatusResponse{{Status: "200", Response: openapiimport.Response{Schema: &openapiimport.Schema{Type: "string"}}}}})
		}
		actual := cloneOpenAPIContract(t, expected)
		actual.Components.Schemas = []openapiimport.NamedSchema{{Name: "Shared/Value", Schema: &openapiimport.Schema{Type: "string"}}}
		for index := range actual.Operations {
			actual.Operations[index].Responses[0].Response.Schema = &openapiimport.Schema{Ref: "#/components/schemas/Shared~1Value"}
		}
		before := cloneOpenAPIContract(t, actual)
		require.NoError(t, compareOpenAPIContracts(expected, actual))
		require.Equal(t, before, actual)
		normalized, err := normalizeGeneratedOpenAPIContract(expected, actual)
		require.NoError(t, err)
		repeated, err := normalizeGeneratedOpenAPIContract(expected, normalized)
		require.NoError(t, err)
		require.Equal(t, normalized, repeated)
	}
}
