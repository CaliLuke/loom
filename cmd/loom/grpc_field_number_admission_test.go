package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/internal/testingx"
)

func TestGenerateRejectsInvalidRPCFieldNumbers(t *testing.T) {
	repositoryRoot, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	for _, tc := range []struct {
		name       string
		fields     string
		diagnostic string
	}{
		{"zero", `Field(0, "value", String)`, "value"},
		{"nonnumeric", `Field("abc", "value", String)`, "value"},
		{"reserved", `Field(19500, "value", String)`, "value"},
		{"numeric_identity", `Field("01", "first", String)
 Field(1, "second", String)`, "already exists"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.Mkdir(filepath.Join(dir, "design"), 0o750))
			mod := "module example.com/fieldnumbers\n\ngo 1.27\n\n" +
				"require github.com/CaliLuke/loom v0.0.0\n\n" +
				"replace github.com/CaliLuke/loom => " + repositoryRoot + "\n"
			require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0o600))
			design := `package design

import . "github.com/CaliLuke/loom/dsl"

var _ = Service("svc", func() {
 Method("call", func() {
  Payload(func() {
` + tc.fields + `
  })
  GRPC(func() {
  })
 })
})
`
			require.NoError(t, os.WriteFile(filepath.Join(dir, "design", "design.go"), []byte(design), 0o600))
			output, err := testingx.RunCmd(dir, "go", "mod", "tidy")
			require.NoError(t, err, output)
			t.Chdir(dir)
			_, stderr, err := captureOutput(t, func() error {
				return generate("gen", "example.com/fieldnumbers/design", dir, false)
			})
			require.Error(t, err)
			diagnostic := err.Error() + "\n" + stderr
			require.Contains(t, diagnostic, "stage eval.RunDSL")
			require.Contains(t, diagnostic, "design.go:")
			require.Contains(t, diagnostic, tc.diagnostic)
			require.NotContains(t, diagnostic, "PANIC")
			require.NotContains(t, diagnostic, "panic:")
			require.NoDirExists(t, filepath.Join(dir, "gen"))
		})
	}
}

func TestGenerateRPCUnusedUnionSeeds(t *testing.T) {
	repositoryRoot, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	var baseline map[string]string
	for _, seed := range []string{"17", "abc", "18446744073709551616"} {
		t.Run(seed, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.Mkdir(filepath.Join(dir, "design"), 0o750))
			mod := "module example.com/unusedseed\n\ngo 1.27\n\n" +
				"require github.com/CaliLuke/loom v0.0.0\n\n" +
				"replace github.com/CaliLuke/loom => " + repositoryRoot + "\n"
			require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0o600))
			design := `package design

import . "github.com/CaliLuke/loom/dsl"

var _ = Service("svc", func() {
 Method("call", func() {
  Payload(func() {
   OneOf("choice", func() {
    Meta("rpc:tag", "` + seed + `")
    Field(1, "first", String)
    Field(2, "second", Int)
   })
  })
  GRPC(func() {
  })
 })
})
`
			require.NoError(t, os.WriteFile(filepath.Join(dir, "design", "design.go"), []byte(design), 0o600))
			output, err := testingx.RunCmd(dir, "go", "mod", "tidy")
			require.NoError(t, err, output)
			t.Chdir(dir)
			_, stderr, err := captureOutput(t, func() error {
				return generate("gen", "example.com/unusedseed/design", dir, false)
			})
			require.NoError(t, err, stderr)
			artifacts := make(map[string]string)
			generated := filepath.Join(dir, "gen")
			err = filepath.WalkDir(generated, func(path string, entry fs.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if entry.IsDir() || (filepath.Ext(path) != ".go" && filepath.Ext(path) != ".proto") {
					return nil
				}
				relative, err := filepath.Rel(generated, path)
				if err != nil {
					return err
				}
				content, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				artifacts[relative] = string(content)
				return nil
			})
			require.NoError(t, err)
			require.NotEmpty(t, artifacts)
			if seed == "17" {
				baseline = artifacts
			} else {
				require.Equal(t, baseline, artifacts, "unused seed must not change emitted Go or protobuf sources")
			}
			output, err = testingx.RunCmd(dir, "go", "mod", "tidy")
			require.NoError(t, err, output)
			output, err = testingx.RunCmd(dir, "go", "build", "./...")
			require.NoError(t, err, output)
			output, err = testingx.RunCmd(dir, "go", "vet", "./...")
			require.NoError(t, err, output)
		})
	}
}
