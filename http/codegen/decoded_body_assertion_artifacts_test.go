package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/stretchr/testify/require"
)

func TestDecodedBodyAssertionArtifacts(t *testing.T) {
	output := os.Getenv("LOOM_DECODED_BODY_OUTPUT")
	if output == "" {
		t.Skip("set LOOM_DECODED_BODY_OUTPUT for isolated artifact comparison")
	}
	for _, tc := range []struct {
		name   string
		design func()
	}{
		{"inner", testdata.PayloadBodyUserInnerDefaultDSL},
		{"recursive", testdata.PayloadBodyInlineRecursiveUserDSL},
		{"optional", optionalValueBodyModuleDSL("v")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(output, tc.name)
			require.NoError(t, os.MkdirAll(dir, 0o700))
			renderHTTPModule(t, dir, "example.com/optvaluebody", RunHTTPDSL(t, tc.design))
			path := filepath.Join(dir, "go.mod")
			manifest, err := os.ReadFile(path)
			require.NoError(t, err)
			manifest = []byte(strings.Replace(string(manifest), "github.com/CaliLuke/loom v1.0.0", "github.com/CaliLuke/loom v1.10.0-alpha.9.0.20261013020843-063198a2f427", 1))
			require.NoError(t, os.WriteFile(path, manifest, 0o600))
		})
	}
}

func TestDecodedBodyAssertionArtifactBuilds(t *testing.T) {
	output := os.Getenv("LOOM_DECODED_BODY_OUTPUT")
	if output == "" {
		t.Skip("set LOOM_DECODED_BODY_OUTPUT to compile the comparison modules")
	}
	for _, name := range []string{"inner", "recursive", "optional"} {
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join(output, name)
			runGoCommand(t, dir, "mod", "tidy")
			runGoCommand(t, dir, "build", "./...")
			runGoCommand(t, dir, "vet", "./...")
		})
	}
}
