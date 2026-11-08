package openapiv3_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	httpgen "github.com/CaliLuke/loom/http/codegen"
	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
	"github.com/stretchr/testify/require"
)

func TestHTTPBodyCompositionProjectionRenamed(t *testing.T) {
	for _, multipart := range []bool{false, true} {
		name := "form"
		if multipart {
			name = "multipart"
		}
		t.Run(name, func(t *testing.T) {
			root := httpgen.RunHTTPDSL(t, func() {
				API("body composition projection", func() {
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
					Attribute("kind", String, func() {
						Meta("struct:tag:form", "k")
					})
					Attribute("thumb", Bytes, func() {
						MinLength(3)
						Meta("struct:tag:form", "t")
					})
				})
				Service("forms", func() {
					Method("submit", func() {
						Payload(child)
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
			})
			spec := bodyProjectionSpec(t, root)
			schemas := requireMap(t, requireMap(t, spec["components"], "components")["schemas"], "schemas")
			media := "application/x-www-form-urlencoded"
			if multipart {
				media = "multipart/form-data"
			}
			body := resolveComponent(t, schemas, requireMap(t, formRequestContent(t, spec)[media], media)["schema"])
			allOf := requireSlice(t, body["allOf"], "selected Child retains authored composition")
			require.Len(t, allOf, 2)
			base := resolveComponent(t, schemas, allOf[0])
			require.Contains(t, requireMap(t, base["properties"], "projected Parent properties"), "k")
			require.NotContains(t, requireMap(t, base["properties"], "projected Parent properties"), "kind")
			require.Contains(t, requireSlice(t, bodyProjectionRequired(t, base), "projected required"), "k")
			require.Equal(t, "k", requireMap(t, base["discriminator"], "projected discriminator")["propertyName"])
			extension := requireMap(t, allOf[1], "Child extension")
			require.Contains(t, requireMap(t, extension["properties"], "Child properties"), "t")
		})
	}
}

// bodyProjectionSpec uses the maintained renderer and decoding helpers while
// retaining the exact document before any behavioral assertion. Version 3.1
// permits optional discriminator properties in the semantic-refinement cases.
func bodyProjectionSpec(t *testing.T, root *expr.RootExpr) map[string]any {
	t.Helper()
	root.API.Meta["openapi:version"] = []string{"3.1"}
	t.Log("DSL_ADMITTED")
	files, err := openapiv3.Files(root)
	require.NoError(t, err)
	for _, file := range files {
		if !strings.HasSuffix(file.Path, ".json") {
			continue
		}
		buffer := renderSection(t, file.AllSections()[0])
		raw := buffer.Bytes()
		if directory := os.Getenv("LOOM_BODY_PROJECTION_SPEC_OUT"); directory != "" {
			name := strings.ReplaceAll(t.Name(), "/", "__") + ".json"
			require.NoError(t, os.WriteFile(filepath.Join(directory, name), raw, 0o600))
		}
		return decodeOpenAPIJSON(t, raw)
	}
	t.Fatal("OpenAPI JSON document missing")
	return nil
}

func TestHTTPBodyCompositionProjectionSelection(t *testing.T) {
	for _, tc := range []struct {
		name     string
		header   string
		form     bool
		selected bool
		inline   bool
	}{
		{name: "json_canonical"},
		{name: "json_partial_base", header: "id"},
		{name: "form_partial_base", header: "id", form: true},
		{name: "form_removed_bound_hint", header: "kind", form: true},
		{name: "form_selected_document", form: true, selected: true},
		{name: "form_explicit_inline_subset", form: true, inline: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := httpgen.RunHTTPDSL(t, func() {
				API("selected composition", func() {
					Meta("openapi:example", "false")
				})
				parent := Type("Parent", func() {
					Meta("openapi:discriminator:propertyName", "kind")
					Attribute("id", String)
					Attribute("kind", String)
					Required("kind", "id")
				})
				child := Type("Child", func() {
					Extend(parent)
					Meta("openapi:allOf:reference", "true")
					Attribute("kind", String, func() {
						Meta("struct:tag:form", "k")
					})
					Attribute("thumb", Bytes, func() {
						Meta("struct:tag:form", "t")
						MinLength(3)
					})
				})
				Service("forms", func() {
					Method("submit", func() {
						if tc.selected {
							Payload(func() {
								Attribute("doc", child)
							})
						} else {
							Payload(child)
						}
						HTTP(func() {
							POST("/form")
							if tc.header != "" {
								Header(tc.header)
							}
							if tc.form {
								FormRequest()
							}
							if tc.selected {
								Body("doc")
							}
							if tc.inline {
								Header("kind")
								Header("id")
								Body(func() {
									Attribute("thumb")
								})
							}
						})
					})
					Method("canonical", func() {
						Result(child)
						HTTP(func() {
							GET("/canonical")
						})
					})
				})
			})
			spec := bodyProjectionSpec(t, root)
			schemas := requireMap(t, requireMap(t, spec["components"], "components")["schemas"], "schemas")
			canonical := requireMap(t, schemas["Parent"], "canonical Parent")
			require.Equal(t, "kind", requireMap(t, canonical["discriminator"], "canonical hint")["propertyName"])
			require.Contains(t, requireMap(t, canonical["properties"], "canonical properties"), "id")
			media := "application/json"
			if tc.form {
				media = "application/x-www-form-urlencoded"
			}
			body := resolveComponent(t, schemas, requireMap(t, formRequestContent(t, spec)[media], media)["schema"])
			if tc.inline {
				require.NotContains(t, body, "allOf", "explicit inline subset has no authored base")
				require.Contains(t, requireMap(t, body["properties"], "inline properties"), "t")
				return
			}
			allOf := requireSlice(t, body["allOf"], "selected composition")
			require.Len(t, allOf, 2)
			base := resolveComponent(t, schemas, allOf[0])
			properties := requireMap(t, base["properties"], "projected base properties")
			if tc.header == "id" {
				require.NotContains(t, properties, "id")
				require.NotContains(t, bodyProjectionRequired(t, base), "id")
			} else {
				require.Contains(t, properties, "id")
			}
			switch {
			case tc.header == "kind":
				require.NotContains(t, properties, "kind")
				require.NotContains(t, properties, "k")
				require.NotContains(t, base, "discriminator", "removed bound hint must not publish variants")
				require.NotContains(t, bodyProjectionRequired(t, base), "kind")
				require.NotContains(t, bodyProjectionRequired(t, base), "k")
			case tc.form:
				require.Contains(t, properties, "k")
				require.Equal(t, "k", requireMap(t, base["discriminator"], "projected hint")["propertyName"])
			default:
				require.Contains(t, properties, "kind")
				if tc.header == "" {
					require.Equal(t, "#/components/schemas/Parent", requireMap(t, allOf[0], "canonical base reference")["$ref"])
				}
			}
		})
	}
}

func TestHTTPBodyCompositionProjectionOwnContracts(t *testing.T) {
	root := httpgen.RunHTTPDSL(t, func() {
		API("contributor contracts", func() {
			Meta("openapi:example", "false")
		})
		parent := Type("Parent", func() {
			Meta("openapi:discriminator:propertyName", "kind")
			Attribute("kind", String)
			Attribute("n", Int, func() {
				Minimum(0)
				Default(2)
			})
		})
		child := Type("Child", func() {
			Extend(parent)
			Meta("openapi:allOf:reference", "true")
			Meta("openapi:encoding:thumb:contentType", "application/json")
			Attribute("kind", String, func() {
				Meta("struct:tag:form", "k")
			})
			Attribute("n", Int, func() {
				Maximum(100)
				Default(5)
			})
			Attribute("thumb", Bytes, func() {
				MinLength(3)
				Meta("struct:tag:form", "t")
			})
			Required("kind")
		})
		sibling := Type("Sibling", func() {
			Extend(parent)
			Meta("openapi:allOf:reference", "true")
			Attribute("n", Int, func() {
				Maximum(2)
				Default(1)
			})
			Attribute("sibling", String)
		})
		Service("forms", func() {
			Method("submit", func() {
				Payload(child)
				HTTP(func() {
					POST("/form")
					FormRequest()
				})
			})
			Method("parent", func() {
				Result(parent)
				HTTP(func() {
					GET("/parent")
				})
			})
			Method("canonical", func() {
				Result(sibling)
				HTTP(func() {
					GET("/canonical")
				})
			})
		})
	})
	spec := bodyProjectionSpec(t, root)
	schemas := requireMap(t, requireMap(t, spec["components"], "components")["schemas"], "schemas")
	canonical := requireMap(t, schemas["Parent"], "canonical Parent")
	require.NotContains(t, bodyProjectionRequired(t, canonical), "kind")
	body := resolveComponent(t, schemas, requireMap(t, formRequestContent(t, spec)["application/x-www-form-urlencoded"], "form")["schema"])
	allOf := requireSlice(t, body["allOf"], "Child composition")
	baseRef := requireMap(t, allOf[0], "projected base reference")["$ref"]
	base := resolveComponent(t, schemas, allOf[0])
	require.NotContains(t, bodyProjectionRequired(t, base), "k", "Child requiredness must not strengthen Parent")
	properties := requireMap(t, base["properties"], "Parent properties")
	require.Contains(t, properties, "k", "Child inherited wire refinement applies to Parent")
	require.NotContains(t, properties, "t", "Child-only encoding slot is not a Parent member")
	n := requireMap(t, properties["n"], "Parent.n")
	require.EqualValues(t, 2, n["default"])
	require.EqualValues(t, 0, n["minimum"])
	require.NotContains(t, n, "maximum", "Child constraints must not strengthen Parent")
	extension := requireMap(t, allOf[1], "Child extension")
	childN := requireMap(t, requireMap(t, extension["properties"], "Child properties")["n"], "Child.n")
	require.EqualValues(t, 5, childN["default"])
	require.EqualValues(t, 100, childN["maximum"])
	require.Contains(t, bodyProjectionRequired(t, extension), "k")
	found := false
	for _, value := range schemas {
		schema := requireMap(t, value, "schema")
		parts, ok := schema["allOf"].([]any)
		if !ok || len(parts) != 2 || requireMap(t, parts[0], "base")["$ref"] != baseRef {
			continue
		}
		own := requireMap(t, parts[1], "extension")
		props, ok := own["properties"].(map[string]any)
		if !ok || props["sibling"] == nil {
			continue
		}
		found = true
		siblingN := requireMap(t, props["n"], "Sibling.n")
		require.EqualValues(t, 1, siblingN["default"])
		require.EqualValues(t, 2, siblingN["maximum"])
		require.NotContains(t, bodyProjectionRequired(t, own), "k")
	}
	require.True(t, found, "projected Parent publishes Sibling with its own inherited refinement")
}

func TestHTTPBodyCompositionProjectionContexts(t *testing.T) {
	root := httpgen.RunHTTPDSL(t, func() {
		API("body wire contexts", func() {
			Meta("openapi:example", "false")
		})
		parent := Type("Parent", func() {
			Meta("openapi:discriminator:propertyName", "kind")
			Attribute("kind", String)
			Required("kind")
		})
		children := make(map[string]expr.UserType)
		for _, wire := range []string{"first", "second"} {
			children[wire] = Type(wire+"Child", func() {
				Extend(parent)
				Meta("openapi:allOf:reference", "true")
				Attribute("kind", String, func() {
					Meta("struct:tag:form", wire)
				})
				Attribute("thumb", Bytes)
			})
		}
		Service("forms", func() {
			for _, wire := range []string{"first", "second"} {
				Method(wire, func() {
					Payload(children[wire])
					HTTP(func() {
						POST("/" + wire)
						FormRequest()
					})
				})
			}
			Method("canonical", func() {
				Result(parent)
				HTTP(func() {
					GET("/canonical")
				})
			})
		})
	})
	spec := bodyProjectionSpec(t, root)
	schemas := requireMap(t, requireMap(t, spec["components"], "components")["schemas"], "schemas")
	canonical := requireMap(t, schemas["Parent"], "canonical Parent")
	require.Contains(t, requireMap(t, canonical["properties"], "canonical properties"), "kind")
	refs := make([]any, 0, 2)
	for _, wire := range []string{"first", "second"} {
		t.Run(wire, func(t *testing.T) {
			operation := requireOperation(t, spec, "/"+wire, "post")
			content := requireMap(t, requireMap(t, operation["requestBody"], "requestBody")["content"], "content")
			media := requireMap(t, content["application/x-www-form-urlencoded"], "form")
			body := resolveComponent(t, schemas, media["schema"])
			allOf := requireSlice(t, body["allOf"], "occurrence Child composition")
			refs = append(refs, requireMap(t, allOf[0], "occurrence parent")["$ref"])
			base := resolveComponent(t, schemas, allOf[0])
			properties := requireMap(t, base["properties"], "occurrence parent properties")
			require.Contains(t, properties, wire)
			require.NotContains(t, properties, "kind")
			require.Equal(t, wire, requireMap(t, base["discriminator"], "occurrence hint")["propertyName"])
		})
	}
	if len(refs) == 2 {
		require.NotEqual(t, refs[0], refs[1], "different resolved wire contexts cannot reuse a stale projection")
	}
}

func bodyProjectionRequired(t *testing.T, schema map[string]any) []any {
	t.Helper()
	if schema["required"] == nil {
		return nil
	}
	return requireSlice(t, schema["required"], "required")
}
