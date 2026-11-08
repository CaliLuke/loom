package openapiv3_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"

	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestInheritedRequiredProjection(t *testing.T) {
	for _, version := range []struct{ target, want string }{{"3.1", openapiv3.OpenAPICompatibilityVersion}, {"3.2", openapiv3.OpenAPIVersion}} {
		t.Run(version.target, func(t *testing.T) {
			artifacts := renderOpenAPIArtifactsForVersion(t, testdata.InheritedRequiredProjectionDSL, version.target, version.want)
			spec := decodeOpenAPIJSON(t, artifacts.JSON)
			upload := requireComponentSchema(t, spec, "UploadPayload")
			require.ElementsMatch(t, []any{"file", "filename"}, upload["required"])
			require.ElementsMatch(t, []string{"file", "filename"}, typedPropertyNames(upload))
			requireTypedWire(t, spec, "UploadPayload", []typedWireCase{
				{`{"file":"x","filename":"a.txt"}`, true}, {`{"filename":"a.txt"}`, false},
				{`{"file":"x"}`, false}, {`{"id":"path-only","file":"x","filename":"a.txt"}`, false},
			})
			media := requireResponseMediaType(t, requireOperation(t, spec, "/rows", "get"), "application/json")
			rows := resolveRefMap(t, spec, media["schema"].(map[string]any)["$ref"].(string))
			properties := rows["properties"].(map[string]any)
			direct := properties["items"].(map[string]any)["items"].(map[string]any)
			groupRef := properties["groups"].(map[string]any)["items"].(map[string]any)["$ref"].(string)
			group := resolveRefMap(t, spec, groupRef)
			nested := group["properties"].(map[string]any)["items"].(map[string]any)["items"].(map[string]any)
			for _, occurrence := range []map[string]any{direct, nested} {
				row := resolveRefMap(t, spec, occurrence["$ref"].(string))
				require.ElementsMatch(t, []any{"extra", "id"}, row["required"])
			}
			require.Equal(t, direct["$ref"], nested["$ref"], "equivalent row contracts reuse their component")
			requireTypedWire(t, spec, "Row", []typedWireCase{
				{`{"extra":"x","id":"y"}`, true}, {`{"extra":"x"}`, false}, {`{"id":"y"}`, false},
			})
			lintEffectiveCompositionArtifacts(t, artifacts)
		})
	}
}

func TestInheritedRequiredUsageProjection(t *testing.T) {
	var vacuum contractVacuum
	if os.Getenv("LOOM_OPENAPI_CONTRACT") != "" {
		vacuum = installContractVacuum(t)
	}
	for _, usage := range []string{"readOnly", "writeOnly"} {
		for _, wire := range []string{"id", "wire_id"} {
			t.Run(usage+"/"+wire, func(t *testing.T) {
				artifacts := renderOpenAPIArtifacts(t, func() {
					base := dsl.Type("Base", func() {
						dsl.Attribute("id", dsl.String, func() {
							dsl.Meta("openapi:"+usage, "true")
							dsl.Meta("struct:tag:json", wire)
						})
						dsl.AdditionalAttributes(dsl.String)
						dsl.Required("id", "id:literal")
					})
					derived := dsl.Type("Derived", func() {
						dsl.Extend(base)
						dsl.Attribute("extra", dsl.String)
						dsl.Required("extra")
					})
					dsl.Service("typed", func() {
						dsl.Method("use", func() {
							dsl.NoAccessCheck("Public contract test")
							if usage == "readOnly" {
								dsl.Payload(derived)
							} else {
								dsl.Result(derived)
							}
							dsl.HTTP(func() {
								dsl.POST("/")
							})
						})
					})
				})
				spec := decodeOpenAPIJSON(t, artifacts.JSON)
				operation := requireOperation(t, spec, "/", "post")
				var media map[string]any
				if usage == "readOnly" {
					media = operation["requestBody"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)
				} else {
					media = requireResponseMediaType(t, operation, "application/json")
				}
				ref := media["schema"].(map[string]any)["$ref"].(string)
				schema := resolveRefMap(t, spec, ref)
				require.ElementsMatch(t, []any{"extra", "id:literal"}, schema["required"])
				require.Contains(t, schema["properties"].(map[string]any)[wire].(map[string]any), "not")
				requireTypedWire(t, spec, ref[len("#/components/schemas/"):], []typedWireCase{
					{`{"extra":"x","id:literal":"v"}`, true},
					{`{"extra":"x"}`, false},
					{`{"extra":"x","id:literal":"v","` + wire + `":"forbidden"}`, false},
				})
				if vacuum.binary != "" {
					for extension, data := range map[string][]byte{"json": artifacts.JSON, "yaml": artifacts.YAML} {
						path := filepath.Join(t.TempDir(), "typed."+extension)
						require.NoError(t, os.WriteFile(path, data, 0o600))
						require.NoError(t, vacuum.lint(path))
					}
				}
			})
		}
	}
}
