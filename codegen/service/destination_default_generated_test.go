package service

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/internal/testingx"
)

func TestRelocatedObjectDefaultGenerated(t *testing.T) {
	root := codegen.RunDSL(t, relocatedObjectDefaultDSL)
	services := NewServicesData(root)
	data := services.Get("svc")
	child := root.Services[0].Methods[0].Payload.Find("child")
	ctx := codegen.NewAttributeContext(false, false, true, data.PkgName, data.Scope)
	value, err := ctx.DefaultValue(child, child.DefaultValue)
	require.NoError(t, err)
	const module = "example.com/destination"
	directory := t.TempDir()
	require.NoError(t, SetUserTypeImports(module+"/gen", data))
	for _, file := range Files(module+"/gen", root.Services[0], services, make(map[string][]string)) {
		_, err := file.Render(directory)
		require.NoError(t, err)
	}
	mod := fmt.Sprintf("module %s\n\ngo 1.27\n\nrequire github.com/CaliLuke/loom v0.0.0\nreplace github.com/CaliLuke/loom => %s\n", module, testingx.RepoRoot())
	require.NoError(t, os.WriteFile(filepath.Join(directory, "go.mod"), []byte(mod), 0o600))
	runtime := `package destination_test
import (
 "testing"
 common "example.com/destination/gen/common"
 loom "github.com/CaliLuke/loom/pkg"
)
func TestRelocatedDefaultNull(t *testing.T) {
 value := ` + value.Expression + `
 if !value.Pick.IsNull() {
  t.Error("authored nested null was not retained")
 }
}
`
	require.NoError(t, os.WriteFile(filepath.Join(directory, "default_test.go"), []byte(runtime), 0o600))
	for _, args := range [][]string{{"mod", "tidy"}, {"test", "./..."}} {
		output, err := testingx.RunCmd(directory, "go", args...)
		require.NoError(t, err, output)
	}
}
