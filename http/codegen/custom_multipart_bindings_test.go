package codegen

import (
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	cg "github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestCustomMultipartBindingAccumulator(t *testing.T) {
	for _, location := range []string{"path_string", "path", "query", "header", "cookie", "none", "basic", "bearer", "custom_query"} {
		t.Run(location, func(t *testing.T) {
			root := RunHTTPDSL(t, testdata.CustomMultipartBindingsDSL(location))
			services := CreateHTTPServices(root)
			endpoint := services.Get("uploads").Endpoints[0]
			require.NotNil(t, endpoint.MultipartRequestDecoder)
			plan := endpoint.Payload.Request.DecodePlan
			require.Equal(t, location != "none", plan.ElementsMustValidate)
			require.Equal(t, location == "basic", plan.MustValidate)
			files := ServerFiles("example.com/multipart/gen", services)
			var code string
			for _, file := range files {
				for _, section := range file.Section("multipart-request-decoder") {
					code += cg.SectionCode(t, section)
				}
			}
			require.NotEmpty(t, code)
			if location == "none" {
				require.NotContains(t, code, "var err error")
			} else {
				require.Contains(t, code, "var err error")
			}
		})
	}
}

func TestCustomMultipartBindingsGenerated(t *testing.T) {
	for _, location := range []string{"path_string", "path", "query", "header", "cookie", "none", "basic", "bearer", "custom_query"} {
		t.Run(location, func(t *testing.T) {
			root := RunHTTPDSL(t, testdata.CustomMultipartBindingsDSL(location))
			dir := t.TempDir()
			renderHTTPModule(t, dir, "example.com/multipart", root)
			module, err := os.ReadFile(filepath.Join(dir, "go.mod"))
			require.NoError(t, err)
			module = []byte(strings.ReplaceAll(string(module), "require github.com/CaliLuke/loom v1.0.0", "require github.com/CaliLuke/loom v1.10.0-alpha.9.0.20261012202605-58916bc96d80"))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), module, 0o600))
			if location == "custom_query" {
				require.NoError(t, os.MkdirAll(filepath.Join(dir, "scalar"), 0o700))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "scalar", "token.go"), []byte(customMultipartTokenSource), 0o600))
			}
			harness := strings.ReplaceAll(customMultipartBindingsHarness, "LOCATION", location)
			valueCheck := ""
			if location != "none" {
				valueCheck = `if fmt.Sprint(got.Value) != "9" {
 t.Errorf("decoded value = %v", got.Value)
}`
			}
			harness = strings.ReplaceAll(harness, "VALUE_CHECK", valueCheck)
			formatted, err := format.Source([]byte(harness))
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "integration_test.go"), formatted, 0o600))
			runGoCommand(t, dir, "mod", "tidy")
			runGoCommand(t, dir, "test", "-race", "./...")
			runGoCommand(t, dir, "vet", "./...")
		})
	}
}
