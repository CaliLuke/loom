package openapiv3_test

import (
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	httpgen "github.com/CaliLuke/loom/http/codegen"
	"github.com/stretchr/testify/require"
)

func TestHTTPBodyProjectionChildOnlyExclusion(t *testing.T) {
	for _, order := range []string{"removed_only", "full_then_removed", "removed_then_full"} {
		t.Run(order, func(t *testing.T) {
			root := httpgen.RunHTTPDSL(t, func() {
				API("child-only exclusion", func() {
					Meta("openapi:example", "false")
				})
				parent := Type("Parent", func() {
					Meta("openapi:discriminator:propertyName", "kind")
					Attribute("kind", String, func() {
						Meta("struct:tag:form", "k")
					})
					Required("kind")
				})
				child := Type("Child", func() {
					Extend(parent)
					Meta("openapi:allOf:reference", "true")
					Attribute("thumb", String)
				})
				methods := []string{"removed"}
				if order == "full_then_removed" {
					methods = []string{"full", "removed"}
				} else if order == "removed_then_full" {
					methods = []string{"removed", "full"}
				}
				Service("forms", func() {
					for _, method := range methods {
						Method(method, func() {
							Payload(child)
							HTTP(func() {
								POST("/" + method)
								FormRequest()
								if method == "removed" {
									Header("thumb")
								}
							})
						})
					}
				})
			})
			spec := bodyProjectionSpec(t, root)
			schemas := requireMap(t, requireMap(t, spec["components"], "components")["schemas"], "schemas")
			parentReference := func(path string) string {
				operation := requireOperation(t, spec, path, "post")
				content := requireMap(t, requireMap(t, operation["requestBody"], "requestBody")["content"], "content")
				body := resolveComponent(t, schemas, requireMap(t, content["application/x-www-form-urlencoded"], "form")["schema"])
				composition := requireSlice(t, body["allOf"], "body composition")
				if path == "/removed" {
					own := requireMap(t, composition[1], "selected body own schema")
					if properties, ok := own["properties"].(map[string]any); ok {
						require.NotContains(t, properties, "thumb", "transport exclusion is physically applied to the body")
					}
				}
				ref := requireString(t, requireMap(t, composition[0], "Parent reference")["$ref"], "Parent ref")
				base := resolveComponent(t, schemas, composition[0])
				require.Equal(t, "k", requireMap(t, base["discriminator"], "Parent hint")["propertyName"])
				return ref
			}
			removedParent := parentReference("/removed")
			if order != "removed_only" {
				fullParent := parentReference("/full")
				// The occurrence selection also governs the family's publication closure.
				// Equal local Parent fields do not make these complete contexts equal.
				require.NotEqual(t, fullParent, removedParent, "full and child-excluded publication contexts must stay distinct")
				fullPublished := 0
				for name, value := range schemas {
					schema := requireMap(t, value, name)
					parts, ok := schema["allOf"].([]any)
					if !ok || len(parts) != 2 || requireMap(t, parts[0], "full published base")["$ref"] != fullParent {
						continue
					}
					fullPublished++
					extension := requireMap(t, parts[1], "full published child extension")
					require.Contains(t, requireMap(t, extension["properties"], "full child properties"), "thumb", "full context must preserve its child's own field")
				}
				require.Positive(t, fullPublished)
			}
			published := 0
			for name, value := range schemas {
				schema := requireMap(t, value, name)
				parts, ok := schema["allOf"].([]any)
				if !ok || len(parts) != 2 || requireMap(t, parts[0], "published base")["$ref"] != removedParent {
					continue
				}
				published++
				extension := requireMap(t, parts[1], "published child extension")
				properties, _ := extension["properties"].(map[string]any)
				require.NotContains(t, properties, "thumb", "published child %s reintroduced a body-excluded member", name)
			}
			require.Positive(t, published, "hinted Parent's selected Child publication must remain reachable")
		})
	}
}
