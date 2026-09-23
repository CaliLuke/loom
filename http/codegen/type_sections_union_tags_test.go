package codegen

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	servicecodegen "github.com/CaliLuke/loom/codegen/service"
)

func TestHTTPUnionJSONBodiesQuoteDiscriminatorKeys(t *testing.T) {
	cases := map[string]struct {
		typeKey  string
		valueKey string
	}{
		"default keys":    {typeKey: "type", valueKey: "value"},
		"backtick keys":   {typeKey: "ty`pe", valueKey: "val`ue"},
		"quote keys":      {typeKey: `ty"pe`, valueKey: `val"ue`},
		"backslash, line": {typeKey: `ty\pe`, valueKey: "val\nue"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			data := &servicecodegen.UnionTypeData{Name: "U", KindName: "UKind", TypeKey: tc.typeKey, ValueKey: tc.valueKey}
			for body, render := range map[string]func(*servicecodegen.UnionTypeData) string{
				"marshal":   renderHTTPUnionMarshalJSONBody,
				"unmarshal": renderHTTPUnionUnmarshalJSONBody,
			} {
				src := "package p\n\nfunc f(u *U, data []byte) ([]byte, error) {\n" + render(data) + "\n}\n"
				file, err := parser.ParseFile(token.NewFileSet(), "p.go", src, 0)
				require.NoError(t, err, src)
				var tags []reflect.StructTag
				ast.Inspect(file, func(n ast.Node) bool {
					if field, ok := n.(*ast.Field); ok && field.Tag != nil {
						raw, err := strconv.Unquote(field.Tag.Value)
						require.NoError(t, err)
						tags = append(tags, reflect.StructTag(raw))
					}
					return true
				})
				require.Len(t, tags, 2, body)
				got, ok := tags[0].Lookup("json")
				require.True(t, ok, body)
				require.Equal(t, tc.typeKey, got, body)
				got, ok = tags[1].Lookup("json")
				require.True(t, ok, body)
				require.Equal(t, tc.valueKey, got, body)
			}
		})
	}
}
