package openapiimport

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSelectionMatchesOperations(t *testing.T) {
	tests := []struct {
		name      string
		selection Selection
		path      string
		tags      []string
		want      bool
	}{
		{name: "no filters", want: true},
		{name: "tag", selection: Selection{Tags: []string{"Face capture"}}, tags: []string{"Face capture"}, want: true},
		{
			name:      "path prefix",
			selection: Selection{PathPrefixes: []string{"/omni/b2b"}},
			path:      "/omni/b2b/v1/session",
			want:      true,
		},
		{
			name:      "path pattern",
			selection: Selection{Paths: []string{"/omni/*/device-*"}},
			path:      "/omni/v1/device-info",
			want:      true,
		},
		{
			name: "union",
			selection: Selection{
				Tags:         []string{"Face"},
				PathPrefixes: []string{"/identity"},
			},
			path: "/identity/1",
			tags: []string{"Other"},
			want: true,
		},
		{name: "no match", selection: Selection{Tags: []string{"Face"}}, path: "/identity/1", tags: []string{"Other"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, test.selection.matches(test.path, test.tags))
		})
	}
}

func TestAnalyzeSelectedPrunesComponentClosure(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Selection, version: "1"}
tags:
  - {name: Face}
  - {name: Other, description: Unselected metadata}
  - {name: Unused declared tag, description: Unselected metadata}
paths:
  /face:
    get:
      tags: [Face]
      parameters:
        - {$ref: '#/components/parameters/Trace'}
      responses:
        "200":
          description: done
          content:
            application/json:
              schema: {$ref: '#/components/schemas/FaceResponse'}
  /other:
    get:
      tags: [Other]
      callbacks:
        ignored: {}
      responses: {"204": {description: done}}
components:
  securitySchemes:
    unused: {type: apiKey, in: header, name: X-Key}
  parameters:
    Trace:
      name: X-Trace-ID
      in: header
      schema: {type: string}
    Unused:
      name: unused
      in: query
      schema: {type: string}
  schemas:
    FaceResponse:
      type: object
      properties:
        leaf: {$ref: '#/components/schemas/Leaf'}
    Leaf: {type: string}
    Unused:
      oneOf:
        - {type: string}
        - {type: integer}
`)

	document, diagnostics, report, err := AnalyzeSelected(source, Selection{Tags: []string{"Face"}})
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	require.Len(t, document.Operations, 1)
	require.Equal(t, "/face", document.Operations[0].Path)
	require.Equal(t, []string{"FaceResponse", "Leaf"}, []string{
		document.Components.Schemas[0].Name,
		document.Components.Schemas[1].Name,
	})
	require.Equal(t, "Trace", document.Components.Parameters[0].Name)
	require.Equal(t, []TagSummary{
		{Name: "Face", Operations: 1, Paths: 1},
		{Name: "Other", Operations: 1, Paths: 1},
		{Name: "Unused declared tag", Operations: 0, Paths: 0},
	}, report.Tags)
	require.Equal(t, []string{"/other"}, report.UnclaimedPaths)
}

func TestAnalyzeSelectedRetainsTagParentClosure(t *testing.T) {
	source := []byte(`openapi: 3.2.0
info: {title: Tag hierarchy, version: "1"}
tags:
  - name: catalog
    description: Catalog operations.
  - name: products
    parent: catalog
    summary: Product operations
  - name: featured
    parent: products
    description: Featured product operations.
  - name: admin
    description: Unselected operations.
paths:
  /products:
    get:
      tags: [featured]
      responses: {"204": {description: done}}
  /admin:
    get:
      tags: [admin]
      responses: {"204": {description: done}}
`)

	for _, selection := range []Selection{
		{Tags: []string{"featured"}},
		{Paths: []string{"/products"}},
	} {
		document, diagnostics, _, err := AnalyzeSelected(source, selection)
		require.NoError(t, err)
		require.Empty(t, diagnostics)
		require.Len(t, document.Operations, 1)
		require.Equal(t, []string{"catalog", "products", "featured"}, document.Tags)
		require.Equal(t, []Tag{
			{Name: "catalog", Description: "Catalog operations."},
			{Name: "products", Summary: "Product operations", Parent: "catalog"},
			{Name: "featured", Description: "Featured product operations.", Parent: "products"},
		}, document.TagMetadata)
	}
}

func TestAnalyzePartialRetainsTagAncestorsAfterSkippingOperation(t *testing.T) {
	source := []byte(`openapi: 3.2.0
info: {title: Partial tag hierarchy, version: "1"}
tags:
  - {name: catalog, description: Catalog operations.}
  - {name: products, parent: catalog}
  - {name: featured, parent: products}
  - {name: admin, description: Unselected operations.}
paths:
  /products:
    get:
      tags: [featured]
      responses: {"204": {description: done}}
  /admin:
    get:
      tags: [admin]
      callbacks: {unsupported: {}}
      responses: {"204": {description: done}}
`)

	analysis, _, err := AnalyzePartial(source, Selection{PathPrefixes: []string{"/"}}, false)
	require.NoError(t, err)
	require.Len(t, analysis.Document.Operations, 1)
	require.Len(t, analysis.Skipped, 1)
	require.Equal(t, "/admin", analysis.Skipped[0].Path)
	require.Equal(t, []string{"catalog", "products", "featured"}, analysis.Document.Tags)
	require.Equal(t, "catalog", analysis.Document.TagMetadata[1].Parent)
	require.Equal(t, "products", analysis.Document.TagMetadata[2].Parent)
}

func TestPruneComponentsTagParentTraversalStopsAtMissingAndCyclicParents(t *testing.T) {
	document := &Document{
		Operations: []Operation{{Tags: []string{"missing-parent", "cycle-a"}}},
		Tags:       []string{"missing-parent", "cycle-a", "cycle-b", "unrelated"},
		TagMetadata: []Tag{
			{Name: "missing-parent", Parent: "absent"},
			{Name: "cycle-a", Parent: "cycle-b"},
			{Name: "cycle-b", Parent: "cycle-a"},
			{Name: "unrelated"},
		},
	}

	pruneComponents(document)
	require.Equal(t, []string{"missing-parent", "cycle-a", "cycle-b"}, document.Tags)
	require.Len(t, document.TagMetadata, 3)
}

func TestAnalyzeSelectedReassignsSchemaNamesAfterPruning(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Selection names, version: "1"}
paths:
  /selected:
    get:
      tags: [Selected]
      responses:
        "200":
          description: done
          content:
            application/json:
              schema: {$ref: '#/components/schemas/pet_id'}
components:
  schemas:
    pet-id: {type: string}
    pet_id: {type: string}
`)

	document, diagnostics, _, err := AnalyzeSelected(source, Selection{Tags: []string{"Selected"}})
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	require.Len(t, document.Components.Schemas, 1)
	require.Equal(t, "pet_id", document.Components.Schemas[0].Name)
	require.Equal(t, "PetID", document.Components.Schemas[0].GoName)
}

func TestSelectionRejectsInvalidPathPattern(t *testing.T) {
	err := (Selection{Paths: []string{"["}}).Validate()
	require.ErrorContains(t, err, `invalid OpenAPI path pattern "["`)
}
