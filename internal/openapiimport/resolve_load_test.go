package openapiimport

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolverConfinement(t *testing.T) {
	source := resolverFiles(t, resolverEntry(""), map[string]string{"models.yaml": "X: {type: string}", "nested/models.yaml": "X: {type: integer}"})
	root := filepath.Dir(source.RetrievalPath)
	source.Documents = []DocumentMapping{{URI: "https://e.test/safe/", Path: root}}
	for _, value := range []string{
		"https://e.test/safe/../secret.yaml", "https://e.test/safe/%2e%2e/secret.yaml",
		"https://e.test/safe/a%2F..%2Fsecret.yaml", "https://e.test/safe/%5Csecret.yaml",
		"https://e.test/safe//models.yaml", "https://e.test/safe/models.yaml?x=1",
		"https://e.test/safe/C%3Asecret.yaml", "https://user:secret@e.test/safe/models.yaml",
		"file://elsewhere/models.yaml", "file:///outside/models.yaml",
	} {
		t.Run(value, func(t *testing.T) {
			_, _, err := loadDocument(source, docURI(value))
			require.Error(t, err)
			require.NotContains(t, err.Error(), "user:secret")
		})
	}
	outside := filepath.Join(t.TempDir(), "secret.yaml")
	require.NoError(t, os.WriteFile(outside, []byte("secret"), 0o600))
	require.NoError(t, os.Symlink(outside, filepath.Join(root, "escape.yaml")))
	_, _, err := loadDocument(source, "https://e.test/safe/escape.yaml")
	require.Error(t, err)
	require.NoError(t, os.Symlink("models.yaml", filepath.Join(root, "alias.yaml")))
	data, real, err := loadDocument(source, "https://e.test/safe/models.yaml")
	require.NoError(t, err)
	alias, aliasReal, err := loadDocument(source, "https://e.test/safe/alias.yaml")
	require.NoError(t, err)
	require.Equal(t, data, alias)
	require.Equal(t, real, aliasReal)

	t.Run("mappingPrecedence", func(t *testing.T) {
		exact := filepath.Join(root, "models.yaml")
		source.Documents = append(source.Documents,
			DocumentMapping{URI: "https://e.test/safe/nested/", Path: root},
			DocumentMapping{URI: "https://e.test/safe/nested/models.yaml", Path: exact})
		got, _, err := loadDocument(source, "https://e.test/safe/nested/models.yaml")
		require.NoError(t, err)
		require.Equal(t, data, got)
		source.Documents = source.Documents[:2]
		got, _, err = loadDocument(source, "https://e.test/safe/nested/models.yaml")
		require.NoError(t, err)
		require.Equal(t, data, got, "longest prefix supplies the root models.yaml")
	})
	t.Run("duplicateMappings", func(t *testing.T) {
		source.Documents = []DocumentMapping{{URI: "https://e.test/safe/", Path: root}, {URI: "https://E.TEST:443/safe/", Path: root}}
		_, diagnostics, err := resolveDescription(source)
		require.True(t, err != nil || len(diagnostics) > 0)
	})
	t.Run("rootContainsEntry", func(t *testing.T) {
		source.Documents = nil
		source.ReferenceRoot = t.TempDir()
		_, diagnostics, err := resolveDescription(source)
		require.True(t, err != nil || len(diagnostics) > 0)
	})
	t.Run("fileURI", func(t *testing.T) {
		source.ReferenceRoot = ""
		uri := (&url.URL{Scheme: "file", Path: filepath.Join(root, "models.yaml")}).String()
		got, _, err := loadDocument(source, docURI(uri))
		require.NoError(t, err)
		require.Equal(t, data, got)
	})
}

func TestResolverBudgets(t *testing.T) {
	for _, tc := range []struct {
		name        string
		count, size int
	}{
		{"documents", 257, 0}, {"perDocument", 1, 16*1024*1024 + 1},
		{"aggregate", 5, 14 * 1024 * 1024},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := make(map[string]string)
			refs := make([]string, 0, tc.count)
			for i := range tc.count {
				name := fmt.Sprintf("doc%d.yaml", i)
				files[name] = "X: {type: string}\n#" + strings.Repeat("x", tc.size)
				refs = append(refs, name+"#/X")
			}
			r, diagnostics, err := resolveDescription(resolverFiles(t, resolverEntry("", refs...), files))
			require.NoError(t, err)
			require.Nil(t, r)
			require.Contains(t, diagnostics.Error(), "resolution budget exceeded")
		})
	}
	t.Run("hops", func(t *testing.T) {
		files := make(map[string]string)
		for i := range 66 {
			files[fmt.Sprintf("doc%d.yaml", i)] = fmt.Sprintf("X: {$ref: 'doc%d.yaml#/X'}", i+1)
		}
		files["doc66.yaml"] = "X: {type: string}"
		r, diagnostics, err := resolveDescription(resolverFiles(t, resolverEntry("", "doc0.yaml#/X"), files))
		require.NoError(t, err)
		require.Nil(t, r)
		require.Contains(t, diagnostics.Error(), "resolution budget exceeded")
	})
}

type resolverCountingTransport struct {
	calls int
}

func (r *resolverCountingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	r.calls++
	return nil, fmt.Errorf("network must not be used")
}

func TestResolverNoNetwork(t *testing.T) {
	prior := http.DefaultTransport
	transport := &resolverCountingTransport{}
	http.DefaultTransport = transport
	t.Cleanup(func() {
		http.DefaultTransport = prior
	})
	r, diagnostics, err := resolveDescription(Source{Bytes: resolverEntry("", "https://never.invalid/models#/X")})
	require.NoError(t, err)
	require.NotNil(t, r)
	require.Contains(t, diagnostics.Error(), "not supplied")
	require.Zero(t, transport.calls)
}

func TestResolverSharedReferenceGraph(t *testing.T) {
	var schemas strings.Builder
	for i := range 30 {
		fmt.Fprintf(&schemas, "N%d: {type: object, properties: {left: {$ref: '#/N%d'}, right: {$ref: '#/N%d'}}}\n", i, i+1, i+1)
	}
	schemas.WriteString("N30: {type: string}\n")
	resolved, diagnostics, err := resolveDescription(resolverFiles(t, resolverEntry("", "models.yaml#/N0"), map[string]string{"models.yaml": schemas.String()}))
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	require.NotNil(t, resolved)
}
