package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/internal/testingx"
)

func TestGenerateRejectsDuplicateErrorDeclarations(t *testing.T) {
	repositoryRoot, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	for _, tc := range []struct {
		name   string
		scope  string
		design string
	}{
		{"api", "design:", `var _ = API("errors", func() {
	Error("boom")
	Error("boom")
})`},
		{"service", `service "svc":`, `var _ = Service("svc", func() {
	Error("boom", String)
	Error("boom", Int)
	Method("call", func() {
		Result(String)
		HTTP(func() {
			GET("/call")
		})
	})
})`},
		{"method", `service "svc" method "call":`, `var _ = Service("svc", func() {
	Method("call", func() {
		Result(String)
		Error("boom")
		Error("boom")
		HTTP(func() {
			GET("/call")
			Response("boom", StatusBadRequest)
		})
	})
})`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.Mkdir(filepath.Join(dir, "design"), 0o750))
			mod := "module example.com/erroradmission\n\ngo 1.27\n\n" +
				"require github.com/CaliLuke/loom v0.0.0\n\n" +
				"replace github.com/CaliLuke/loom => " + repositoryRoot + "\n"
			require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0o600))
			design := "package design\n\nimport . \"github.com/CaliLuke/loom/dsl\"\n\n" + tc.design + "\n"
			require.NoError(t, os.WriteFile(filepath.Join(dir, "design", "design.go"), []byte(design), 0o600))
			output, err := testingx.RunCmd(dir, "go", "mod", "tidy")
			require.NoError(t, err, output)
			t.Chdir(dir)
			_, stderr, err := captureOutput(t, func() error {
				return generate("gen", "example.com/erroradmission/design", dir, false)
			})
			require.Error(t, err)
			diagnostic := err.Error() + "\n" + stderr
			require.Contains(t, diagnostic, "stage eval.RunDSL")
			require.Contains(t, diagnostic, tc.scope)
			require.Contains(t, diagnostic, `error "boom" defined twice`)
			require.NotContains(t, diagnostic, "PANIC")
			require.NoDirExists(t, filepath.Join(dir, "gen"))
		})
	}
}
