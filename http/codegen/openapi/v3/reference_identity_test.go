package openapiv3

import (
	"testing"

	"github.com/CaliLuke/loom/http/codegen/openapi"
	"github.com/stretchr/testify/require"
)

func TestEscapedComponentReachabilityWitness(t *testing.T) {
	for _, tc := range []struct {
		name string
		ref  string
	}{
		{"Plain", "#/components/schemas/Plain"},
		{"ItemName", "#/components/schemas/%49temName"},
		{"Item_Name", "#/components/schemas/Item%5FName"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			paths := map[string]*PathItem{
				"/value": {Get: &Operation{Responses: map[string]*ResponseRef{
					"200": {Value: &Response{Content: map[string]*MediaType{
						"application/json": {Schema: &openapi.Schema{Ref: tc.ref}},
					}}},
				}}},
			}
			schemas := map[string]*openapi.Schema{tc.name: {Type: openapi.String}}
			pruned := pruneUnusedComponentSchemas(paths, schemas, reusableComponents{})
			require.Contains(t, pruned, tc.name, "reachable component must survive pruning")
		})
	}
}

func TestEncodedAliasReferenceIdentity(t *testing.T) {
	schemas := map[string]*openapi.Schema{
		"Alias":  {Ref: "#/components/schemas/%54arget"},
		"Target": {Type: openapi.String},
		"Unused": {Type: openapi.String},
	}
	schema := &openapi.Schema{Ref: "#/components/schemas/%41lias"}
	paths := map[string]*PathItem{"/value": {Get: &Operation{Responses: map[string]*ResponseRef{
		"200": {Value: &Response{Content: map[string]*MediaType{"application/json": {Schema: schema}}}},
	}}}}
	collapseSchemaAliases(paths, schemas, reusableComponents{})
	require.Equal(t, "#/components/schemas/Target", schema.Ref)
	require.Equal(t, map[string]*openapi.Schema{"Target": {Type: openapi.String}}, pruneUnusedComponentSchemas(paths, schemas, reusableComponents{}))
}

func TestEncodedSchemaPropertyIdentity(t *testing.T) {
	property := &openapi.Schema{Type: openapi.String}
	components := &Components{Schemas: map[string]*openapi.Schema{
		"ItemName": {Properties: map[string]*openapi.Schema{"value": property}},
	}}
	for _, ref := range []string{"#/components/schemas/ItemName", "#/components/schemas/%49temName", "#%2Fcomponents%2Fschemas%2FItemName"} {
		require.Same(t, property, schemaProperty(&openapi.Schema{Ref: ref}, "value", components))
	}
	require.Nil(t, schemaProperty(&openapi.Schema{Ref: "#/components/schemas/Missing"}, "value", components))
	require.Nil(t, schemaProperty(&openapi.Schema{Ref: "#/components/schemas/%ZZ"}, "value", components))
}
