package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/internal/testingx"
)

func TestGenerateRejectsDuplicateUnionBranches(t *testing.T) {
	repositoryRoot, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	for _, tc := range []struct {
		name     string
		branches string
	}{
		{"name", `Attribute("same", String)
   Attribute("same", Int)`},
		{"tag", `Attribute("left", String, func() {
    Meta("oneof:type:tag", "same")
   })
   Attribute("right", Int, func() {
    Meta("oneof:type:tag", "same")
   })`},
		{"name_with_distinct_tags", `Attribute("same", String, func() {
    Meta("oneof:type:tag", "left")
   })
   Attribute("same", Int, func() {
    Meta("oneof:type:tag", "right")
   })`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.Mkdir(filepath.Join(dir, "design"), 0o750))
			mod := "module example.com/unionadmission\n\ngo 1.27\n\n" +
				"require github.com/CaliLuke/loom v0.0.0\n\n" +
				"replace github.com/CaliLuke/loom => " + repositoryRoot + "\n"
			require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0o600))
			design := `package design

import . "github.com/CaliLuke/loom/dsl"

var _ = Service("svc", func() {
 Method("call", func() {
  Payload(func() {
   OneOf("choice", func() {
` + tc.branches + `
   })
  })
  HTTP(func() {
   POST("/call")
  })
 })
})
`
			require.NoError(t, os.WriteFile(filepath.Join(dir, "design", "design.go"), []byte(design), 0o600))
			output, err := testingx.RunCmd(dir, "go", "mod", "tidy")
			require.NoError(t, err, output)
			t.Chdir(dir)
			_, stderr, err := captureOutput(t, func() error {
				return generate("gen", "example.com/unionadmission/design", dir, false)
			})
			require.Error(t, err)
			diagnostic := err.Error() + "\n" + stderr
			require.Contains(t, diagnostic, "stage eval.RunDSL")
			require.Contains(t, diagnostic, "design.go:")
			require.Contains(t, diagnostic, "OneOf")
			require.Contains(t, diagnostic, "same")
			require.NotContains(t, diagnostic, "PANIC")
			require.NotContains(t, diagnostic, "panic:")
			require.NoDirExists(t, filepath.Join(dir, "gen"))
		})
	}
}
