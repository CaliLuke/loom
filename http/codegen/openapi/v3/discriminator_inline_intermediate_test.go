package openapiv3_test

import (
	"maps"
	"slices"
	"strings"
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	httpgen "github.com/CaliLuke/loom/http/codegen"
	"github.com/stretchr/testify/require"
)

// TestDiscriminatorInlineIntermediateCodecContext reaches a byte-free hinted
// parent through inline composed attributes below intermediates that denote no
// declaration: an inline holder, an inline composition whose own body holds the
// composed attribute, an inline holder as a tagged union branch, and the same
// inline holder in a recursive declaration and behind an array of a named
// holder. Untagged union branch fields and array or map elements cannot be
// inline objects in the DSL. Each JSON
// response publishes the parent's byte-bearing child in the JSON
// representation.
func TestDiscriminatorInlineIntermediateCodecContext(t *testing.T) {
	composed := func(parent expr.UserType) func() {
		return func() {
			Extend(parent)
			Meta("openapi:allOf:reference", "true")
			Attribute("note", String)
		}
	}
	for _, tc := range []struct {
		name   string
		result func(parent expr.UserType) any
	}{
		{name: "inline holder", result: func(parent expr.UserType) any {
			return Type("Wrapper", func() {
				Attribute("holder", func() {
					Attribute("pet", composed(parent))
				})
			})
		}},
		{name: "inline composition holding a composed member", result: func(parent expr.UserType) any {
			base := Type("Base", func() {
				Attribute("label", String)
			})
			return Type("Wrapper", func() {
				Attribute("outer", func() {
					Extend(base)
					Meta("openapi:allOf:reference", "true")
					Attribute("pet", composed(parent))
				})
			})
		}},
		{name: "tagged union branch holder", result: func(parent expr.UserType) any {
			return Type("Choice", OneOf(func() {
				Attribute("holder", func() {
					Attribute("pet", composed(parent))
				})
				Attribute("name", String)
			}))
		}},
		{name: "recursive declaration with inline holder", result: func(parent expr.UserType) any {
			return Type("Tree", func() {
				Attribute("children", ArrayOf("Tree"))
				Attribute("holder", func() {
					Attribute("pet", composed(parent))
				})
			})
		}},
		{name: "array of named holder with inline composition", result: func(parent expr.UserType) any {
			holder := Type("Holder", func() {
				Attribute("pet", composed(parent))
			})
			return Type("Wrapper", func() {
				Attribute("holders", ArrayOf(holder))
			})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spec := representationSpec(t, httpgen.RunHTTPDSL(t, inlineIntermediateDSL(tc.result)))
			schemas := requireMap(t, requireMap(t, spec["components"], "components")["schemas"], "schemas")
			for _, name := range slices.Sorted(maps.Keys(schemas)) {
				require.NotContains(t, name, "_", "no second representation in a JSON-only document")
			}
			// Child's own allOf references Parent once; the authored inline
			// composition is a second reference.
			require.GreaterOrEqual(t, composedParentReferences(spec), 2, "the authored inline composition references Parent")
			child := requireMap(t, schemas["Child"], "child")
			allOf := requireSlice(t, child["allOf"], "child allOf")
			require.Equal(t, "#/components/schemas/Parent", requireMap(t, allOf[0], "child base")["$ref"])
			requireDecodedBytesField(t, requireMap(t, allOf[1], "child member"), "thumb")
		})
	}
}

// TestDiscriminatorInlineIntermediateMixedCodec is the codec boundary control.
// The inline holder in a JSON response publishes the JSON child. A named parent
// field in a form request is a default object field, which the form contract
// encodes as JSON content, so it references the same JSON parent. A form body
// that is the parent itself is a form occurrence: it keeps the raw parent and
// its raw child, and JSON context does not leak into it.
func TestDiscriminatorInlineIntermediateMixedCodec(t *testing.T) {
	spec := representationSpec(t, httpgen.RunHTTPDSL(t, func() {
		inlineIntermediateDSL(func(parent expr.UserType) any {
			return Type("Wrapper", func() {
				Attribute("holder", func() {
					Attribute("pet", func() {
						Extend(parent)
						Meta("openapi:allOf:reference", "true")
						Attribute("note", String)
					})
				})
			})
		})()
		Service("forms", func() {
			Method("submit", func() {
				Payload(func() {
					Attribute("pet", expr.Root.UserType("Parent"))
				})
				HTTP(func() {
					POST("/form")
					FormRequest()
				})
			})
			Method("root", func() {
				Payload(expr.Root.UserType("Parent"))
				HTTP(func() {
					POST("/root")
					FormRequest()
				})
			})
		})
	}))
	schemas := requireMap(t, requireMap(t, spec["components"], "components")["schemas"], "schemas")
	children := map[string]string{}
	for name, value := range schemas {
		schema := requireMap(t, value, name)
		if allOf, ok := schema["allOf"].([]any); ok && name != "Wrapper" {
			if base, ok := allOf[0].(map[string]any); ok {
				children[name] = requireString(t, base["$ref"], "child base")
			}
		}
	}
	require.Contains(t, children, "Child")
	require.Equal(t, "#/components/schemas/Parent", children["Child"], "the JSON parent publishes the JSON child")
	requireDecodedBytesField(t, requireMap(t, requireSlice(t, requireMap(t, schemas["Child"], "child")["allOf"], "allOf")[1], "member"), "thumb")
	paths := requireMap(t, spec["paths"], "paths")
	formContent := func(path string) map[string]any {
		post := requireMap(t, requireMap(t, paths[path], path)["post"], "post")
		content := requireMap(t, requireMap(t, post["requestBody"], "request body")["content"], "content")
		return requireMap(t, content["application/x-www-form-urlencoded"], "form content")
	}
	field := formContent("/form")
	require.Equal(t, "application/json", requireMap(t, requireMap(t, field["encoding"], "encoding")["pet"], "pet encoding")["contentType"])
	body := resolveComponent(t, schemas, field["schema"])
	require.Equal(t, "#/components/schemas/Parent", requireMap(t, requireMap(t, body["properties"], "properties")["pet"], "pet")["$ref"],
		"the JSON content field references the JSON parent")
	rawParent := requireString(t, requireMap(t, formContent("/root")["schema"], "root schema")["$ref"], "root reference")
	require.NotEqual(t, "#/components/schemas/Parent", rawParent, "the form root keeps the raw parent")
	var raw int
	for child, base := range children {
		if child == "Child" || base == "#/components/schemas/Parent" {
			continue
		}
		raw++
		member := requireMap(t, requireSlice(t, requireMap(t, schemas[child], child)["allOf"], "allOf")[1], "member")
		thumb := requireMap(t, requireMap(t, member["properties"], "properties")["thumb"], "thumb")
		require.Equal(t, "binary", thumb["format"], "the form representation keeps raw bytes")
		require.Equal(t, rawParent, base, "the raw child extends the form root's parent")
	}
	require.Equal(t, 1, raw, "one raw child for the form parent representation")
}

func inlineIntermediateDSL(result func(parent expr.UserType) any) func() {
	return func() {
		API("inline intermediate", func() {
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
		body := result(parent)
		Service("holders", func() {
			Method("show", func() {
				Result(body)
				HTTP(func() {
					GET("/holder")
				})
			})
		})
	}
}

// composedParentReferences counts allOf lists whose first member references
// Parent anywhere in the rendered document.
func composedParentReferences(document any) int {
	count := 0
	var walk func(any)
	walk = func(value any) {
		switch actual := value.(type) {
		case map[string]any:
			if allOf, ok := actual["allOf"].([]any); ok && len(allOf) > 0 {
				if first, ok := allOf[0].(map[string]any); ok && first["$ref"] == "#/components/schemas/Parent" {
					count++
				}
			}
			for _, child := range actual {
				walk(child)
			}
		case []any:
			for _, child := range actual {
				walk(child)
			}
		}
	}
	walk(document)
	return count
}

// TestDiscriminatorJSONFieldComposition reaches the hinted parent through an
// inline composed attribute below a JSON-encoded form or multipart field: an
// inline holder, the field itself and a field typed by a named holder. Form
// and multipart bodies are analyzed from value-plan copies, which carry no
// composition bases, so the authored allOf reference, the requiredness the
// parent imposes and the children it publishes all depend on reading the
// bases from the original graph. The field is JSON content, so the parent it
// references is the JSON representation, whose child carries the decoded-byte
// gate.
func TestDiscriminatorJSONFieldComposition(t *testing.T) {
	composed := func(parent expr.UserType) func() {
		return func() {
			Extend(parent)
			Meta("openapi:allOf:reference", "true")
			Attribute("note", String)
		}
	}
	for _, shape := range []struct {
		name   string
		field  string
		holder bool
		fields func(parent expr.UserType) func()
	}{
		{name: "inline holder", field: "holder", holder: true, fields: func(parent expr.UserType) func() {
			return func() {
				Attribute("holder", func() {
					Attribute("pet", composed(parent))
				})
			}
		}},
		{name: "inline field", field: "pet", fields: func(parent expr.UserType) func() {
			return func() {
				Attribute("pet", composed(parent))
			}
		}},
		{name: "named holder", field: "holder", holder: true, fields: func(parent expr.UserType) func() {
			holder := Type("Holder", func() {
				Attribute("pet", composed(parent))
			})
			return func() {
				Attribute("holder", holder)
			}
		}},
	} {
		for _, multipart := range []bool{false, true} {
			name, media := shape.name+" form", "application/x-www-form-urlencoded"
			if multipart {
				name, media = shape.name+" multipart", "multipart/form-data"
			}
			t.Run(name, func(t *testing.T) {
				spec := representationSpec(t, httpgen.RunHTTPDSL(t, formRequestDSL(shape.fields, multipart)))
				schemas := requireMap(t, requireMap(t, spec["components"], "components")["schemas"], "schemas")
				content := requireMap(t, formRequestContent(t, spec)[media], media)
				if !multipart {
					encoding := requireMap(t, requireMap(t, content["encoding"], "encoding")[shape.field], "field encoding")
					require.Equal(t, "application/json", encoding["contentType"])
				}
				body := resolveComponent(t, schemas, content["schema"])
				pet := resolveComponent(t, schemas, requireMap(t, body["properties"], "body properties")[shape.field])
				if shape.holder {
					pet = resolveComponent(t, schemas, requireMap(t, pet["properties"], "holder properties")["pet"])
				}
				allOf := requireSlice(t, pet["allOf"], "authored composition")
				require.Len(t, allOf, 2)
				parentRef := requireString(t, requireMap(t, allOf[0], "base")["$ref"], "base reference")
				require.Contains(t, requireMap(t, requireMap(t, allOf[1], "own body")["properties"], "own properties"), "note")
				parent := resolveComponent(t, schemas, allOf[0])
				require.Contains(t, requireSlice(t, parent["required"], "parent required"), "kind", "the parent's requiredness applies through the reference")
				require.Equal(t, "kind", requireMap(t, parent["discriminator"], "discriminator")["propertyName"])
				var children []string
				for child, value := range schemas {
					childAllOf, ok := requireMap(t, value, child)["allOf"].([]any)
					if !ok || len(childAllOf) != 2 || requireMap(t, childAllOf[0], "child base")["$ref"] != parentRef {
						continue
					}
					if properties, ok := requireMap(t, childAllOf[1], "member")["properties"].(map[string]any); ok && properties["thumb"] != nil {
						children = append(children, child)
					}
				}
				require.Len(t, children, 1, "the parent representation publishes its child")
				require.Equal(t, "#/components/schemas/Parent", parentRef, "JSON content references the JSON parent")
				requireDecodedBytesField(t, requireMap(t, requireSlice(t, requireMap(t, schemas[children[0]], "child")["allOf"], "child allOf")[1], "member"), "thumb")
				for name := range schemas {
					require.False(t, strings.HasPrefix(name, "Parent_") || strings.HasPrefix(name, "Child_"), "no raw parent or child representation remains: %s", name)
				}
				require.GreaterOrEqual(t, composedParentReferences(spec), 2, "the authored composition and the child both reference Parent")
			})
		}
	}
}

// TestDiscriminatorFormRootKeepsFormCodec is the form codec control: a form
// or multipart body that is the composed child itself keeps its composition
// and its raw byte field, because the root's fields are form fields.
func TestDiscriminatorFormRootKeepsFormCodec(t *testing.T) {
	for _, multipart := range []bool{false, true} {
		spec := representationSpec(t, httpgen.RunHTTPDSL(t, formRequestDSL(func(expr.UserType) func() {
			return nil
		}, multipart)))
		schemas := requireMap(t, requireMap(t, spec["components"], "components")["schemas"], "schemas")
		allOf := requireSlice(t, requireMap(t, schemas["Child"], "child")["allOf"], "child allOf")
		require.Equal(t, "#/components/schemas/Parent", requireMap(t, allOf[0], "child base")["$ref"])
		require.Contains(t, requireSlice(t, requireMap(t, schemas["Parent"], "parent")["required"], "parent required"), "kind")
		thumb := requireMap(t, requireMap(t, requireMap(t, allOf[1], "member")["properties"], "properties")["thumb"], "thumb")
		require.Equal(t, "binary", thumb["format"])
		require.Equal(t, float64(3), thumb["minLength"])
	}
}

// formRequestDSL declares the hinted Parent and its byte-bearing Child and a
// form or multipart request whose payload declares fields, or is Child when
// fields declares none.
func formRequestDSL(fields func(parent expr.UserType) func(), multipart bool) func() {
	return func() {
		API("form composition", func() {
			Meta("openapi:example", "false")
		})
		parent := Type("Parent", func() {
			Meta("openapi:discriminator:propertyName", "kind")
			Attribute("kind", String)
			Required("kind")
		})
		child := Type("Child", func() {
			Extend(parent)
			Meta("openapi:allOf:reference", "true")
			Attribute("thumb", Bytes, func() {
				MinLength(3)
			})
		})
		var payload any = child
		if declare := fields(parent); declare != nil {
			payload = declare
		}
		Service("forms", func() {
			Method("submit", func() {
				Payload(payload)
				HTTP(func() {
					POST("/form")
					if multipart {
						MultipartRequest()
					} else {
						FormRequest()
					}
				})
			})
		})
	}
}

func formRequestContent(t *testing.T, spec map[string]any) map[string]any {
	t.Helper()
	post := requireMap(t, requireMap(t, requireMap(t, spec["paths"], "paths")["/form"], "form path")["post"], "post")
	return requireMap(t, requireMap(t, post["requestBody"], "request body")["content"], "content")
}

// resolveComponent returns the schema value denotes, following one local
// component reference.
func resolveComponent(t *testing.T, schemas map[string]any, value any) map[string]any {
	t.Helper()
	schema := requireMap(t, value, "schema")
	if ref, ok := schema["$ref"].(string); ok {
		name := strings.TrimPrefix(ref, "#/components/schemas/")
		return requireMap(t, schemas[name], name)
	}
	return schema
}

// TestFormJSONContentCodecs renders the codec each form or multipart contract
// selects for a byte-bearing value. The same named array is JSON content per
// item in one field and form text in another; multipart bytes stay binary
// parts even under a JSON label. The same named union is JSON content in one
// field and styled form text in another, so each references its own
// representation of the shared record. Each value of a form map root is JSON
// content.
func TestFormJSONContentCodecs(t *testing.T) {
	itemField := func(t *testing.T, schemas map[string]any, body map[string]any, name string) map[string]any {
		t.Helper()
		array := resolveComponent(t, schemas, requireMap(t, body["properties"], "properties")[name])
		return map[string]any{"properties": map[string]any{"item": array["items"]}}
	}
	for _, multipart := range []bool{false, true} {
		name, media := "form", "application/x-www-form-urlencoded"
		if multipart {
			name, media = "multipart", "multipart/form-data"
		}
		t.Run("named array "+name, func(t *testing.T) {
			spec := representationSpec(t, httpgen.RunHTTPDSL(t, formCodecDSL(func() func() {
				blobs := Type("Blobs", ArrayOf(Bytes, func() {
					MinLength(3)
				}))
				return func() {
					Payload(func() {
						Attribute("jsonBlobs", blobs)
						Attribute("textBlobs", blobs)
						Required("jsonBlobs", "textBlobs")
						Meta("openapi:encoding:jsonBlobs:contentType", "application/json")
					})
				}
			}, multipart)))
			schemas := requireMap(t, requireMap(t, spec["components"], "components")["schemas"], "schemas")
			body := resolveComponent(t, schemas, requireMap(t, formRequestContent(t, spec)[media], media)["schema"])
			if multipart {
				for _, field := range []string{"jsonBlobs", "textBlobs"} {
					item := requireMap(t, itemField(t, schemas, body, field)["properties"], "properties")["item"]
					require.Equal(t, "binary", requireMap(t, item, field)["format"], "%s items stay binary parts", field)
				}
				return
			}
			requireDecodedBytesField(t, itemField(t, schemas, body, "jsonBlobs"), "item")
			text := requireMap(t, requireMap(t, itemField(t, schemas, body, "textBlobs")["properties"], "properties")["item"], "text item")
			require.Equal(t, "binary", text["format"])
			require.Equal(t, float64(3), text["minLength"])
		})
		t.Run("named union "+name, func(t *testing.T) {
			spec := representationSpec(t, httpgen.RunHTTPDSL(t, formCodecDSL(func() func() {
				record := Type("Record", func() {
					Attribute("x", String)
					Attribute("blob", Bytes, func() {
						MinLength(3)
					})
					Required("x", "blob")
				})
				choice := Type("Choice", OneOf(record, Boolean), Untagged)
				return func() {
					Payload(func() {
						Attribute("styled", choice)
						Attribute("json", choice)
						Required("styled", "json")
						Meta("openapi:encoding:styled:style", "form")
						Meta("openapi:encoding:styled:explode", "false")
						Meta("openapi:encoding:json:contentType", "application/json")
					})
				}
			}, multipart)))
			schemas := requireMap(t, requireMap(t, spec["components"], "components")["schemas"], "schemas")
			body := resolveComponent(t, schemas, requireMap(t, formRequestContent(t, spec)[media], media)["schema"])
			recordRef := func(field string) string {
				for _, branch := range requireSlice(t, requireMap(t, requireMap(t, body["properties"], "properties")[field], field)["oneOf"], field) {
					if ref, ok := requireMap(t, branch, "branch")["$ref"].(string); ok {
						return ref
					}
				}
				require.Failf(t, "missing record branch", "%s", field)
				return ""
			}
			jsonRef, styledRef := recordRef("json"), recordRef("styled")
			require.NotEqual(t, jsonRef, styledRef, "each codec references its own record representation")
			requireDecodedBytesField(t, resolveComponent(t, schemas, map[string]any{"$ref": jsonRef}), "blob")
			styled := requireMap(t, requireMap(t, resolveComponent(t, schemas, map[string]any{"$ref": styledRef})["properties"], "properties")["blob"], "blob")
			require.Equal(t, "binary", styled["format"])
		})
	}
	t.Run("map root form", func(t *testing.T) {
		spec := representationSpec(t, httpgen.RunHTTPDSL(t, func() {
			formRequestDSL(func(expr.UserType) func() {
				return nil
			}, false)()
			Service("maps", func() {
				Method("store", func() {
					Payload(MapOf(String, expr.Root.UserType("Child")))
					HTTP(func() {
						POST("/map")
						FormRequest()
					})
				})
			})
		}))
		schemas := requireMap(t, requireMap(t, spec["components"], "components")["schemas"], "schemas")
		post := requireMap(t, requireMap(t, requireMap(t, spec["paths"], "paths")["/map"], "map path")["post"], "post")
		content := requireMap(t, requireMap(t, post["requestBody"], "request body")["content"], "content")
		root := requireMap(t, requireMap(t, content["application/x-www-form-urlencoded"], "form")["schema"], "schema")
		value := resolveComponent(t, schemas, requireMap(t, root["additionalProperties"], "additionalProperties"))
		allOf := requireSlice(t, value["allOf"], "value allOf")
		require.Equal(t, "#/components/schemas/Parent", requireMap(t, allOf[0], "value base")["$ref"])
		requireDecodedBytesField(t, requireMap(t, allOf[1], "value member"), "thumb")
	})
}

// formCodecDSL declares a form or multipart request whose types and payload
// design declares.
func formCodecDSL(design func() func(), multipart bool) func() {
	return func() {
		API("form codecs", func() {
			Meta("openapi:example", "false")
		})
		payload := design()
		Service("codecs", func() {
			Method("submit", func() {
				payload()
				HTTP(func() {
					POST("/form")
					if multipart {
						MultipartRequest()
					} else {
						FormRequest()
					}
				})
			})
		})
	}
}
