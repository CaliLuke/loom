package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
	"github.com/CaliLuke/loom/internal/testingx"
)

func TestLoomGenIsByteStableAcrossIndependentModules(t *testing.T) {
	repoRoot := testingx.RepoRoot()
	loomBin := filepath.Join(t.TempDir(), "loom")
	output, err := testingx.RunCmd(repoRoot, "go", "build", "-o", loomBin, "./cmd/loom")
	require.NoError(t, err, output)

	tests := []struct {
		name        string
		target      string
		prefix      string
		indent      string
		implicitAPI bool
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
		{
			name:        "implicit API and server naming",
			implicitAPI: true,
			wantVersion: openapiv3.OpenAPIVersion,
			wantStart:   "{\n  \"openapi\"",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			first := generateWithLoomCLI(
				t, repoRoot, loomBin, test.target, test.prefix, test.indent, test.implicitAPI, false,
			)
			second := generateWithLoomCLI(
				t, repoRoot, loomBin, test.target, test.prefix, test.indent, test.implicitAPI, true,
			)

			require.Equal(t, first, second)
			openAPI := first[filepath.Join("http", "openapi.json")]
			require.NotEmpty(t, openAPI)
			require.True(t, strings.HasPrefix(string(openAPI), test.wantStart))
			require.Contains(t, string(openAPI), fmt.Sprintf(`"openapi": "%s"`, test.wantVersion))
			alpha := strings.Index(string(openAPI), `"alpha": "first"`)
			zulu := strings.Index(string(openAPI), `"zulu": "last"`)
			require.NotEqual(t, -1, alpha)
			require.NotEqual(t, -1, zulu)
			require.Less(t, alpha, zulu, "nested authored maps must use deterministic key ordering")
			alphaTag := strings.Index(string(openAPI), `"name": "Alpha"`)
			apiTag := strings.Index(string(openAPI), `"name": "API"`)
			catalogTag := strings.Index(string(openAPI), `"name": "Catalog"`)
			zebraTag := strings.Index(string(openAPI), `"name": "Zebra"`)
			require.NotEqual(t, -1, alphaTag)
			require.NotEqual(t, -1, apiTag)
			require.NotEqual(t, -1, catalogTag)
			require.NotEqual(t, -1, zebraTag)
			require.Less(t, apiTag, alphaTag, "implicit service tags must use stable name ordering")
			require.Less(t, alphaTag, catalogTag, "implicit service tags must use stable name ordering")
			require.Less(t, catalogTag, zebraTag, "implicit service tags must use stable name ordering")
		})
	}
}

func generateWithLoomCLI(
	t *testing.T,
	repoRoot string,
	loomBin string,
	target string,
	prefix string,
	indent string,
	implicitAPI bool,
	reverseServices bool,
) map[string][]byte {
	t.Helper()

	moduleDir := t.TempDir()
	const modulePath = "example.com/openapi-determinism"
	writeRoundTripModule(t, moduleDir, modulePath, repoRoot)
	designDir := filepath.Join(moduleDir, "design")
	require.NoError(t, os.MkdirAll(designDir, 0o700))
	require.NoError(t, os.WriteFile(
		filepath.Join(designDir, "design.go"),
		[]byte(openAPIDeterminismDesign(target, prefix, indent, implicitAPI, reverseServices)),
		0o600,
	))

	output, err := testingx.RunCmd(moduleDir, "go", "mod", "tidy")
	require.NoError(t, err, output)
	output, err = testingx.RunCmd(moduleDir, loomBin, "gen", modulePath+"/design", "-o", ".")
	require.NoError(t, err, output)
	generated := make(map[string][]byte)
	genDir := filepath.Join(moduleDir, "gen")
	err = filepath.WalkDir(genDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relative, relErr := filepath.Rel(genDir, path)
		if relErr != nil {
			return relErr
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		generated[relative] = content
		return nil
	})
	require.NoError(t, err)
	return generated
}

func openAPIDeterminismDesign(target, prefix, indent string, implicitAPI, reverseServices bool) string {
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

	header := `package design

import . "github.com/CaliLuke/loom/dsl"

`
	if !implicitAPI {
		header += fmt.Sprintf(`var _ = API("Determinism", func() {
%s	Meta("openapi:extension:x-order", "{\"zulu\":\"last\",\"alpha\":\"first\"}")
})

`, metadata.String())
	}
	services := []string{`var _ = Service("Catalog", func() {
	Method("Show", func() {
		Result(MapOf(String, String), func() {
			Example(map[string]any{"zulu": "last", "alpha": "first"})
		})
		HTTP(func() {
			GET("/catalog")
			Response(StatusOK)
		})
	})
})`, `var _ = Service("Zebra", func() {
	Method("Show", func() {
		HTTP(func() {
			GET("/zebra")
			Response(StatusNoContent)
		})
	})
})`, `var _ = Service("Alpha", func() {
	Method("Show", func() {
		HTTP(func() {
			GET("/alpha")
			Response(StatusNoContent)
		})
	})
})`, `var _ = Service("API", func() {
	Method("Show", func() {
		HTTP(func() {
			GET("/api")
			Response(StatusNoContent)
		})
	})
})`}
	if reverseServices {
		slices.Reverse(services)
	}
	return header + strings.Join(services, "\n\n") + "\n"
}
