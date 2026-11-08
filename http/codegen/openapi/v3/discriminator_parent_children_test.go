package openapiv3_test

import (
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	httpgen "github.com/CaliLuke/loom/http/codegen"
	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
	"github.com/stretchr/testify/require"
)

// TestDiscriminatorParentHintPublishesChildren uses the declaration shape the
// OpenAPI importer renders for an allOf child of a discriminator parent: both
// declarations carry canonical names and only the parent is used by an
// operation. Publication of the parent retains its direct children.
func TestDiscriminatorParentHintPublishesChildren(t *testing.T) {
	for _, tc := range []struct {
		name      string
		mapping   string
		reference bool
		composed  bool
		published []string
		pruned    []string
		wantHint  map[string]any
	}{
		{
			name:      "implicit allOf child",
			reference: true,
			published: []string{"ImplicitParent", "ImplicitChild"},
			pruned:    []string{"ImplicitGrandchild", "FlatChild", "Unrelated", "OtherChild"},
			wantHint:  map[string]any{"propertyName": "kind"},
		},
		{
			name:      "explicit mapping",
			mapping:   "child=ImplicitChild",
			reference: true,
			published: []string{"ImplicitParent", "ImplicitChild"},
			pruned:    []string{"ImplicitGrandchild", "FlatChild", "Unrelated", "OtherChild"},
			wantHint:  map[string]any{"propertyName": "kind", "mapping": map[string]any{"child": "#/components/schemas/ImplicitChild"}},
		},
		{
			name:      "explicit mapping to flat child",
			mapping:   "flat=FlatChild",
			reference: true,
			published: []string{"ImplicitParent", "ImplicitChild", "FlatChild"},
			pruned:    []string{"ImplicitGrandchild", "Unrelated", "OtherChild"},
			wantHint:  map[string]any{"propertyName": "kind", "mapping": map[string]any{"flat": "#/components/schemas/FlatChild"}},
		},
		{
			name:      "composed parent",
			reference: true,
			composed:  true,
			published: []string{"ParentBase", "ImplicitParent", "ImplicitChild"},
			pruned:    []string{"ImplicitGrandchild", "FlatChild", "Unrelated", "OtherChild"},
			wantHint:  map[string]any{"propertyName": "kind"},
		},
		{
			name:      "composed parent explicit mapping",
			mapping:   "child=ImplicitChild",
			reference: true,
			composed:  true,
			published: []string{"ParentBase", "ImplicitParent", "ImplicitChild"},
			pruned:    []string{"ImplicitGrandchild", "FlatChild", "Unrelated", "OtherChild"},
			wantHint:  map[string]any{"propertyName": "kind", "mapping": map[string]any{"child": "#/components/schemas/ImplicitChild"}},
		},
		{
			name:      "child without allOf reference",
			published: []string{"ImplicitParent"},
			pruned:    []string{"ImplicitChild", "ImplicitGrandchild", "FlatChild", "Unrelated", "OtherChild"},
			wantHint:  map[string]any{"propertyName": "kind"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := httpgen.RunHTTPDSL(t, discriminatorParentChildrenDSL(tc.mapping, tc.reference, tc.composed))
			for _, target := range []string{"3.1", "3.2"} {
				root.API.Meta["openapi:version"] = []string{target}
				files, err := openapiv3.Files(root)
				require.NoError(t, err)
				for _, file := range files {
					buffer := renderSection(t, file.AllSections()[0])
					raw := buffer.Bytes()
					if target == "3.1" {
						validateOpenAPIVersion(t, raw, "3.1.1")
					} else {
						validateOpenAPIVersion(t, raw, "3.2.0")
					}
					if file.Path[len(file.Path)-5:] != ".json" {
						continue
					}
					spec := decodeOpenAPIJSON(t, raw)
					schemas := requireMap(t, requireMap(t, spec["components"], "components")["schemas"], "schemas")
					for _, name := range tc.published {
						require.Contains(t, schemas, name, target)
					}
					for _, name := range tc.pruned {
						require.NotContains(t, schemas, name, target)
					}
					parent := requireMap(t, schemas["ImplicitParent"], "parent")
					if tc.composed {
						// The composed parent's own member owns the hint.
						allOf := requireSlice(t, parent["allOf"], "parent allOf")
						require.Equal(t, "#/components/schemas/ParentBase", requireMap(t, allOf[0], "parent base")["$ref"], target)
						require.NotContains(t, parent, "discriminator", target)
						parent = requireMap(t, allOf[1], "parent member")
					}
					require.Equal(t, tc.wantHint, parent["discriminator"], target)
					if tc.reference {
						child := requireMap(t, schemas["ImplicitChild"], "child")
						allOf := requireSlice(t, child["allOf"], "child allOf")
						require.Equal(t, "#/components/schemas/ImplicitParent", requireMap(t, allOf[0], "child base")["$ref"], target)
					}
				}
			}
		})
	}
}

func discriminatorParentChildrenDSL(mapping string, reference, composed bool) func() {
	return func() {
		API("parent children", func() {
			Meta("openapi:example", "false")
		})
		base := Type("ParentBase", func() {
			Meta("openapi:typename:canonical", "true")
			Attribute("base", String)
		})
		parent := Type("ImplicitParent", func() {
			Meta("openapi:typename:canonical", "true")
			if composed {
				Extend(base)
				Meta("openapi:allOf:reference", "true")
			}
			Attribute("kind", String)
			Meta("openapi:discriminator:propertyName", "kind")
			if mapping != "" {
				Meta("openapi:discriminator:mapping", mapping)
			}
			Required("kind")
		})
		child := Type("ImplicitChild", func() {
			Meta("openapi:typename:canonical", "true")
			Extend(parent)
			Attribute("name", String)
			if reference {
				Meta("openapi:allOf:reference", "true")
			}
		})
		// A grandchild extends the parent only through ImplicitChild.
		Type("ImplicitGrandchild", func() {
			Meta("openapi:typename:canonical", "true")
			Extend(child)
			Attribute("age", Int)
			Meta("openapi:allOf:reference", "true")
		})
		// A flat extension has no allOf reference to the parent.
		Type("FlatChild", func() {
			Meta("openapi:typename:canonical", "true")
			Extend(parent)
			Attribute("size", Int)
		})
		other := Type("OtherParent", func() {
			Meta("openapi:typename:canonical", "true")
			Attribute("kind", String)
		})
		Type("OtherChild", func() {
			Meta("openapi:typename:canonical", "true")
			Extend(other)
			Attribute("name", String)
			Meta("openapi:allOf:reference", "true")
		})
		Type("Unrelated", func() {
			Meta("openapi:typename:canonical", "true")
			Attribute("kind", String)
		})
		Service("DiscriminatorRoundtrip", func() {
			Method("ImplicitParentGet", func() {
				Result(parent)
				HTTP(func() {
					GET("/implicit-parent")
					Response(StatusOK, func() {
						ContentType("application/json")
						Body(parent)
						OpenAPIBody(parent)
					})
				})
			})
		})
	}
}
