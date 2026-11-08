package openapiv3

import (
	"maps"
	"slices"
	"testing"

	"github.com/CaliLuke/loom/http/codegen/openapi"
	"github.com/stretchr/testify/require"
)

func TestDiscriminatorFallbackSurvivesAliasCleanupAndPruning(t *testing.T) {
	nested := &openapi.Schema{Discriminator: &openapi.Discriminator{PropertyName: "kind", DefaultMapping: "#/components/schemas/Alias"}}
	schemas := map[string]*openapi.Schema{
		"Root":     {Properties: map[string]*openapi.Schema{"nested": {Items: nested}}},
		"Alias":    {Ref: "#/components/schemas/Fallback"},
		"Fallback": {Type: openapi.Object},
		"Unused":   {Type: openapi.String},
	}
	paths := map[string]*PathItem{"/": {Get: &Operation{Responses: map[string]*ResponseRef{"200": {Value: &Response{Content: map[string]*MediaType{"application/json": {Schema: &openapi.Schema{Ref: "#/components/schemas/Root"}}}}}}}}}
	collapseSchemaAliases(paths, schemas, reusableComponents{})
	require.Equal(t, "#/components/schemas/Fallback", nested.Discriminator.DefaultMapping)
	pruned := pruneUnusedComponentSchemas(paths, schemas, reusableComponents{}, nil)
	require.Contains(t, pruned, "Fallback")
	require.NotContains(t, pruned, "Alias")
	require.NotContains(t, pruned, "Unused")
}

func TestDiscriminatorMappingAndChildrenSurviveAliasCleanupAndPruning(t *testing.T) {
	hint := &openapi.Discriminator{PropertyName: "kind", Mapping: map[string]string{"alias": "#/components/schemas/Alias", "mapped": "#/components/schemas/Mapped"}}
	schemas := map[string]*openapi.Schema{
		"Parent":  {Type: openapi.Object, Discriminator: hint},
		"Alias":   {Ref: "#/components/schemas/Target"},
		"Target":  {Type: openapi.Object},
		"Mapped":  {Type: openapi.Object},
		"Child":   {AllOf: []*openapi.Schema{{Ref: "#/components/schemas/Parent"}, {Type: openapi.Object}}},
		"Orphan":  {AllOf: []*openapi.Schema{{Ref: "#/components/schemas/Unused"}, {Type: openapi.Object}}},
		"Unused":  {Type: openapi.Object, Discriminator: &openapi.Discriminator{PropertyName: "kind"}},
		"Sibling": {AllOf: []*openapi.Schema{{Ref: "#/components/schemas/Parent"}, {Type: openapi.Object}}},
	}
	children := map[string][]string{"Parent": {"Child"}, "Unused": {"Orphan"}}
	paths := map[string]*PathItem{"/": {Get: &Operation{Responses: map[string]*ResponseRef{"200": {Value: &Response{Content: map[string]*MediaType{"application/json": {Schema: &openapi.Schema{Ref: "#/components/schemas/Parent"}}}}}}}}}
	collapseSchemaAliases(paths, schemas, reusableComponents{})
	require.Equal(t, map[string]string{"alias": "#/components/schemas/Target", "mapped": "#/components/schemas/Mapped"}, hint.Mapping)
	pruned := pruneUnusedComponentSchemas(paths, schemas, reusableComponents{}, children)
	require.ElementsMatch(t, []string{"Parent", "Target", "Mapped", "Child"}, slices.Collect(maps.Keys(pruned)))
}
