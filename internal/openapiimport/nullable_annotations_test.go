package openapiimport

import (
	"encoding/json/v2"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/libopenapi/datamodel/high/base"
)

func TestNullableAnyOfAnnotationOwnership(t *testing.T) {
	tests := []struct {
		name       string
		outer      map[string]any
		branch     map[string]any
		wantTitle  string
		readOnly   bool
		writeOnly  bool
		deprecated bool
		extensions map[string]any
	}{
		{name: "outer read annotations", outer: map[string]any{"title": "Read value", "readOnly": true, "deprecated": true}, branch: map[string]any{"type": "string"}, wantTitle: "Read value", readOnly: true, deprecated: true},
		{name: "outer write annotations", outer: map[string]any{"title": "Write value", "writeOnly": true}, branch: map[string]any{"type": "string"}, wantTitle: "Write value", writeOnly: true},
		{name: "branch annotations", outer: map[string]any{}, branch: map[string]any{"type": "string", "title": "Branch", "readOnly": true, "deprecated": true}, wantTitle: "Branch", readOnly: true, deprecated: true},
		{name: "outer precedence", outer: map[string]any{"title": "Outer", "readOnly": false, "deprecated": false}, branch: map[string]any{"type": "string", "title": "Branch", "readOnly": true, "deprecated": true}, wantTitle: "Outer", readOnly: true, deprecated: true},
		{name: "unconstrained", outer: map[string]any{"title": "Anything", "deprecated": true}, branch: map[string]any{}, wantTitle: "Anything", deprecated: true},
		{name: "outer flags", outer: map[string]any{"readOnly": true, "deprecated": true}, branch: map[string]any{"type": "string"}, readOnly: true, deprecated: true},
		{name: "nested nullable annotations", outer: map[string]any{"title": "Outer"}, branch: map[string]any{"title": "Nested", "writeOnly": true, "anyOf": []any{map[string]any{"type": "string", "deprecated": true}, map[string]any{"type": "null"}}}, wantTitle: "Outer", writeOnly: true, deprecated: true},
		{name: "reference branch", outer: map[string]any{"title": "Referenced", "deprecated": true}, branch: map[string]any{"$ref": "#/components/schemas/Status"}, wantTitle: "Referenced", deprecated: true},
		{name: "extension ownership", outer: map[string]any{"x-shared": "outer", "x-outer": true}, branch: map[string]any{"type": "string", "x-shared": "branch", "x-branch": true}, extensions: map[string]any{"x-shared": "outer", "x-outer": true, "x-branch": true}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var source map[string]any
			require.NoError(t, json.Unmarshal(readFixture(t, "supported.json"), &source))
			schemas := source["components"].(map[string]any)["schemas"].(map[string]any)
			tc.outer["anyOf"] = []any{tc.branch, map[string]any{"type": "null"}}
			schemas["Pet"].(map[string]any)["properties"].(map[string]any)["annotation"] = tc.outer
			encoded, err := json.Marshal(source, json.Deterministic(true))
			require.NoError(t, err)
			document, diagnostics, err := Analyze(encoded)
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			var got *Schema
			for _, named := range document.Components.Schemas {
				if named.Name != "Pet" {
					continue
				}
				for _, property := range named.Schema.Properties {
					if property.Name == "annotation" {
						got = property.Schema
					}
				}
			}
			require.NotNil(t, got)
			require.Equal(t, tc.wantTitle, got.Title)
			require.Equal(t, tc.readOnly, got.ReadOnly)
			require.Equal(t, tc.writeOnly, got.WriteOnly)
			require.Equal(t, tc.deprecated, got.Deprecated)
			require.Equal(t, tc.extensions, got.Extensions)
			rendered, err := Render(document, Options{PackageName: "design"})
			require.NoError(t, err)
			if tc.wantTitle != "" {
				require.Equal(t, 1, strings.Count(string(rendered), `Title("`+tc.wantTitle+`")`))
			}
		})
	}
}

func TestSchemaAnnotationLayerIsolation(t *testing.T) {
	branch := &Schema{Title: "Branch", Description: "Branch description", ReadOnly: true, Extensions: map[string]any{"x-branch": "original"}}
	selected := *branch
	source := &base.Schema{Title: "Outer", Description: "Outer description"}
	a := &analyzer{}
	a.schemaAnnotations(&selected, source, "#/schema")
	require.Equal(t, "Outer", selected.Title)
	require.Equal(t, "Outer description", selected.Description)
	require.True(t, selected.ReadOnly)
	selected.Extensions["x-branch"] = "changed"
	require.Equal(t, "Branch", branch.Title)
	require.Equal(t, "Branch description", branch.Description)
	require.Equal(t, "original", branch.Extensions["x-branch"])
}
