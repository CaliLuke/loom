package openapiv3_test

import (
	"maps"
	"slices"
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	httpgen "github.com/CaliLuke/loom/http/codegen"
	"github.com/stretchr/testify/require"
)

// TestDiscriminatorOccurrenceCodecContext reaches a byte-free hinted parent
// through the occurrence shapes of a JSON response that HTTP lowering
// produces: fields, array elements and map values typed by the parent (renamed
// HTTP copies), an inline field composed with it, the same inline field inside
// a nested holder declaration, and an inline root composed with it. Each shape
// publishes the parent's byte-bearing child in the JSON
// representation. Direct HTTP body copies and flattened unions are covered by
// TestDiscriminatorPublicationCodecContext and
// TestDiscriminatorUnionHintFallbackCodecContext.
func TestDiscriminatorOccurrenceCodecContext(t *testing.T) {
	property := func(name string) func(*testing.T, map[string]any, map[string]any) map[string]any {
		return func(t *testing.T, _, body map[string]any) map[string]any {
			t.Helper()
			return requireMap(t, requireMap(t, body["properties"], "properties")[name], name)
		}
	}
	for _, tc := range []struct {
		name string
		// composed reports whether the occurrence references the parent as an
		// allOf contributor rather than through a value typed by it.
		composed bool
		result   func(parent expr.UserType) any
		// locate returns the schema that refers to the parent, given the spec
		// and the response body schema.
		locate func(*testing.T, map[string]any, map[string]any) map[string]any
	}{
		{name: "nested field", locate: property("pet"), result: func(parent expr.UserType) any {
			return Type("Holder", func() {
				Attribute("pet", parent)
			})
		}},
		{name: "array element", result: func(parent expr.UserType) any {
			return Type("Holder", func() {
				Attribute("pets", ArrayOf(parent))
			})
		}, locate: func(t *testing.T, spec, body map[string]any) map[string]any {
			t.Helper()
			return requireMap(t, property("pets")(t, spec, body)["items"], "items")
		}},
		{name: "map value", result: func(parent expr.UserType) any {
			return Type("Holder", func() {
				Attribute("byName", MapOf(String, parent))
			})
		}, locate: func(t *testing.T, spec, body map[string]any) map[string]any {
			t.Helper()
			return requireMap(t, property("byName")(t, spec, body)["additionalProperties"], "additionalProperties")
		}},
		{name: "inline composed field", composed: true, locate: property("pet"), result: func(parent expr.UserType) any {
			return Type("Holder", func() {
				Attribute("pet", func() {
					Extend(parent)
					Meta("openapi:allOf:reference", "true")
					Attribute("note", String)
				})
			})
		}},
		{name: "deep inline composed field", composed: true, result: func(parent expr.UserType) any {
			holder := Type("Holder", func() {
				Attribute("pet", func() {
					Extend(parent)
					Meta("openapi:allOf:reference", "true")
					Attribute("note", String)
				})
			})
			return Type("Wrapper", func() {
				Attribute("holder", holder)
			})
		}, locate: func(t *testing.T, spec, body map[string]any) map[string]any {
			t.Helper()
			holder := resolveCardinalitySchema(t, spec, property("holder")(t, spec, body))
			return property("pet")(t, spec, holder)
		}},
		{name: "inline composed root", composed: true, result: func(parent expr.UserType) any {
			return func() {
				Extend(parent)
				Meta("openapi:allOf:reference", "true")
				Attribute("note", String)
			}
		}, locate: func(_ *testing.T, _, body map[string]any) map[string]any {
			return body
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := httpgen.RunHTTPDSL(t, occurrenceContextDSL(tc.result))
			spec := representationSpec(t, root)
			schemas := requireMap(t, requireMap(t, spec["components"], "components")["schemas"], "schemas")
			for _, name := range slices.Sorted(maps.Keys(schemas)) {
				require.NotContains(t, name, "_", "no second representation in a JSON-only document")
			}
			media := requireResponseMediaType(t, requireOperation(t, spec, "/holder", "get"), "application/json")
			referring := tc.locate(t, spec, resolveCardinalitySchema(t, spec, requireMap(t, media["schema"], "holder")))
			if tc.composed {
				// The authored allOf reference marker keeps the contributor.
				allOf := requireSlice(t, referring["allOf"], "composed allOf")
				require.Equal(t, "#/components/schemas/Parent", requireMap(t, allOf[0], "contributor")["$ref"])
			} else {
				require.Equal(t, "#/components/schemas/Parent", referring["$ref"])
			}
			require.Contains(t, schemas, "Parent")
			child := requireMap(t, schemas["Child"], "child")
			allOf := requireSlice(t, child["allOf"], "child allOf")
			require.Equal(t, "#/components/schemas/Parent", requireMap(t, allOf[0], "child base")["$ref"])
			requireDecodedBytesField(t, requireMap(t, allOf[1], "child member"), "thumb")
		})
	}
}

func occurrenceContextDSL(result func(parent expr.UserType) any) func() {
	return func() {
		API("occurrence context", func() {
			Meta("openapi:example", "false")
		})
		parent := Type("Parent", func() {
			Meta("openapi:discriminator:propertyName", "kind")
			Attribute("kind", String)
			Required("kind")
		})
		Type("Child", func() {
			Extend(parent)
			Meta("openapi:allOf:reference", "true")
			Attribute("thumb", Bytes, func() {
				MinLength(3)
			})
		})
		holder := result(parent)
		Service("holders", func() {
			Method("show", func() {
				Result(holder)
				HTTP(func() {
					GET("/holder")
				})
			})
		})
	}
}
