package openapi

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/CaliLuke/loom/expr"
)

func TestMetadataHelperContracts(t *testing.T) {
	t.Run("generation defaults on and accepts explicit suppression", func(t *testing.T) {
		require.True(t, MustGenerate(nil))
		require.True(t, MustGenerate(expr.MetaExpr{"openapi:generate": {"true"}}))
		require.False(t, MustGenerate(expr.MetaExpr{"openapi:generate": {"false"}}))
	})

	t.Run("additional properties recognizes only explicit false", func(t *testing.T) {
		require.Nil(t, AdditionalPropertiesFromExpr(nil))
		require.Equal(t, false, AdditionalPropertiesFromExpr(expr.MetaExpr{"openapi:additionalProperties": {"false"}}))
		require.Nil(t, AdditionalPropertiesFromExpr(expr.MetaExpr{"openapi:additionalProperties": {"true"}}))
	})

	t.Run("closed object mode requires explicit true", func(t *testing.T) {
		require.False(t, ClosedObjectModeFromExpr(nil))
		require.True(t, ClosedObjectModeFromExpr(expr.MetaExpr{"openapi:closed-objects": {"true"}}))
		require.False(t, ClosedObjectModeFromExpr(expr.MetaExpr{"openapi:closed-objects": {"false"}}))
	})
}

func TestDocumentationAndExtensionHelperContracts(t *testing.T) {
	require.Nil(t, DocsFromExpr(nil))
	docs := DocsFromExpr(&expr.DocsExpr{Description: "Reference", URL: "https://example.com/docs"})
	require.Equal(t, &ExternalDocs{
		Description: "Reference",
		URL:         "https://example.com/docs",
	}, docs)

	require.Nil(t, MergeExtensions(nil, nil))
	require.Equal(t, map[string]any{
		"x-first":  "one",
		"x-shared": "second",
	}, MergeExtensions(
		map[string]any{"x-first": "one", "x-shared": "first"},
		map[string]any{"x-shared": "second"},
	))
}

func TestTagContract(t *testing.T) {
	meta := expr.MetaExpr{
		"openapi:tag:billing:desc":                   {"Billing operations"},
		"openapi:tag:billing:summary":                {"Billing"},
		"openapi:tag:billing:parent":                 {"commerce"},
		"openapi:tag:billing:kind":                   {"nav"},
		"openapi:tag:billing:url":                    {"https://example.com/billing"},
		"openapi:tag:billing:url:desc":               {"Billing guide"},
		"openapi:tag:billing:extension:x-visibility": {`"public"`},
	}

	tags := TagsFromExpr(meta)
	require.Len(t, tags, 1)
	tag := tags[0]
	require.Equal(t, "billing", tag.Name)
	require.Equal(t, "Billing", tag.Summary)
	require.Equal(t, "Billing operations", tag.Description)
	require.Equal(t, "commerce", tag.Parent)
	require.Equal(t, "nav", tag.Kind)
	require.Equal(t, &ExternalDocs{
		Description: "Billing guide",
		URL:         "https://example.com/billing",
	}, tag.ExternalDocs)
	require.Equal(t, map[string]any{"x-visibility": jsontext.Value(`"public"`)}, tag.Extensions)
	require.Equal(t, []string{"billing"}, TagNamesFromExpr(meta))

	encodedJSON, err := json.Marshal(tag)
	require.NoError(t, err)
	require.Contains(t, string(encodedJSON), `"x-visibility":"public"`)

	encodedYAML, err := yaml.Marshal(tag)
	require.NoError(t, err)
	require.Contains(t, string(encodedYAML), "x-visibility: public")
}
