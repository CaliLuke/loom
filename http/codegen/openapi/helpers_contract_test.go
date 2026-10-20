package openapi

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"testing"

	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"

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

func TestTagContractPreservesColonAndLiteralEscapeNames(t *testing.T) {
	meta := expr.MetaExpr{
		"openapi:tag-id:YmlsbGluZzpkZXNj":                                    {},
		"openapi:tag-id:YmlsbGluZzpkZXNj:desc":                               {"Billing"},
		"openapi:tag-id:b3BzOnBhcmVudDpsaXRlcmFsJQ:parent":                   {"root"},
		"openapi:tag-id:b3BzOnBhcmVudDpsaXRlcmFsJQ:extension:x-visible:mode": {`true`},
		"openapi:tag:literal%3Aname":                                         {},
		"openapi:tag:ordinary":                                               {},
	}

	tags := TagsFromExpr(meta)
	require.Len(t, tags, 4)
	require.Equal(t, "billing:desc", tags[0].Name)
	require.Equal(t, "Billing", tags[0].Description)
	require.Equal(t, "ops:parent:literal%", tags[1].Name)
	require.Equal(t, "root", tags[1].Parent)
	require.Equal(t, map[string]any{"x-visible:mode": jsontext.Value(`true`)}, tags[1].Extensions)
	require.Equal(t, "literal%3Aname", tags[2].Name)
	require.Equal(t, "ordinary", tags[3].Name)
	require.Equal(t, []string{"billing:desc", "ops:parent:literal%", "literal%3Aname", "ordinary"}, TagNamesFromExpr(meta))
}

func TestTagContractAllowsEmptyIdentityAndSkipsMalformedValuesWithoutPanic(t *testing.T) {
	meta := expr.MetaExpr{
		"openapi:tag:":              {},
		"openapi:tag:ordinary:desc": {},
	}

	tags := TagsFromExpr(meta)

	require.Len(t, tags, 2)
	require.Equal(t, "", tags[0].Name)
	require.Equal(t, "ordinary", tags[1].Name)
	require.Empty(t, tags[1].Description)
}
