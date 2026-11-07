package openapiimport

import (
	"encoding/json/v2"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
)

func singletonClosureProfiles() []oneOfClosureProfile {
	return []oneOfClosureProfile{
		{"singleton-primitive", `[{"type":"string"}]`, []oneOfSemanticCase{{`"ok"`, true}, {`1`, false}, {`null`, false}}},
		{"singleton-constrained", `[{"type":"string","enum":["ok"]}]`, []oneOfSemanticCase{{`"ok"`, true}, {`"bad"`, false}, {`null`, false}}},
		{"singleton-object", `[{"type":"object","properties":{"x":{"type":"integer"}},"required":["x"]}]`, []oneOfSemanticCase{{`{"x":1}`, true}, {`{"x":1,"z":2}`, false}, {`{}`, false}}},
		{"singleton-any", `[{}]`, []oneOfSemanticCase{{`"ok"`, true}, {`{}`, true}, {`{"x":1}`, false}, {`null`, true}, {`[1]`, true}}},
		{"singleton-array", `[{"type":"array","items":{"type":"string"},"minItems":1}]`, []oneOfSemanticCase{{`["ok"]`, true}, {`[]`, false}, {`[1]`, false}, {`null`, false}}},
		{"singleton-map", `[{"type":"object","additionalProperties":{"type":"integer"}}]`, []oneOfSemanticCase{{`{"x":1}`, true}, {`{}`, true}, {`{"x":"bad"}`, false}}},
		{"singleton-nullable", `[{"type":"string"},{"type":"null"}]`, []oneOfSemanticCase{{`"ok"`, true}, {`null`, true}, {`1`, false}}},
		{"singleton-nested", `[{"oneOf":[{"$ref":"#/components/schemas/X"},{"type":"string"}]}]`, []oneOfSemanticCase{{`{"x":1}`, true}, {`{"x":1,"z":2}`, false}, {`"ok"`, true}, {`{}`, false}}},
	}
}

func TestOneOfSingletonClosureConstructor(t *testing.T) {
	source := oneOfClosureDocument(t, singletonClosureProfiles()...)
	imported, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	for _, diagnostic := range diagnostics {
		require.Equal(t, "schema-inline-object-promoted", diagnostic.Code)
	}
	design, err := Render(imported, Options{PackageName: "design"})
	require.NoError(t, err)
	requireRenderedDesignEvaluates(t, design, len(singletonClosureProfiles()))
}

func TestOneOfSingletonClosureGeneratedContract(t *testing.T) {
	profiles := append(singletonClosureProfiles(), oneOfClosureProfiles()...)
	source := oneOfClosureDocument(t, profiles...)
	imported, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	for _, diagnostic := range diagnostics {
		require.Equal(t, "schema-inline-object-promoted", diagnostic.Code)
	}
	design, err := Render(imported, Options{PackageName: "design"})
	require.NoError(t, err)
	directory := requireRenderedDesignGenerates(t, design)
	for _, filename := range []string{"openapi.json", "openapi.yaml"} {
		wire, err := os.ReadFile(filepath.Join(directory, "gen", "http", filename))
		require.NoError(t, err)
		parsed, err := libopenapi.NewDocument(wire)
		require.NoError(t, err)
		_, err = parsed.BuildV3Model()
		require.NoError(t, err)
	}
	wire, err := os.ReadFile(filepath.Join(directory, "gen", "http", "openapi.json"))
	require.NoError(t, err)
	// Match the existing strict round-trip harness: document identity and server
	// publication are excluded import capabilities, not part of schema semantics.
	var roundTrip map[string]any
	require.NoError(t, json.Unmarshal(wire, &roundTrip))
	delete(roundTrip, "$self")
	delete(roundTrip, "jsonSchemaDialect")
	delete(roundTrip, "servers")
	wire, err = json.Marshal(roundTrip, json.Deterministic(true))
	require.NoError(t, err)
	reimported, diagnostics, err := Analyze(wire)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	repeated, err := Render(reimported, Options{PackageName: "design"})
	require.NoError(t, err)
	requireRenderedDesignEvaluates(t, repeated, len(profiles))
	var original map[string]any
	require.NoError(t, json.Unmarshal(source, &original))
	exported := readGeneratedOpenAPIContract(t, directory)
	for _, document := range []map[string]any{original, exported} {
		for _, profile := range profiles {
			operation := document["paths"].(map[string]any)["/"+profile.name].(map[string]any)["post"].(map[string]any)
			schema := operation["responses"].(map[string]any)["200"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)
			require.Equal(t, 1, countOneOfClosureKeywords(t, document, schema, make(map[string]bool)), profile.name)
			for _, value := range profile.values {
				var instance any
				require.NoError(t, json.Unmarshal([]byte(value.wire), &instance))
				valid, _ := evaluateOneOfClosureSchema(t, document, schema, instance)
				require.Equal(t, value.valid, valid, "%s: %s", profile.name, value.wire)
			}
		}
	}
	requireSingletonGeneratedRuntime(t, directory, profiles)
	requireIndependentRegeneration(t, directory)
}

func requireSingletonGeneratedRuntime(t *testing.T, directory string, profiles []oneOfClosureProfile) {
	t.Helper()
	services, err := filepath.Glob(filepath.Join(directory, "gen", "*", "service.go"))
	require.NoError(t, err)
	require.Len(t, services, 1)
	servicePath, err := filepath.Rel(directory, filepath.Dir(services[0]))
	require.NoError(t, err)
	var cases strings.Builder
	for _, profile := range profiles {
		for _, value := range profile.values {
			fmt.Fprintf(&cases, "{method: %q, wire: %q, valid: %t},\n", codegen.Goify(profile.name, true), value.wire, value.valid)
		}
	}
	source := fmt.Sprintf(`package imported_test
import (
 "encoding/json/v2"
 "reflect"
 "testing"
 service %q
)
func TestImportedClosedUnionCodecs(t *testing.T) {
 for _, tc := range []struct {
  method, wire string
  valid bool
 }{
 %s
 } {
  t.Run(tc.method+tc.wire,func(t *testing.T) {
   method, ok := reflect.TypeFor[service.Service]().MethodByName(tc.method)
   if !ok {
    t.Fatalf("missing imported method %%s", tc.method)
   }
   typ := method.Type.Out(0)
   for typ.Kind() == reflect.Pointer {
    typ = typ.Elem()
   }
   destination := reflect.New(typ)
   err := json.Unmarshal([]byte(tc.wire),destination.Interface())
   if (err == nil) != tc.valid {
    t.Fatalf("decode %%s: %%v, valid=%%t",tc.wire,err,tc.valid)
   }
   if err != nil {
    return
   }
   encoded, err := json.Marshal(destination.Interface())
   if err != nil {
    t.Fatal(err)
   }
   var before, after any
   if err := json.Unmarshal([]byte(tc.wire),&before); err != nil {
    t.Fatal(err)
   }
   if err := json.Unmarshal(encoded,&after); err != nil {
    t.Fatal(err)
   }
   if !reflect.DeepEqual(before,after) {
    t.Fatalf("round trip %%s -> %%s",tc.wire,encoded)
   }
  })
 }
}
`, "example.com/imported/"+filepath.ToSlash(servicePath), cases.String())
	require.NoError(t, os.WriteFile(filepath.Join(directory, "singleton_test.go"), []byte(source), 0o600))
	command := exec.Command("go", "test", "-mod=mod", "./...")
	command.Dir = directory
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
}

func TestSingletonOneOfRenderingKeepsRepresentationMode(t *testing.T) {
	for _, tagged := range []bool{false, true} {
		t.Run(fmt.Sprint(tagged), func(t *testing.T) {
			schema := &Schema{OneOf: []*Schema{{Type: "string"}}}
			if tagged {
				schema.MediaTypes = []string{"application/json"}
			}
			renderer := &renderer{}
			expression, object, err := renderer.oneOfExpression(schema, "/schema")
			require.NoError(t, err)
			require.False(t, object)
			require.Contains(t, expression, "OneOf(func()")
			require.Equal(t, !tagged, strings.Contains(expression, "Untagged()"))
			schema.OneOf = append(schema.OneOf, &Schema{Type: "boolean"})
			if tagged {
				schema.MediaTypes = append(schema.MediaTypes, "application/example+json")
			}
			expression, _, err = renderer.oneOfExpression(schema, "/schema")
			require.NoError(t, err)
			require.Equal(t, "OneOf(String, Boolean)", expression)
		})
	}
}

func TestNullableClosedOneOfAdmission(t *testing.T) {
	for _, tc := range []struct {
		name, schema string
		branches     int
		accepted     bool
	}{
		{"singleton", `{"anyOf":[{"oneOf":[{"type":"string"}],"unevaluatedProperties":false},{"type":"null"}]}`, 1, true},
		{"ordinary union", `{"anyOf":[{"oneOf":[{"type":"string"},{"type":"integer"}],"unevaluatedProperties":false},{"type":"null"}]}`, 2, true},
		{"overlapping union", `{"anyOf":[{"oneOf":[{"type":"string"},{"type":"string"}],"unevaluatedProperties":false},{"type":"null"}]}`, 2, true},
		{"no null alternative", `{"anyOf":[{"oneOf":[{"type":"string"}],"unevaluatedProperties":false},{"type":"boolean"}]}`, 0, false},
		{"too many alternatives", `{"anyOf":[{"oneOf":[{"type":"string"}],"unevaluatedProperties":false},{"type":"null"},{"type":"boolean"}]}`, 0, false},
		{"unsupported non-null value", `{"anyOf":[{"allOf":[{"type":"string"},{"type":"integer"}]},{"type":"null"}]}`, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			document, diagnostics, err := Analyze(schemaAdmissionSource("3.1.1", tc.schema, "component"))
			require.NoError(t, err)
			if !tc.accepted {
				requireDiagnosticCode(t, diagnostics, "schema-composition")
				return
			}
			require.Empty(t, diagnostics)
			schema := document.Components.Schemas[0].Schema
			require.True(t, schema.Nullable)
			require.True(t, schema.UnionClosed)
			require.Len(t, schema.OneOf, tc.branches)
		})
	}
}
