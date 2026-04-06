package codegen

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	loomNamedImportPattern = regexp.MustCompile(`LoomNamedImport\("([^"]+)", "([^"]+)"\)`)
)

func TestGeneratorFilesDoNotMixNamedLoomImportsWithJenQual(t *testing.T) {
	t.Parallel()

	repoRoot, err := filepath.Abs("..")
	require.NoError(t, err)

	var offenders []string
	for _, dir := range []string{"codegen", "grpc", "http", "jsonrpc"} {
		root := filepath.Join(repoRoot, dir)
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
			require.NoError(t, walkErr)
			if d.IsDir() {
				return nil
			}
			if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
				return nil
			}

			srcBytes, readErr := os.ReadFile(path)
			require.NoError(t, readErr)
			src := string(srcBytes)

			matches := loomNamedImportPattern.FindAllStringSubmatch(src, -1)
			if len(matches) == 0 {
				return nil
			}

			for _, match := range matches {
				importPath := loomImportPath(match[1])
				qualNeedle := `jen.Qual("` + importPath + `",`
				if strings.Contains(src, qualNeedle) {
					relPath, relErr := filepath.Rel(repoRoot, path)
					require.NoError(t, relErr)
					offenders = append(offenders, relPath+": "+match[2]+" + "+qualNeedle)
				}
			}
			return nil
		})
		require.NoError(t, err)
	}

	require.Empty(t, offenders, "generator files must not mix LoomNamedImport aliases with jen.Qual for the same Loom package")
}

func loomImportPath(rel string) string {
	if rel == "" {
		return "github.com/CaliLuke/loom"
	}
	return "github.com/CaliLuke/loom/" + rel
}
