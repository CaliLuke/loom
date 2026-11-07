package main

import (
	"encoding/json/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/internal/httpencoding"
	"github.com/CaliLuke/loom/internal/openapiimport"
	"github.com/CaliLuke/loom/internal/testingx"
)

func TestOpenAPIImportOneOfRoundTrip(t *testing.T) {
	if os.Getenv("LOOM_OPENAPI_CONTRACT") == "" {
		t.Skip("set LOOM_OPENAPI_CONTRACT=1 to run the OpenAPI import contract")
	}
	repoRoot := testingx.RepoRoot()
	bin := filepath.Join(t.TempDir(), "loom")
	output, err := testingx.RunCmd(repoRoot, "go", "build", "-o", bin, "./cmd/loom")
	require.NoError(t, err, output)
	fixture, err := os.ReadFile(filepath.Join(repoRoot, "internal/openapiimport/testdata/oneof_symmetry.yaml"))
	require.NoError(t, err)
	for _, version := range []string{"3.1.1", "3.2.0"} {
		t.Run(version, func(t *testing.T) {
			source := []byte(strings.Replace(string(fixture), "openapi: 3.2.0", "openapi: "+version, 1))
			expected := analyzeOpenAPIContract(t, source)
			// The source object field has no authored encoding. Its effective
			// OpenAPI default is JSON; export makes that default concrete.
			require.Equal(t, "/form", expected.Operations[0].Path)
			encoding := expected.Operations[0].RequestBody.Schema.MediaEncoding
			require.NotContains(t, encoding, "record")
			encoding["record"] = httpencoding.Serialization{MediaType: "application/json"}
			directory := t.TempDir()
			module := "example.com/oneof-symmetry"
			writeRoundTripModule(t, directory, module, repoRoot)
			input := filepath.Join(directory, "input.yaml")
			require.NoError(t, os.WriteFile(input, source, 0o600))
			for _, destination := range []string{"design", "independent/design"} {
				output, err := testingx.RunCmd(directory, bin, "import", "openapi", input, "-o", destination)
				require.NoError(t, err, output)
			}
			first, err := os.ReadFile(filepath.Join(directory, "design/design.go"))
			require.NoError(t, err)
			second, err := os.ReadFile(filepath.Join(directory, "independent/design/design.go"))
			require.NoError(t, err)
			require.Equal(t, string(first), string(second), "independent CLI imports differ")
			for _, command := range [][]string{{"go", "mod", "tidy"}, {bin, "gen", module + "/design", "-o", "."}} {
				output, err := testingx.RunCmd(directory, command[0], command[1:]...)
				require.NoError(t, err, output)
			}
			writeOneOfRoundTripRuntime(t, directory)
			for _, command := range [][]string{{"go", "mod", "tidy"}, {"go", "test", "./..."}} {
				output, err := testingx.RunCmd(directory, command[0], command[1:]...)
				require.NoError(t, err, output)
			}
			prepared := verifyOneOfRoundTripExport(t, directory, expected)
			reinput := filepath.Join(directory, "regenerated.json")
			require.NoError(t, os.WriteFile(reinput, prepared, 0o600))
			output, err = testingx.RunCmd(directory, bin, "import", "openapi", reinput, "-o", "reimport")
			require.NoError(t, err, output)
			// Regenerate from the imported export and exercise the same runtime
			// contract, so schema symmetry also proves usable generated code.
			for _, command := range [][]string{{bin, "gen", module + "/reimport", "-o", "."}, {"go", "mod", "tidy"}, {"go", "test", "./..."}} {
				output, err := testingx.RunCmd(directory, command[0], command[1:]...)
				require.NoError(t, err, output)
			}
			verifyOneOfRoundTripExport(t, directory, expected)
		})
	}
}

func verifyOneOfRoundTripExport(t *testing.T, directory string, expected *openapiimport.Document) []byte {
	t.Helper()
	for _, name := range []string{"openapi.json", "openapi.yaml"} {
		generated, err := os.ReadFile(filepath.Join(directory, "gen/http", name))
		require.NoError(t, err)
		document, err := libopenapi.NewDocument(generated)
		require.NoError(t, err)
		_, err = document.BuildV3Model()
		require.NoError(t, err)
	}
	generated, err := os.ReadFile(filepath.Join(directory, "gen/http/openapi.json"))
	require.NoError(t, err)
	requireOneOfSourceEncoding(t, generated)
	prepared, err := prepareGeneratedOpenAPIContract(generated, expected.Title)
	require.NoError(t, err)
	require.NoError(t, compareOpenAPIContracts(expected, analyzeOpenAPIContract(t, prepared)))
	return prepared
}

func writeOneOfRoundTripRuntime(t *testing.T, directory string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(directory, "symmetry_test.go"), []byte(oneOfRoundTripRuntime), 0o600))
}

const oneOfRoundTripRuntime = `package symmetry_test

import (
 "encoding/json/v2"
 "reflect"
 "testing"

 svc "example.com/oneof-symmetry/gen/one_of_symmetry"
)

func TestComposedOneOfRuntime(t *testing.T) {
 for _, tc := range []struct {
  name string
  value func() any
  valid, invalid []string
 }{
  {"single null", func() any {
   return new(svc.NullOnce)
  }, []string{"null", "1", "\"text\""}, []string{"true"}},
  {"duplicate nullable", func() any {
   return new(svc.NullTwice)
  }, []string{"1", "\"text\""}, []string{"null"}},
  {"local null removed", func() any {
   return new(svc.LocalNullRemoved)
  }, []string{"1", "\"text\""}, []string{"null"}},
  {"unconstrained overlap", func() any {
   return new(svc.AnyChoice)
  }, []string{"null", "5"}, []string{"\"overlap\""}},
  {"map array", func() any {
   return new(svc.MapChoice)
  }, []string{"{}", "[]", "{\"x\":0}"}, []string{"{\"x\":\"wrong\"}"}},
  {"nested", func() any {
   return new(svc.Nested)
  }, []string{"true", "1", "\"text\""}, []string{"\"other\""}},
  {"closed", func() any {
   return new(svc.ClosedChoice)
  }, []string{"{\"kind\":\"a\",\"x\":0}"}, []string{"{\"kind\":\"a\",\"x\":0,\"extra\":1}", "{\"kind\":\"a\"}"}},
  {"inline", func() any {
   return new(svc.InlineChoice)
  }, []string{"{\"flag\":false}", "\"inline\""}, []string{"{\"flag\":false,\"extra\":1}"}},
 } {
  t.Run(tc.name, func(t *testing.T) {
   for _, input := range tc.valid {
    target := tc.value()
    if err := json.Unmarshal([]byte(input), target); err != nil {
     t.Errorf("decode %s: %v", input, err)
     continue
    }
    encoded, err := json.Marshal(target)
    if err != nil {
     t.Errorf("encode %s: %v", input, err)
     continue
    }
    var want, got any
    if err := json.Unmarshal([]byte(input), &want); err != nil {
     t.Fatal(err)
    }
    if err := json.Unmarshal(encoded, &got); err != nil {
     t.Fatal(err)
    }
    if !reflect.DeepEqual(want, got) {
     t.Errorf("roundtrip %s: %s", input, encoded)
    }
   }
   for _, input := range tc.invalid {
    if err := json.Unmarshal([]byte(input), tc.value()); err == nil {
     t.Errorf("accepted %s", input)
    }
   }
  })
 }
}
`

func requireOneOfSourceEncoding(t *testing.T, source []byte) {
	t.Helper()
	var document map[string]any
	require.NoError(t, json.Unmarshal(source, &document))
	paths := document["paths"].(map[string]any)
	operation := paths["/form"].(map[string]any)["post"].(map[string]any)
	media := operation["requestBody"].(map[string]any)["content"].(map[string]any)["application/x-www-form-urlencoded"].(map[string]any)
	reference := media["schema"].(map[string]any)["$ref"].(string)
	name := strings.TrimPrefix(reference, "#/components/schemas/")
	schema := document["components"].(map[string]any)["schemas"].(map[string]any)[name].(map[string]any)
	properties := schema["properties"].(map[string]any)
	require.Equal(t, "array", properties["items"].(map[string]any)["type"], "source array must retain per-item application")
	require.Equal(t, "object", properties["record"].(map[string]any)["type"])
	encoding := media["encoding"].(map[string]any)
	require.NotContains(t, encoding, "items", "heterogeneous source array items retain branch-specific defaults")
	require.Equal(t, map[string]any{"contentType": "application/json"}, encoding["record"])
	require.Equal(t, map[string]any{"style": "form", "explode": false}, encoding["textChoice"])
}
