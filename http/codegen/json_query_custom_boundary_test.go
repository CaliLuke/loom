package codegen

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
)

// TestJSONQueryCarrierCustomBoundary derives the opaque JSON query boundary
// from the emitted representation owner rather than raw metadata. Composite
// metadata that does not replace the emitted map keeps the framework physical
// carrier, its member null checks and its validation on both server and CLI;
// an effective custom element codec stays visible inside that carrier.
func TestJSONQueryCarrierCustomBoundary(t *testing.T) {
	replacement := expr.MetaExpr{"struct:field:type": {"url.Values", "net/url"}}
	composite := &expr.AttributeExpr{Type: &expr.Map{
		KeyType:  &expr.AttributeExpr{Type: expr.String},
		ElemType: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}},
	}, Meta: replacement}
	require.Empty(t, representation.CustomCodec(composite), "composite metadata selects no emitted custom root")
	text := &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{"struct:field:type": {"label.Label", "example.com/customs/label"}}}
	require.NotEmpty(t, representation.CustomCodec(text), "a primitive replacement is an effective custom codec")

	root := RunHTTPDSL(t, jsonQueryCustomBoundaryDSL)
	dir := t.TempDir()
	renderHTTPModuleFiles(t, dir, "example.com/customs/gen", root)
	read := func(path ...string) string {
		data, err := os.ReadFile(filepath.Join(append([]string{dir, "gen"}, path...)...))
		require.NoError(t, err)
		return string(data)
	}
	server, cli, service := read("http", "customs", "server", "encode_decode.go"), read("http", "customs", "client", "cli.go"), read("customs", "service.go")

	require.Regexp(t, `Values\s+map\[string\]\[\]string`, service, "the emitted service field is the framework map")
	require.Regexp(t, `Labels\s+map\[string\]label\.Label`, service, "the custom element type is emitted")
	for _, name := range []string{"values", "labels"} {
		serverCarrier := regexp.MustCompile(`var decoded (.+)\n\s*present, decodeErr := loomhttp\.DecodeJSONField\(qp, "` + name + `"`).FindStringSubmatch(server)
		require.Len(t, serverCarrier, 2, name)
		require.Contains(t, serverCarrier[1], "loom.Optional", "%s: framework-owned members keep null checks", name)
		cliCarrier := regexp.MustCompile(`var decoded (.+)\n\s*err = loomhttpcli\.UnmarshalNonNullableJSON\(\[\]byte\(customsRead` + map[string]string{"values": "Values", "labels": "Labels"}[name] + `\), &decoded\)`).FindStringSubmatch(cli)
		require.Len(t, cliCarrier, 2, "%s: the CLI decodes through the same physical carrier", name)
		require.Equal(t, serverCarrier[1], cliCarrier[1], name)
	}
	require.Contains(t, server, `"labels"`)
	require.Regexp(t, `var decoded map\[string\]loom\.Optional\[label\.Label\]`, server, "the element codec stays inside the carrier")
	require.NotContains(t, server, "values = decoded", "inert composite metadata must not bypass the physical carrier")
	values := server[regexp.MustCompile(`DecodeJSONField\(qp, "values"`).FindStringIndex(server)[0]:]
	require.Regexp(t, `^[\s\S]*?InvalidLengthError\("values[\s\S]*?values = make\(`, values, "validation precedes conversion of the composite carrier")
}

func jsonQueryCustomBoundaryDSL() {
	Service("customs", func() {
		Method("read", func() {
			Payload(func() {
				Attribute("values", MapOf(String, ArrayOf(String)), func() {
					Meta("struct:field:type", "url.Values", "net/url")
					MinLength(1)
				})
				Attribute("labels", MapOf(String, String), func() {
					Elem(func() {
						Meta("struct:field:type", "label.Label", "example.com/customs/label")
					})
				})
			})
			HTTP(func() {
				GET("/read")
				Param("values")
				Param("labels")
				Response(StatusNoContent)
			})
		})
	})
}
