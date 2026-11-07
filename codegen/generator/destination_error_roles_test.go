package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/codegen/testdata"
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/internal/testingx"
)

func TestDestinationErrorRolesGenerated(t *testing.T) {
	for _, kind := range []string{"collision", "shared", "ordinary"} {
		for _, reverse := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/reverse=%t", kind, reverse), func(t *testing.T) {
				root := codegen.RunDSL(t, testdata.DestinationErrorRolesDSL(kind))
				traversal := *root
				traversal.Services = slices.Clone(root.Services)
				if reverse {
					slices.Reverse(traversal.Services)
				}
				const module = "example.com/errorroles"
				roots := []eval.Root{&traversal}
				files, err := Service(module+"/gen", roots)
				require.NoError(t, err)
				transportFiles, err := Transport(module+"/gen", roots)
				require.NoError(t, err)
				directory := t.TempDir()
				for _, file := range mergeFilesByPath(append(files, transportFiles...)) {
					_, err := file.Render(directory)
					require.NoError(t, err, file.Path)
				}
				data := service.NewServicesData(root).Get("alpha")
				errorType := strings.TrimPrefix(data.Methods[0].Errors[0].TypeRef, "*")
				harness := fmt.Sprintf(`package errorroles_test
import (
 "testing"
 common "example.com/errorroles/gen/common"
)
func TestAuthoredErrorMethods(t *testing.T) {
 message := "authored message"
 value := &%s{Message: &message}
 if got := value.Error(); got != message {
  t.Errorf("Error() = %%q, want %%q", got, message)
 }
 if got := value.LoomErrorName(); got != "authored_failure" {
  t.Errorf("LoomErrorName() = %%q", got)
 }
}
`, errorType)
				require.NoError(t, os.WriteFile(filepath.Join(directory, "harness_test.go"), []byte(harness), 0o600))
				mod := fmt.Sprintf("module %s\n\ngo 1.27\n\nrequire github.com/CaliLuke/loom v0.0.0\nreplace github.com/CaliLuke/loom => %s\n", module, testingx.RepoRoot())
				require.NoError(t, os.WriteFile(filepath.Join(directory, "go.mod"), []byte(mod), 0o600))
				for _, args := range [][]string{{"mod", "tidy"}, {"test", "./..."}} {
					output, err := testingx.RunCmd(directory, "go", args...)
					require.NoError(t, err, output)
				}
			})
		}
	}
}
