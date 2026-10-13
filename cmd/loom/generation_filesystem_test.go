package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/mod/modfile"

	"github.com/CaliLuke/loom/codegen"
)

func TestGenerationStageUsesWritableOutput(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("requires ordinary POSIX directory permissions")
	}
	parent := t.TempDir()
	module := filepath.Join(parent, "module")
	require.NoError(t, os.Mkdir(module, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(module, "go.mod"), []byte("module example.com/app\n\ngo 1.27.2\n"), 0o600))
	require.NoError(t, os.Chmod(parent, 0o500))
	t.Cleanup(func() {
		require.NoError(t, os.Chmod(parent, 0o700))
	})
	tx, err := newGenerationTransaction(module)
	require.NoError(t, err)
	require.NoError(t, tx.cleanup())
}

func TestGenerationStageModuleContext(t *testing.T) {
	for _, nested := range []string{".", "nested/output"} {
		t.Run(nested, func(t *testing.T) {
			root := t.TempDir()
			module := filepath.Join(root, "module")
			output := filepath.Join(module, nested)
			require.NoError(t, os.MkdirAll(output, 0o750))
			absoluteTarget := filepath.Join(root, "absolute")
			original := []byte(fmt.Sprintf("module example.com/app\n\ngo 1.27.2\n\nreplace (\nexample.com/dep => ../dep\nexample.com/other v1.0.0 => ./other\nexample.com/absolute => %q\nexample.com/version v1.0.0 => example.com/release v1.2.3\n)\n", absoluteTarget))
			require.NoError(t, os.WriteFile(filepath.Join(module, "go.mod"), original, 0o600))
			tx, err := newGenerationTransaction(output)
			require.NoError(t, err)
			t.Cleanup(func() {
				require.NoError(t, tx.cleanup())
			})
			rel, err := filepath.Rel(output, tx.root)
			require.NoError(t, err)
			if !filepath.IsLocal(rel) {
				t.Errorf("render stage must reside under writable output: %s", tx.root)
			}
			staged, err := os.ReadFile(filepath.Join(tx.root, "go.mod"))
			require.NoError(t, err)
			parsed, err := modfile.Parse("go.mod", staged, nil)
			require.NoError(t, err)
			require.Equal(t, "example.com/app", parsed.Module.Mod.Path)
			require.Len(t, parsed.Replace, 4)
			target := parsed.Replace[0].New.Path
			if !filepath.IsAbs(target) {
				target = filepath.Join(tx.root, target)
			}
			require.Equal(t, filepath.Join(root, "dep"), filepath.Clean(target))
			require.Equal(t, filepath.Join(module, "other"), parsed.Replace[1].New.Path)
			require.Equal(t, "v1.0.0", parsed.Replace[1].Old.Version)
			require.Equal(t, absoluteTarget, parsed.Replace[2].New.Path)
			require.Equal(t, "example.com/release", parsed.Replace[3].New.Path)
			require.Equal(t, "v1.2.3", parsed.Replace[3].New.Version)
			rel, err = filepath.Rel(tx.root, tx.stage)
			require.NoError(t, err)
			require.Equal(t, filepath.Clean(nested), rel)
			after, err := os.ReadFile(filepath.Join(module, "go.mod"))
			require.NoError(t, err)
			require.Equal(t, original, after)
		})
	}
}

// TestGenerationMountedDestinations runs in a disposable container with a tmpfs
// mounted at LOOM_TEST_GENERATION_MOUNT. Its Go binary is built on the host.
func TestGenerationMountedDestinations(t *testing.T) {
	mount := os.Getenv("LOOM_TEST_GENERATION_MOUNT")
	if mount == "" {
		t.Skip("requires an isolated mount; see generation publication roadmap")
	}
	for _, kind := range []string{"module", "nested-output", "external-create", "external-replace"} {
		t.Run(kind, func(t *testing.T) {
			base := t.TempDir()
			output := base
			switch kind {
			case "module":
				output = mount
			case "nested-output":
				require.NoError(t, os.WriteFile(filepath.Join(base, "go.mod"), []byte("module example.com/app\n\ngo 1.27.2\n"), 0o600))
				output = filepath.Join(base, "nested")
				require.NoError(t, os.Symlink(mount, output))
			default:
				require.NoError(t, os.Symlink(mount, filepath.Join(output, "plugin")))
			}
			if kind == "module" {
				require.NoError(t, os.WriteFile(filepath.Join(output, "go.mod"), []byte("module example.com/app\n\ngo 1.27.2\n"), 0o600))
			}
			t.Cleanup(func() {
				require.NoError(t, os.RemoveAll(filepath.Join(mount, "go.mod")))
				require.NoError(t, os.RemoveAll(filepath.Join(mount, "gen")))
			})
			tx, err := newGenerationTransaction(output)
			require.NoError(t, err)
			t.Cleanup(func() {
				require.NoError(t, tx.cleanup())
			})
			require.NoError(t, os.MkdirAll(filepath.Join(output, "gen"), 0o750))
			require.NoError(t, os.WriteFile(filepath.Join(output, "gen", "old.txt"), []byte("old"), 0o600))
			require.NoError(t, os.MkdirAll(filepath.Join(tx.stage, "gen"), 0o750))
			staged := filepath.Join(tx.stage, "gen", "current.txt")
			require.NoError(t, os.WriteFile(staged, []byte("generated"), 0o600))
			artifacts := []codegen.Artifact{{Path: staged}}
			if kind == "external-create" || kind == "external-replace" {
				require.NoError(t, os.MkdirAll(filepath.Join(tx.stage, "plugin"), 0o750))
				staged = filepath.Join(tx.stage, "plugin", kind+".txt")
				require.NoError(t, os.WriteFile(staged, []byte("external"), 0o600))
				if kind == "external-replace" {
					require.NoError(t, os.WriteFile(filepath.Join(output, "plugin", kind+".txt"), []byte("prior"), 0o600))
				}
				artifacts = append(artifacts, codegen.Artifact{Path: staged, SkipExist: kind == "external-create"})
			}
			_, err = tx.commit(artifacts)
			require.NoError(t, err)
			content, err := os.ReadFile(filepath.Join(output, "gen", "current.txt"))
			require.NoError(t, err)
			require.Equal(t, "generated", string(content))
			require.NoFileExists(t, filepath.Join(output, "gen", "old.txt"))
			if kind == "external-create" || kind == "external-replace" {
				content, err = os.ReadFile(filepath.Join(output, "plugin", kind+".txt"))
				require.NoError(t, err)
				require.Equal(t, "external", string(content))
			}
			require.NoError(t, tx.cleanup())
			requireNoGenerationArtifacts(t, output)
		})
	}
}

// TestGenerationMountedTreePreservesLive requires a mount directly at the
// LOOM_TEST_GENERATION_TREE output's gen directory.
func TestGenerationMountedTreePreservesLive(t *testing.T) {
	output := os.Getenv("LOOM_TEST_GENERATION_TREE")
	if output == "" {
		t.Skip("requires a gen mount; see generation publication roadmap")
	}
	original := filepath.Join(output, "gen", "original.txt")
	require.NoError(t, os.WriteFile(original, []byte("keep mounted generation"), 0o600))
	before, err := os.Stat(original)
	require.NoError(t, err)
	tx, err := newGenerationTransaction(output)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, tx.cleanup())
	})
	require.NoError(t, os.MkdirAll(filepath.Join(tx.stage, "gen"), 0o750))
	staged := filepath.Join(tx.stage, "gen", "replacement.txt")
	require.NoError(t, os.WriteFile(staged, []byte("new"), 0o600))
	_, err = tx.commit([]codegen.Artifact{{Path: staged}})
	require.ErrorContains(t, err, "preserve generation tree")
	after, err := os.Stat(original)
	require.NoError(t, err)
	require.True(t, os.SameFile(before, after))
	content, err := os.ReadFile(original)
	require.NoError(t, err)
	require.Equal(t, "keep mounted generation", string(content))
	require.NoFileExists(t, filepath.Join(output, "gen", "replacement.txt"))
	require.NoError(t, tx.cleanup())
	requireNoGenerationArtifacts(t, output)
}
