package openapiv3_test

import (
	"strings"
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	httpgen "github.com/CaliLuke/loom/http/codegen"
	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
	"github.com/stretchr/testify/require"
)

// TestNullableParentExplicitMapping keeps the parent hint on the non-null
// assertion while its explicitly mapped child refers back through allOf.
func TestNullableParentExplicitMapping(t *testing.T) {
	for _, tc := range []struct {
		name, version string
		fallback      bool
		wantError     bool
	}{
		{name: "3.1", version: "3.1"},
		{name: "3.2-default", version: "3.2", fallback: true},
		{name: "3.2-missing-default", version: "3.2", wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, childFirst := range []bool{false, true} {
				t.Run(map[bool]string{false: "parent_first", true: "child_first"}[childFirst], func(t *testing.T) {
					var root *expr.RootExpr
					t.Run("admission", func(t *testing.T) {
						root = httpgen.RunHTTPDSL(t, nullableParentExplicitMappingDSL(tc.fallback, childFirst))
						parent := root.UserType("NullableParent")
						require.NotNil(t, parent)
						require.True(t, expr.IsNullable(parent.Attribute()))
						require.Empty(t, parent.Attribute().CompositionReferenceBases())
						require.Equal(t, []string{"child=NullableChild"}, parent.Attribute().Meta["openapi:discriminator:mapping"])
						child := root.UserType("NullableChild")
						require.NotNil(t, child)
						require.Len(t, child.Attribute().CompositionReferenceBases(), 1)
					})
					require.NotNil(t, root)
					t.Run("render", func(t *testing.T) {
						root.API.Meta["openapi:version"] = []string{tc.version}
						files, err := openapiv3.Files(root)
						if tc.wantError {
							require.ErrorContains(t, err, "requires an explicit defaultMapping")
							require.Empty(t, files)
							return
						}
						require.NoError(t, err)
						found := false
						for _, file := range files {
							if !strings.HasSuffix(file.Path, ".json") {
								continue
							}
							found = true
							buffer := renderSection(t, file.AllSections()[0])
							spec := decodeOpenAPIJSON(t, buffer.Bytes())
							schemas := requireMap(t, requireMap(t, spec["components"], "components")["schemas"], "schemas")
							parent := requireMap(t, schemas["NullableParent"], "parent")
							require.NotContains(t, parent, "discriminator")
							alternatives := requireSlice(t, parent["anyOf"], "parent alternatives")
							require.Len(t, alternatives, 2)
							body := requireMap(t, alternatives[0], "parent non-null body")
							require.Equal(t, map[string]any{"type": "null"}, alternatives[1])
							require.Equal(t, []any{"kind"}, body["required"])
							hint := requireMap(t, body["discriminator"], "parent hint")
							require.Equal(t, "kind", hint["propertyName"])
							mapping := requireMap(t, hint["mapping"], "parent mapping")
							require.Equal(t, map[string]any{"child": "#/components/schemas/NullableChild"}, mapping)
							if tc.fallback {
								require.Equal(t, mapping["child"], hint["defaultMapping"])
							} else {
								require.NotContains(t, hint, "defaultMapping")
							}
							child := requireMap(t, schemas["NullableChild"], "mapped child")
							if nullable, ok := child["anyOf"]; ok {
								arms := requireSlice(t, nullable, "child alternatives")
								require.Len(t, arms, 2)
								require.Equal(t, map[string]any{"type": "null"}, arms[1])
								child = requireMap(t, arms[0], "child non-null body")
							}
							members := requireSlice(t, child["allOf"], "child composition")
							require.Len(t, members, 2)
							require.Equal(t, map[string]any{"$ref": "#/components/schemas/NullableParent"}, members[0])
							own := requireMap(t, members[1], "child own contract")
							properties := requireMap(t, own["properties"], "child properties")
							require.Equal(t, "string", requireMap(t, properties["name"], "child name")["type"])
						}
						require.True(t, found)
					})
				})
			}
		})
	}
}

func nullableParentExplicitMappingDSL(fallback, childFirst bool) func() {
	return func() {
		API("nullable mapping", func() {
			Meta("openapi:example", "false")
		})
		parent := Type("NullableParent", func() {
			Meta("openapi:typename:canonical", "true")
			Attribute("kind", String)
			Required("kind")
			Nullable()
			Meta("openapi:discriminator:propertyName", "kind")
			Meta("openapi:discriminator:mapping", "child=NullableChild")
			if fallback {
				Meta("openapi:discriminator:defaultMapping", "NullableChild")
			}
		})
		child := Type("NullableChild", func() {
			Meta("openapi:typename:canonical", "true")
			Extend(parent)
			Meta("openapi:allOf:reference", "true")
			Attribute("name", String)
		})
		Service("nullableParents", func() {
			methods := []struct {
				name   string
				result expr.UserType
			}{{"parent", parent}, {"child", child}}
			if childFirst {
				methods[0], methods[1] = methods[1], methods[0]
			}
			for _, method := range methods {
				Method(method.name, func() {
					Result(method.result)
					HTTP(func() {
						GET("/nullable-" + method.name)
					})
				})
			}
		})
	}
}
