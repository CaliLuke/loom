package openapiimport

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func resolverEntry(self string, references ...string) []byte {
	var source strings.Builder
	source.WriteString("openapi: 3.2.0\ninfo: {title: Resolver, version: '1'}\npaths: {}\n")
	if self != "" {
		fmt.Fprintf(&source, "$self: %q\n", self)
	}
	source.WriteString("components:\n  schemas:\n    Pet: {type: string}\n")
	for i, ref := range references {
		fmt.Fprintf(&source, "    R%d: {$ref: %q}\n", i, ref)
	}
	return []byte(source.String())
}

func resolverFiles(t *testing.T, entry []byte, files map[string]string) Source {
	t.Helper()
	root := t.TempDir()
	for name, value := range files {
		path := filepath.Join(root, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
		require.NoError(t, os.WriteFile(path, []byte(value), 0o600))
	}
	path := filepath.Join(root, "openapi.yaml")
	require.NoError(t, os.WriteFile(path, entry, 0o600))
	return Source{Bytes: entry, RetrievalPath: path}
}

func resolverEdge(t *testing.T, r *resolution, authored string) *edge {
	t.Helper()
	for _, e := range r.edges {
		if e.authored == authored {
			return e
		}
	}
	t.Fatalf("missing edge %q", authored)
	return nil
}

func TestResolverIdentity(t *testing.T) {
	t.Run("spellingKeys", func(t *testing.T) {
		source := resolverFiles(t, resolverEntry("", "models.yaml#/X", "./sub/../models.yaml#/X"), map[string]string{"models.yaml": "X: {type: string}"})
		r, diagnostics, err := resolveDescription(source)
		require.NoError(t, err)
		require.Empty(t, diagnostics)
		require.Equal(t, resolverEdge(t, r, "models.yaml#/X").target, resolverEdge(t, r, "./sub/../models.yaml#/X").target)
		require.Len(t, r.registry.docs, 2)
	})
	t.Run("earlyFail", func(t *testing.T) {
		source := resolverFiles(t, resolverEntry("", "https://e.test/shared#/components/schemas/Pet", "b.yaml#/components/schemas/Pet"), map[string]string{"b.yaml": string(resolverEntry("https://e.test/shared"))})
		r, diagnostics, err := resolveDescription(source)
		require.NoError(t, err)
		require.Empty(t, diagnostics)
		require.Equal(t, resolverEdge(t, r, "https://e.test/shared#/components/schemas/Pet").target, resolverEdge(t, r, "b.yaml#/components/schemas/Pet").target)
	})
	for _, tc := range []struct {
		name, self, b, c string
		refs             []string
	}{
		{"noBaseCheck", "", "X: {type: string}", "", []string{"b.yaml#/X", "https://e.test/b.yaml#/X"}},
		{"canonicalDedupe", "", string(resolverEntry("https://e.test/shared")), string(resolverEntry("https://e.test/shared")), []string{"b.yaml#/components/schemas/Pet", "c.yaml#/components/schemas/Pet"}},
		{"aliasShortcut", "https://e.test/b.yaml", string(resolverEntry("")), "", []string{"https://e.test/b.yaml#/components/schemas/Pet"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := resolverFiles(t, resolverEntry(tc.self, tc.refs...), map[string]string{"b.yaml": tc.b, "c.yaml": tc.c})
			source.Documents = []DocumentMapping{{URI: "https://e.test/", Path: filepath.Dir(source.RetrievalPath)}}
			r, diagnostics, err := resolveDescription(source)
			require.NoError(t, err)
			require.Nil(t, r)
			require.Contains(t, diagnostics.Error(), "document-identity")
		})
	}
	t.Run("sharedSelfAddsAliases", func(t *testing.T) {
		source := resolverFiles(t, resolverEntry("", "b.yaml#/components/schemas/Pet", "https://e.test/b.yaml#/components/schemas/Pet"), map[string]string{"b.yaml": string(resolverEntry("https://e.test/shared"))})
		source.Documents = []DocumentMapping{{URI: "https://e.test/", Path: filepath.Dir(source.RetrievalPath)}}
		r, diagnostics, err := resolveDescription(source)
		require.NoError(t, err)
		require.Empty(t, diagnostics)
		require.Len(t, r.registry.docs, 2)
		require.Equal(t, resolverEdge(t, r, "b.yaml#/components/schemas/Pet").target, resolverEdge(t, r, "https://e.test/b.yaml#/components/schemas/Pet").target)
	})
}

func TestResolverNoBase(t *testing.T) {
	for _, tc := range []struct{ name, self, base string }{
		{"absent", "", ""}, {"relative", "openapi.json", ""},
		{"absolute", "https://e.test/api/openapi.json", "https://e.test/api/openapi.json"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, diagnostics, err := resolveDescription(Source{Bytes: resolverEntry(tc.self, "#/components/schemas/Pet", "openapi.json#/components/schemas/Pet", "https://e.test/api/openapi.json#/components/schemas/Pet", "models.yaml#/X", "https://e.test/x.yaml#/X")})
			require.NoError(t, err)
			require.NotNil(t, r)
			require.Equal(t, docURI(tc.base), r.registry.docs["entry"].base)
			require.Equal(t, &targetKey{doc: "entry", pointer: "/components/schemas/Pet"}, resolverEdge(t, r, "#/components/schemas/Pet").target)
			if tc.base == "" {
				require.Empty(t, r.registry.alias, "inventBase control")
				require.Contains(t, diagnostics.Error(), "the entry has no base URI")
			} else {
				require.Equal(t, sourceID("entry"), r.registry.alias[docURI(tc.base)])
				require.Equal(t, resolverEdge(t, r, "#/components/schemas/Pet").target, resolverEdge(t, r, "openapi.json#/components/schemas/Pet").target)
			}
			require.Contains(t, diagnostics.Error(), "not supplied")
		})
	}
}
