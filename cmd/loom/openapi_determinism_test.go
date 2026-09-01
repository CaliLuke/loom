package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
	"github.com/CaliLuke/loom/internal/testingx"
)

func TestLoomGenOpenAPIIsByteStableAcrossIndependentModules(t *testing.T) {
	repoRoot := testingx.RepoRoot()
	loomBin := filepath.Join(t.TempDir(), "loom")
	output, err := testingx.RunCmd(repoRoot, "go", "build", "-o", loomBin, "./cmd/loom")
	require.NoError(t, err, output)

	tests := []struct {
		name        string
		target      string
		prefix      string
		indent      string
		wantVersion string
		wantStart   string
	}{
		{
			name:        "OpenAPI 3.2 defaults",
			wantVersion: openapiv3.OpenAPIVersion,
			wantStart:   "{\n  \"openapi\"",
		},
		{
			name:        "OpenAPI 3.1 configured formatting",
			target:      "3.1",
			prefix:      " ",
			indent:      "\t",
			wantVersion: openapiv3.OpenAPICompatibilityVersion,
			wantStart:   "{\n \t\"openapi\"",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			first := generateOpenAPIWithLoomCLI(t, repoRoot, loomBin, test.target, test.prefix, test.indent)
			second := generateOpenAPIWithLoomCLI(t, repoRoot, loomBin, test.target, test.prefix, test.indent)

			require.Equal(t, first, second)
			require.True(t, strings.HasPrefix(string(first), test.wantStart))
			require.Contains(t, string(first), fmt.Sprintf(`"openapi": "%s"`, test.wantVersion))
			alpha := strings.Index(string(first), `"alpha": "first"`)
			zulu := strings.Index(string(first), `"zulu": "last"`)
			require.NotEqual(t, -1, alpha)
			require.NotEqual(t, -1, zulu)
			require.Less(t, alpha, zulu, "nested authored maps must use deterministic key ordering")
		})
	}
}

func generateOpenAPIWithLoomCLI(
	t *testing.T,
	repoRoot string,
	loomBin string,
	target string,
	prefix string,
	indent string,
) []byte {
	t.Helper()

	moduleDir := t.TempDir()
	const modulePath = "example.com/openapi-determinism"
	writeRoundTripModule(t, moduleDir, modulePath, repoRoot)
	designDir := filepath.Join(moduleDir, "design")
	require.NoError(t, os.MkdirAll(designDir, 0o700))
	require.NoError(t, os.WriteFile(
		filepath.Join(designDir, "design.go"),
		[]byte(openAPIDeterminismDesign(target, prefix, indent)),
		0o600,
	))

	output, err := testingx.RunCmd(moduleDir, "go", "mod", "tidy")
	require.NoError(t, err, output)
	output, err = testingx.RunCmd(moduleDir, loomBin, "gen", modulePath+"/design", "-o", ".")
	require.NoError(t, err, output)
	generated, err := os.ReadFile(filepath.Join(moduleDir, "gen", "http", "openapi.json"))
	require.NoError(t, err)
	return generated
}

func openAPIDeterminismDesign(target, prefix, indent string) string {
	var metadata strings.Builder
	if target != "" {
		fmt.Fprintf(&metadata, "\t\tMeta(%q, %q)\n", "openapi:version", target)
	}
	if prefix != "" {
		fmt.Fprintf(&metadata, "\t\tMeta(%q, %q)\n", "openapi:json:prefix", prefix)
	}
	if indent != "" {
		fmt.Fprintf(&metadata, "\t\tMeta(%q, %q)\n", "openapi:json:indent", indent)
	}

	return fmt.Sprintf(`package design

import . "github.com/CaliLuke/loom/dsl"

var _ = API("Determinism", func() {
%s	Meta("openapi:extension:x-order", "{\"zulu\":\"last\",\"alpha\":\"first\"}")
})

var _ = Service("Catalog", func() {
	Method("Show", func() {
		Result(MapOf(String, String), func() {
			Example(map[string]any{"zulu": "last", "alpha": "first"})
		})
		HTTP(func() {
			GET("/catalog")
			Response(StatusOK)
		})
	})
})
`, metadata.String())
}
