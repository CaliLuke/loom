package openapiimport

import (
	"encoding/json/v2"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/CaliLuke/libopenapi"
	"github.com/stretchr/testify/require"
)

func TestImportedPatternGeneratedContract(t *testing.T) {
	for _, tc := range []struct {
		name, pattern  string
		accept, reject []string
	}{
		{"ordinary", `^[a-z]{1,8}$`, []string{"cat", "a"}, []string{"A", "toolongname"}},
		{"whitespace", `^[\s]+$`, []string{"\v", "\u00a0", "\ufeff"}, []string{"x", "\u0085"}},
		{"unicode", `^(?:\u0041|[\b])$`, []string{"A", "\b"}, []string{"a", "b"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var source map[string]any
			require.NoError(t, json.Unmarshal(readFixture(t, "supported.json"), &source))
			schemas := source["components"].(map[string]any)["schemas"].(map[string]any)
			for _, name := range []string{"Pet", "NewPet"} {
				properties := schemas[name].(map[string]any)["properties"].(map[string]any)
				properties["name"].(map[string]any)["pattern"] = tc.pattern
			}
			encoded, err := json.Marshal(source, json.Deterministic(true))
			require.NoError(t, err)
			document, diagnostics, err := Analyze(encoded)
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			design, err := Render(document, Options{PackageName: "design"})
			require.NoError(t, err)
			module := requireRenderedDesignGenerates(t, design)
			probe := fmt.Sprintf(patternGeneratedProbe, tc.accept, tc.reject)
			require.NoError(t, os.WriteFile(filepath.Join(module, "pattern_test.go"), []byte(probe), 0600))
			test := exec.Command("go", "test", "-race", "-mod=mod", "./...")
			test.Dir = module
			output, err := test.CombinedOutput()
			require.NoError(t, err, string(output))
			if evidence := os.Getenv("LOOM_PATTERN_EVIDENCE"); evidence != "" {
				output := filepath.Join(evidence, tc.name)
				require.NoError(t, os.MkdirAll(output, 0750))
				require.NoError(t, os.WriteFile(filepath.Join(output, "design.go"), design, 0600))
				require.NoError(t, os.CopyFS(filepath.Join(output, "gen"), os.DirFS(filepath.Join(module, "gen"))))
			}
			for _, extension := range []string{"json", "yaml"} {
				wire, err := os.ReadFile(filepath.Join(module, "gen", "http", "openapi."+extension))
				require.NoError(t, err)
				parsed, err := libopenapi.NewDocument(wire)
				require.NoError(t, err)
				_, err = parsed.BuildV3Model()
				require.NoError(t, err)
			}
			contract := readGeneratedOpenAPIContract(t, module)
			for _, name := range []string{"Pet", "NewPet"} {
				properties := contract["components"].(map[string]any)["schemas"].(map[string]any)[name].(map[string]any)["properties"].(map[string]any)
				projected := properties["name"].(map[string]any)["pattern"].(string)
				command := exec.Command("node", "-e", `const [a,b]=process.argv.slice(1).map(p=>new RegExp(p));for(const x of ['','cat','A','\b','\v','\u00a0','\ufeff','\u0085','😀']){if(a.test(x)!==b.test(x))throw new Error(JSON.stringify(x));}`, tc.pattern, projected)
				output, err := command.CombinedOutput()
				require.NoError(t, err, string(output))
			}
		})
	}
}

// patternGeneratedProbe exercises the actual request and response validators,
// including the compiled regex declarations emitted by the File finalizer.
const patternGeneratedProbe = `package imported_test

import (
    "testing"

    client "example.com/imported/gen/http/pet_api/client"
    server "example.com/imported/gen/http/pet_api/server"
)

func TestPatternValidators(t *testing.T) {
    accepted := %#v
    rejected := %#v
    for _, group := range []struct {
        values []string
        valid bool
    }{{accepted, true}, {rejected, false}} {
        for _, value := range group.values {
            id, status := int64(1), client.StatusResponseBody(1)
            request := &server.PetsCreateRequestBody{Name: &value}
            response := &client.PetsCreateResponseBody{ID: &id, Name: &value, Status: &status}
            for name, err := range map[string]error{
                "request": server.ValidatePetsCreateRequestBody(request),
                "response": client.ValidatePetsCreateResponseBody(response),
            } {
                if (err == nil) != group.valid {
                    t.Errorf("%%s value %%q valid=%%t: %%v", name, value, group.valid, err)
                }
            }
        }
    }
}
`
