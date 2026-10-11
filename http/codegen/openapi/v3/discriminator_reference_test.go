package openapiv3

import (
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
	pruned := pruneUnusedComponentSchemas(paths, schemas, reusableComponents{})
	require.Contains(t, pruned, "Fallback")
	require.NotContains(t, pruned, "Alias")
	require.NotContains(t, pruned, "Unused")
}
