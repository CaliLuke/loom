package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/internal/testingx"
)

func TestArtifactWriteIntentRealPlugin(t *testing.T) {
	root := testingx.RepoRoot()
	bin := filepath.Join(t.TempDir(), "loom")
	out, err := testingx.RunCmd(root, "go", "build", "-o", bin, "./cmd/loom")
	require.NoError(t, err, out)
	dir := t.TempDir()
	writeRoundTripModule(t, dir, "example.com/artifacts", root)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "design"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "design", "design.go"), []byte(artifactPluginDesign), 0o600))
	out, err = testingx.RunCmd(dir, "go", "mod", "tidy")
	require.NoError(t, err, out)
	out, err = testingx.RunCmd(dir, bin, "gen", "example.com/artifacts/design", "-o", ".")
	require.NoError(t, err, out)
	require.Contains(t, strings.Split(strings.TrimSpace(out), "\n"), "custom.txt")
	got, err := os.ReadFile(filepath.Join(dir, "custom.txt"))
	require.NoError(t, err)
	require.Equal(t, "template", string(got))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "custom.txt"), []byte("user"), 0o600))
	out, err = testingx.RunCmd(dir, bin, "gen", "example.com/artifacts/design", "-o", ".")
	require.NoError(t, err, out)
	got, err = os.ReadFile(filepath.Join(dir, "custom.txt"))
	require.NoError(t, err)
	require.Equal(t, "user", string(got))
	require.NotContains(t, strings.Split(strings.TrimSpace(out), "\n"), "custom.txt")
	out, err = testingx.RunCmd(dir, "go", "mod", "tidy")
	require.NoError(t, err, out)
	out, err = testingx.RunCmd(dir, "go", "build", "./...")
	require.NoError(t, err, out)
	out, err = testingx.RunCmd(dir, bin, "test-scaffold", "example.com/artifacts/design", "-o", ".")
	require.NoError(t, err, out)
	require.Contains(t, out, "internal/contracttest/")
	out, err = testingx.RunCmd(dir, "go", "test", "-run", "^$", "./...")
	require.NoError(t, err, out)
	manifest, err := os.ReadFile(filepath.Join(dir, "gen", "loom.json"))
	require.NoError(t, err)
	rejected := strings.Replace(artifactPluginDesign, `Path: "custom.txt"`, `Path: "gen/custom.txt"`, 1)
	require.NotEqual(t, artifactPluginDesign, rejected)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "design", "design.go"), []byte(rejected), 0o600))
	out, err = testingx.RunCmd(dir, bin, "gen", "example.com/artifacts/design", "-o", ".")
	require.Error(t, err)
	require.Contains(t, out, "create-only artifact gen/custom.txt must be outside gen/")
	after, err := os.ReadFile(filepath.Join(dir, "gen", "loom.json"))
	require.NoError(t, err)
	require.Equal(t, manifest, after)
	require.NoFileExists(t, filepath.Join(dir, "gen", "custom.txt"))
}

const artifactPluginDesign = `package design

import (
 . "github.com/CaliLuke/loom/dsl"
 "github.com/CaliLuke/loom/codegen"
 "github.com/CaliLuke/loom/eval"
)

var _ = API("artifacts", func() {
})
var _ = Service("echo", func() {
 Method("show", func() {
  Result(String)
  HTTP(func() {
   GET("/show")
  })
 })
})

func init() {
 codegen.RegisterPlugin("write-intent", "gen", nil, func(_ string, _ []eval.Root, files []*codegen.File) ([]*codegen.File, error) {
  return append(files, &codegen.File{
   Path: "custom.txt",
   SkipExist: true,
   Sections: []codegen.Section{codegen.NewRawSection("content", "template")},
  }), nil
 })
}
`
