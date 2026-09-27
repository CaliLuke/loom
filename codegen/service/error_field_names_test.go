package service

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/internal/testingx"
)

const errorFieldNamesHarness = `package errorfields_test

import (
	"testing"

	svc "example.com/errorfields/gen/faults"
)

func TestErrorMethods(t *testing.T) {
	e := &svc.Failure{ErrorDetail: "bad input", ErrorIdentity: "failure", ErrorGuidance: "wire guidance"}
	if got := e.Error(); got != "bad input" {
		t.Errorf("Error() = %q", got)
	}
	if got := e.LoomErrorName(); got != "failure" {
		t.Errorf("LoomErrorName() = %q", got)
	}
	if got := e.LoomErrorRemedy(); got == nil || got.Code != "fix" {
		t.Errorf("LoomErrorRemedy() = %#v", got)
	}
}
`

func TestErrorFieldNamesGeneratedModule(t *testing.T) {
	const module = "example.com/errorfields"
	root := codegen.RunDSL(t, func() {
		failure := dsl.Type("Failure", func() {
			dsl.Attribute("error", dsl.String, func() {
				dsl.Meta("struct:field:name", "ErrorDetail")
			})
			dsl.ErrorName("loom_error_name", dsl.String, func() {
				dsl.Meta("struct:field:name", "ErrorIdentity")
			})
			dsl.Attribute("loom_error_remedy", dsl.String, func() {
				dsl.Meta("struct:field:name", "ErrorGuidance")
			})
			dsl.Required("error", "loom_error_name", "loom_error_remedy")
		})
		dsl.Service("faults", func() {
			dsl.Method("show", func() {
				dsl.Error("failure", failure, func() {
					dsl.Remedy(func() {
						dsl.RemedyCode("fix")
					})
				})
			})
		})
	})
	dir := t.TempDir()
	services := NewServicesData(root)
	for _, service := range root.Services {
		for _, file := range Files(module+"/gen", service, services, make(map[string][]string)) {
			_, err := file.Render(dir)
			require.NoError(t, err, file.Path)
		}
	}
	mod := fmt.Sprintf("module %s\n\ngo 1.27.0\nrequire github.com/CaliLuke/loom v1.0.0\nreplace github.com/CaliLuke/loom => %s\n", module, testingx.RepoRoot())
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "error_test.go"), []byte(errorFieldNamesHarness), 0o600))
	for _, args := range [][]string{{"mod", "tidy"}, {"vet", "./..."}, {"test", "-count=1", "./..."}} {
		output, err := testingx.RunCmd(dir, "go", args...)
		require.NoError(t, err, output)
	}
}
