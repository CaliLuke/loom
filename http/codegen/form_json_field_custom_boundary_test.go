package codegen

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

// TestFormJSONFieldCustomBoundary requires form JSON fields to take their
// opaque boundary from the emitted representation owner, like JSON query
// values. Composite Go type metadata that does not replace the emitted map
// keeps the field's JSON carrier and its member null checks; custom element
// types stay inside the carrier, and a custom text field keeps its own codec.
func TestFormJSONFieldCustomBoundary(t *testing.T) {
	root := RunHTTPDSL(t, formJSONFieldCustomBoundaryDSL)
	dir := t.TempDir()
	renderHTTPModuleFiles(t, dir, "example.com/forms/gen", root)
	read := func(path ...string) string {
		data, err := os.ReadFile(filepath.Join(append([]string{dir, "gen"}, path...)...))
		require.NoError(t, err)
		return string(data)
	}
	types, service := read("http", "forms", "server", "types.go"), read("forms", "service.go")

	require.Regexp(t, `Values\s+map\[string\]\[\]string`, service, "the emitted service field is the framework map")
	field := func(name string) string {
		match := regexp.MustCompile(`(?m)^\t` + name + `\s+(\S+)\s+` + "`").FindStringSubmatch(types)
		require.Len(t, match, 2, name)
		return match[1]
	}
	declaration := func(typ string) string {
		match := regexp.MustCompile(`(?m)^type ` + regexp.QuoteMeta(typ) + ` (.+)$`).FindStringSubmatch(types)
		require.Len(t, match, 2, typ)
		return match[1]
	}
	plain, values := field("Plain"), field("Values")
	require.NotEqual(t, "map[string][]string", values, "inert composite metadata keeps the JSON field carrier")
	require.Equal(t, declaration(plain), declaration(values), "the carrier matches the same map without metadata")
	require.Contains(t, declaration(values), "loom.Optional", "framework-owned members keep null checks")
	require.Contains(t, declaration(field("Labels")), "loom.Optional[label.Label]", "the element codec stays inside the carrier")
	require.Equal(t, "*label.Label", field("Note"), "a custom text field keeps its own codec")
	for _, pkg := range []string{"server", "client"} {
		require.Empty(t, undefinedPackageIdentifiers(t, filepath.Join(dir, "gen", "http", "forms", pkg)), pkg)
	}
}

func formJSONFieldCustomBoundaryDSL() {
	label := func() {
		Meta("struct:field:type", "label.Label", "example.com/forms/label")
	}
	Service("forms", func() {
		Method("submit", func() {
			Payload(func() {
				Attribute("plain", MapOf(String, ArrayOf(String)))
				Attribute("values", MapOf(String, ArrayOf(String)), func() {
					Meta("struct:field:type", "url.Values", "net/url")
				})
				Attribute("labels", MapOf(String, String), func() {
					Elem(label)
				})
				Attribute("note", String, label)
			})
			HTTP(func() {
				POST("/submit")
				FormRequest()
				Response(StatusNoContent)
			})
		})
	})
}
