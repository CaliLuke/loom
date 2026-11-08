package main

import (
	"encoding/json/v2"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/CaliLuke/loom/internal/testingx"
	"github.com/pb33f/libopenapi"
	"github.com/stretchr/testify/require"
)

func TestOpenAPIImportDiscriminatorRoundTrip(t *testing.T) {
	if os.Getenv("LOOM_OPENAPI_CONTRACT") == "" {
		t.Skip("set LOOM_OPENAPI_CONTRACT=1 to run the OpenAPI import contract")
	}
	root := testingx.RepoRoot()
	bin := filepath.Join(t.TempDir(), "loom")
	output, err := testingx.RunCmd(root, "go", "build", "-o", bin, "./cmd/loom")
	require.NoError(t, err, output)
	fixture, err := os.ReadFile(filepath.Join(root, "internal/openapiimport/testdata/discriminator/roundtrip.yaml"))
	require.NoError(t, err)
	for _, tc := range []struct {
		name    string
		version string
	}{
		{"3.2", "3.2.0"},
		{"3.1", "3.1.1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := fixture
			if tc.version == "3.1.1" {
				// Match TestDiscriminatorImportedOptional31: defaultMapping is
				// a 3.2 feature; the same optional-tag contract is valid in 3.1.
				text := strings.Replace(string(fixture), "openapi: 3.2.0", "openapi: 3.1.1", 1)
				lines := slices.DeleteFunc(strings.Split(text, "\n"), func(line string) bool {
					return strings.Contains(line, "defaultMapping:")
				})
				source = []byte(strings.Join(lines, "\n"))
			}
			directory := t.TempDir()
			module := "example.com/discriminator-roundtrip"
			writeRoundTripModule(t, directory, module, root)
			input := filepath.Join(directory, "input.yaml")
			require.NoError(t, os.WriteFile(input, source, 0o600))
			for _, destination := range []string{"design", "independent/design"} {
				output, err = testingx.RunCmd(directory, bin, "import", "openapi", input, "-o", destination)
				require.NoError(t, err, output)
			}
			first, err := os.ReadFile(filepath.Join(directory, "design/design.go"))
			require.NoError(t, err)
			second, err := os.ReadFile(filepath.Join(directory, "independent/design/design.go"))
			require.NoError(t, err)
			require.Equal(t, first, second, "independent import process bytes")
			for _, command := range [][]string{{"go", "mod", "tidy"}, {bin, "gen", module + "/design", "-o", "."}, {"go", "test", "-mod=mod", "./..."}} {
				if command[0] == "go" && command[1] == "test" {
					require.NoError(t, os.WriteFile(filepath.Join(directory, "roundtrip_test.go"), []byte(discriminatorRoundtripRuntime), 0o600))
				}
				output, err = testingx.RunCmd(directory, command[0], command[1:]...)
				require.NoError(t, err, output)
			}
			generated := map[string][]byte{}
			for _, name := range []string{"openapi.json", "openapi.yaml"} {
				value, err := os.ReadFile(filepath.Join(directory, "gen/http", name))
				require.NoError(t, err)
				parsed, err := libopenapi.NewDocument(value)
				require.NoError(t, err)
				require.Equal(t, tc.version, parsed.GetVersion())
				_, err = parsed.BuildV3Model()
				require.NoError(t, err)
				generated[name] = value
			}
			output, err = testingx.RunCmd(directory, bin, "gen", module+"/design", "-o", ".")
			require.NoError(t, err, output)
			for name, before := range generated {
				after, err := os.ReadFile(filepath.Join(directory, "gen/http", name))
				require.NoError(t, err)
				require.Equal(t, before, after, "independent generation process bytes: %s", name)
			}
			var contract map[string]any
			require.NoError(t, json.Unmarshal(generated["openapi.json"], &contract))
			if tc.version == "3.1.1" {
				schemas := contract["components"].(map[string]any)["schemas"].(map[string]any)
				pet := schemas["Pet"].(map[string]any)
				require.NotContains(t, pet, "required", "hint projection must not narrow an optional tag")
				require.NotContains(t, pet["discriminator"].(map[string]any), "defaultMapping")
			}
			delete(contract, "jsonSchemaDialect")
			delete(contract, "servers")
			prepared, err := json.Marshal(contract, json.Deterministic(true))
			require.NoError(t, err)
			reinput := filepath.Join(directory, "regenerated.json")
			require.NoError(t, os.WriteFile(reinput, prepared, 0o600))
			output, err = testingx.RunCmd(directory, bin, "import", "openapi", reinput, "-o", "reimport")
			require.NoError(t, err, output)
			for _, command := range [][]string{{bin, "gen", module + "/reimport", "-o", "."}, {"go", "test", "-mod=mod", "./..."}} {
				output, err = testingx.RunCmd(directory, command[0], command[1:]...)
				require.NoError(t, err, output)
			}
		})
	}
}

const discriminatorRoundtripRuntime = `package roundtrip_test

import (
	"encoding/json/v2"
	svc "example.com/discriminator-roundtrip/gen/discriminator_roundtrip"
	client "example.com/discriminator-roundtrip/gen/http/discriminator_roundtrip/client"
	server "example.com/discriminator-roundtrip/gen/http/discriminator_roundtrip/server"
	loomhttp "github.com/CaliLuke/loom/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

func TestDiscriminatorValidationNeutrality(t *testing.T) {
	for _, tc := range []struct {
		name, wire string
		value      func() any
		valid      bool
		errorText  string
	}{
		{"tagged", "{\"kind\":\"created\",\"data\":{\"name\":\"N\"}}", func() any {
			return new(svc.Event)
		}, true, ""},
		{"recursive tagged payload", "{\"kind\":\"created\",\"data\":{\"name\":\"N\",\"child\":{\"kind\":\"deleted\",\"data\":2}}}", func() any {
			return new(svc.Event)
		}, true, ""},
		{"recursive invalid tag", "{\"kind\":\"created\",\"data\":{\"name\":\"N\",\"child\":{\"kind\":\"made\",\"data\":{\"name\":\"C\"}}}}", func() any {
			return new(svc.Event)
		}, false, ""},
		{"alias is only a hint", "{\"kind\":\"made\",\"data\":{\"name\":\"N\"}}", func() any {
			return new(svc.Event)
		}, false, ""},
		{"missing tag", "{\"data\":{\"name\":\"N\"}}", func() any {
			return new(svc.Event)
		}, false, ""},
		{"closed envelope", "{\"kind\":\"created\",\"data\":{\"name\":\"N\"},\"extra\":1}", func() any {
			return new(svc.Event)
		}, false, ""},
		{"full object", "{\"kind\":\"cat\",\"name\":\"M\"}", func() any {
			return new(svc.Pet)
		}, true, ""},
		{"optional tag", "{\"age\":1}", func() any {
			return new(svc.Pet)
		}, true, ""},
		{"unmapped enum", "{\"kind\":\"other\",\"age\":1}", func() any {
			return new(svc.Pet)
		}, false, ""},
		{"closed branch", "{\"kind\":\"cat\",\"name\":\"M\",\"extra\":1}", func() any {
			return new(svc.Pet)
		}, false, ""},
		{"duplicate alternatives", "{\"kind\":\"cat\",\"name\":\"M\"}", func() any {
			return new(svc.DuplicatePet)
		}, false, "matched 2"},
		{"equal inline alternatives", "{\"kind\":\"first\",\"code\":1}", func() any {
			return new(svc.InlinePet)
		}, false, "matched 2"},
		{"unnamed implicit alternative", "{\"kind\":\"any\",\"name\":\"N\"}", func() any {
			return new(svc.ImplicitPet2)
		}, true, ""},
		{"unique alternative", "{\"kind\":\"dog\",\"age\":1}", func() any {
			return new(svc.DuplicatePet)
		}, true, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value := tc.value()
			err := json.Unmarshal([]byte(tc.wire), value)
			if (err == nil) != tc.valid {
				t.Errorf("decode %s: %v; valid=%t", tc.wire, err, tc.valid)
			}
			if tc.errorText != "" && (err == nil || !strings.Contains(err.Error(), tc.errorText)) {
				t.Errorf("error %v does not contain %q", err, tc.errorText)
			}
			if err == nil {
				if _, err := json.Marshal(value); err != nil {
					t.Errorf("encode admitted value: %v", err)
				}
			}
		})
	}
}

func TestImportedJSONQueryCodecs(t *testing.T) {
	values := url.Values{
		"event": {"{\"kind\":\"created\",\"data\":\"N\"}"},
		"pet":   {"{\"age\":1}"},
		"plain": {"{\"age\":1}"},
	}
	request := httptest.NewRequest("GET", "/query-json?"+values.Encode(), nil)
	payload, err := server.DecodeQueryJSONgetRequest(loomhttp.NewMuxer(), nil)(request)
	if err != nil {
		t.Fatalf("decode captured JSON query: %v", err)
	}
	encoded := httptest.NewRequest("GET", "/query-json", nil)
	if err := client.EncodeQueryJSONgetRequest(nil)(encoded, payload); err != nil {
		t.Fatalf("encode captured JSON query: %v", err)
	}
	for name, original := range values {
		var want, got any
		if err := json.Unmarshal([]byte(original[0]), &want); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal([]byte(encoded.URL.Query().Get(name)), &got); err != nil {
			t.Fatalf("query %s is not JSON: %v", name, err)
		}
		if !reflect.DeepEqual(want, got) {
			t.Errorf("query %s changed: %#v != %#v", name, want, got)
		}
	}
	for _, invalid := range []string{
		"{\"kind\":\"unknown\",\"data\":\"N\"}",
		"{\"kind\":\"created\",\"data\":2}",
		"{\"kind\":\"created\",\"data\":\"N\",\"extra\":true}",
	} {
		values.Set("event", invalid)
		request := httptest.NewRequest("GET", "/query-json?"+values.Encode(), nil)
		if _, err := server.DecodeQueryJSONgetRequest(loomhttp.NewMuxer(), nil)(request); err == nil {
			t.Errorf("accepted invalid query envelope %s", invalid)
		}
	}

}
`
