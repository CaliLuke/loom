package main

import (
	"debug/buildinfo"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGeneratorUsesCallerWorkspace(t *testing.T) {
	repositoryRoot, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	for _, workspace := range []string{"replace", "use"} {
		for _, command := range []string{"gen", "example", "vet"} {
			t.Run(workspace+"/"+command, func(t *testing.T) {
				root := t.TempDir()
				moduleDir := filepath.Join(root, "app")
				require.NoError(t, os.MkdirAll(filepath.Join(moduleDir, "design"), 0o755))
				mod := []byte("module example.com/service\n\ngo 1.27.2\n\nrequire github.com/CaliLuke/loom v0.0.0\n")
				if workspace == "use" {
					mod = []byte("module example.com/service\n\ngo 1.27.2\n")
				}
				require.NoError(t, os.WriteFile(filepath.Join(moduleDir, "go.mod"), mod, 0o600))
				sum, err := os.ReadFile(filepath.Join(repositoryRoot, "go.sum"))
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(filepath.Join(moduleDir, "go.sum"), sum, 0o600))
				require.NoError(t, os.WriteFile(filepath.Join(moduleDir, "design", "design.go"), []byte(`package design
import . "github.com/CaliLuke/loom/dsl"
var _ = API("service", func() {})
`), 0o600))
				work := "go 1.27.2\n\nuse ./app\n"
				if workspace == "replace" {
					work += fmt.Sprintf("replace github.com/CaliLuke/loom => %q\n", repositoryRoot)
				} else {
					work += fmt.Sprintf("use %q\n", repositoryRoot)
				}
				workFile := filepath.Join(root, "go.work")
				require.NoError(t, os.WriteFile(workFile, []byte(work), 0o600))
				t.Setenv("GOPROXY", "off")
				if workspace == "replace" {
					t.Setenv("GOWORK", workFile)
					t.Chdir(moduleDir)
				} else {
					t.Setenv("GOWORK", "")
					t.Chdir(root)
				}

				generator := NewGenerator(command, "example.com/service/design", moduleDir, false)
				require.NoError(t, generator.Write(false))
				t.Cleanup(func() { require.NoError(t, generator.Remove()) })
				require.NoError(t, generator.Compile(false))
				info, err := buildinfo.ReadFile(filepath.Join(generator.tmpDir, generator.bin))
				require.NoError(t, err)
				found := false
				for _, dependency := range info.Deps {
					if dependency.Path != "github.com/CaliLuke/loom" {
						continue
					}
					found = true
					if workspace == "replace" {
						require.NotNil(t, dependency.Replace)
						require.Equal(t, repositoryRoot, dependency.Replace.Path)
					} else {
						require.Equal(t, "(devel)", dependency.Version)
					}
				}
				require.True(t, found, "helper must link the workspace Loom module")
				if command == "vet" {
					afterMod, err := os.ReadFile(filepath.Join(moduleDir, "go.mod"))
					require.NoError(t, err)
					afterSum, err := os.ReadFile(filepath.Join(moduleDir, "go.sum"))
					require.NoError(t, err)
					require.Equal(t, mod, afterMod)
					require.Equal(t, sum, afterSum)
				}
			})
		}
	}
}
