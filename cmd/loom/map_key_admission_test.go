package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/internal/testingx"
)

const mapKeyControlDesign = `package design

import . "github.com/CaliLuke/loom/dsl"

var name = Type("Name", String)
var object = Type("ObjectKey", func() {
 Attribute("name", String)
 Attribute("values", ArrayOf(String))
})
var _ = Service("mapping", func() {
 Method("send", func() {
  Payload(func() {
   Attribute("text", MapOf(String, String))
   Attribute("flag", MapOf(Boolean, String))
   Attribute("number", MapOf(Float64, String))
   Attribute("named", MapOf(name, String))
   Attribute("object", MapOf(object, String))
   Attribute("dynamic", MapOf(Any, String), func() {
    Example(map[any]string{"key": "value"})
   })
   Attribute("custom", MapOf(Bytes, String, func() {
    Key(func() {
     Meta("struct:field:type", "[16]byte")
    })
   }), func() {
    Example(map[[16]byte]string{})
   })
  })
 })
})
`

func TestGenerateRejectsNativeMapKeys(t *testing.T) {
	for _, tc := range []struct {
		name        string
		declaration string
		key         string
	}{
		{"bytes", "", "Bytes"},
		{"array", "", "ArrayOf(String)"},
		{"named-bytes", `var key = Type("Key", Bytes)`, "key"},
		{"named-array", `var key = Type("Key", ArrayOf(String))`, "key"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			design := `package design
import . "github.com/CaliLuke/loom/dsl"
` + tc.declaration + `
var _ = Service("mapping", func() {
 Method("send", func() {
  Payload(func() {
   Attribute("entries", MapOf(` + tc.key + `, String))
  })
 })
})
`
			dir := mapKeyModule(t, design)
			t.Chdir(dir)
			_, stderr, err := captureOutput(t, func() error {
				return generate("gen", "example.com/mapkeys/design", dir, false)
			})
			require.Error(t, err)
			diagnostic := err.Error() + "\n" + stderr
			require.Contains(t, diagnostic, "stage eval.RunDSL")
			require.Contains(t, diagnostic, "map key")
			require.Contains(t, diagnostic, "entries")
			require.Contains(t, diagnostic, "design.go:")
			require.NotContains(t, diagnostic, "PANIC")
			require.NotContains(t, diagnostic, "panic:")
			require.NoDirExists(t, filepath.Join(dir, "gen"))
		})
	}
}

func TestGenerateMapKeyRepresentationControls(t *testing.T) {
	dir := mapKeyModule(t, mapKeyControlDesign)
	t.Chdir(dir)
	_, stderr, err := captureOutput(t, func() error {
		return generate("gen", "example.com/mapkeys/design", dir, false)
	})
	require.NoError(t, err, stderr)
	content, err := os.ReadFile(filepath.Join(dir, "gen", "mapping", "service.go"))
	require.NoError(t, err)
	for _, fragment := range []string{"map[string]string", "map[bool]string", "map[float64]string", "map[Name]string", "map[*ObjectKey]string", "map[any]string", "map[[16]byte]string"} {
		require.Contains(t, string(content), fragment)
	}
	output, err := testingx.RunCmd(dir, "go", "mod", "tidy")
	require.NoError(t, err, output)
	output, err = testingx.RunCmd(dir, "go", "build", "./...")
	require.NoError(t, err, output)
	output, err = testingx.RunCmd(dir, "go", "vet", "./...")
	require.NoError(t, err, output)
}

func mapKeyModule(t *testing.T, design string) string {
	t.Helper()
	repositoryRoot := testingx.RepoRoot()
	dir := t.TempDir()
	if capture := os.Getenv("LOOM_MAP_KEY_OUTPUT"); capture != "" {
		dir = filepath.Join(capture, strings.ReplaceAll(t.Name(), "/", "_"))
		require.NoError(t, os.MkdirAll(dir, 0o750))
	}
	require.NoError(t, os.Mkdir(filepath.Join(dir, "design"), 0o750))
	module := "module example.com/mapkeys\n\ngo 1.27\n\nrequire github.com/CaliLuke/loom v0.0.0\n\nreplace github.com/CaliLuke/loom => " + repositoryRoot + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(module), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "design", "design.go"), []byte(design), 0o600))
	output, err := testingx.RunCmd(dir, "go", "mod", "tidy")
	require.NoError(t, err, output)
	return dir
}
