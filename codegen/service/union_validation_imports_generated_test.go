package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/internal/testingx"
)

func TestUnionValidationImportsGenerated(t *testing.T) {
	for _, tc := range []struct {
		name, shape    string
		custom, length bool
	}{
		{"custom object", "object", true, true},
		{"custom primitive", "primitive", true, true},
		{"custom nested", "nested", true, true},
		{"ordinary", "object", false, true},
		{"no length", "object", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := codegen.RunDSL(t, unionValidationImportsDSL(tc.shape, tc.custom, tc.length))
			services := NewServicesData(root)
			directory := t.TempDir()
			const module = "example.com/imports"
			service := root.Services[0]
			require.NoError(t, SetUserTypeImports(module+"/gen", services.Get(service.Name)))
			for _, file := range Files(module+"/gen", service, services, make(map[string][]string)) {
				_, err := file.Render(directory)
				require.NoError(t, err)
			}
			mod := fmt.Sprintf("module %s\n\ngo 1.27.0\n\nrequire github.com/CaliLuke/loom v1.10.0-alpha.9.0.20261012201303-8c7d8d711b9e\nreplace github.com/CaliLuke/loom => %s\n", module, testingx.RepoRoot())
			require.NoError(t, os.WriteFile(filepath.Join(directory, "go.mod"), []byte(mod), 0o600))
			pkg := "imports"
			if tc.custom {
				pkg = "types"
			}
			wire := `{"name":%q}`
			switch tc.shape {
			case "primitive":
				wire = "%q"
			case "nested":
				wire = `{"cat":{"name":%q}}`
			}
			input := func(value string) string {
				return fmt.Sprintf("%q", fmt.Sprintf(wire, value))
			}
			runtime := strings.NewReplacer("PACKAGE", pkg, "GOOD", input("éé"), "SHORT", input("é"), "LONG", input("123456789"), "LIMITED", fmt.Sprint(tc.length)).Replace(unionValidationImportsRuntime)
			require.NoError(t, os.WriteFile(filepath.Join(directory, "validation_test.go"), []byte(runtime), 0o600))
			for _, args := range [][]string{{"mod", "tidy"}, {"test", "-race", "./..."}} {
				output, err := testingx.RunCmd(directory, "go", args...)
				require.NoError(t, err, output)
			}
		})
	}
}

const unionValidationImportsRuntime = `package validation_test
import (
 "encoding/json/v2"
 "testing"
 model "example.com/imports/gen/PACKAGE"
)
func TestUnicodeLength(t *testing.T) {
 for _, tc := range []struct { wire string; valid bool }{
  {GOOD, true}, {SHORT, !LIMITED}, {LONG, !LIMITED},
 } {
  var choice model.Choice
  err := json.Unmarshal([]byte(tc.wire), &choice)
  if (err == nil) != tc.valid {
   t.Errorf("wire %s: valid=%t, error=%v", tc.wire, tc.valid, err)
   continue
  }
  if !tc.valid {
   continue
  }
  if choice.Kind() != model.ChoiceKindCat {
   t.Errorf("wrong selected kind: %s", choice.Kind())
  }
  output, err := json.Marshal(choice)
  if err != nil || string(output) != tc.wire {
   t.Errorf("roundtrip %s: %s, %v", tc.wire, output, err)
  }
 }
}
`
