package uniongen

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/dave/jennifer/jen"
	"github.com/stretchr/testify/require"
)

func TestUnionKindConstCommentCannotInjectCode(t *testing.T) {
	cases := map[string]string{
		"plain":   "String",
		"hostile": "x\n*/ func Injected() {} /*",
	}
	for name, branch := range cases {
		t.Run(name, func(t *testing.T) {
			data := &Type{Name: "U", KindName: "UKind", Fields: []*Field{{KindConst: "UKindString", Name: branch, TypeTag: "String"}}}
			file := jen.NewFile("p")
			stmt := jen.Null()
			addUnionKindConsts(stmt, data)
			file.Add(stmt)
			var buf bytes.Buffer
			require.NoError(t, file.Render(&buf))
			parsed, err := parser.ParseFile(token.NewFileSet(), "p.go", buf.Bytes(), 0)
			require.NoError(t, err, buf.String())
			require.Len(t, parsed.Decls, 1, buf.String())
			require.Len(t, parsed.Decls[0].(*ast.GenDecl).Specs, 1, buf.String())
		})
	}
}
