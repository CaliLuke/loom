package service

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/dave/jennifer/jen"
	"github.com/stretchr/testify/require"
)

func TestViewedResultCommentCannotInjectCode(t *testing.T) {
	cases := map[string]struct {
		description string
		comment     string
	}{
		"plain": {
			description: "tiny view",
			comment:     "//   - \"tiny\": tiny view",
		},
		"multi line": {
			description: "first line\nsecond line",
			comment:     "//   - \"tiny\": first line\n//     second line",
		},
		"hostile": {
			description: "a\n*/ }\nfunc Injected() {}\ntype Z interface { /*",
			comment:     "//   - \"tiny\": a\n//     */ }\n//     func Injected() {}\n//     type Z interface { /*",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			method := &MethodData{ViewedResult: &ViewedResultTypeData{Views: []*ViewData{{Name: "tiny", Description: tc.description}}}}
			file := jen.NewFile("p")
			file.Type().Id("Service").InterfaceFunc(func(group *jen.Group) {
				addViewedResultComment(group, method)
				group.Id("Show").Params()
			})
			var buf bytes.Buffer
			require.NoError(t, file.Render(&buf))
			parsed, err := parser.ParseFile(token.NewFileSet(), "p.go", buf.Bytes(), parser.ParseComments)
			require.NoError(t, err, buf.String())
			require.Len(t, parsed.Decls, 1, buf.String())
			spec := parsed.Decls[0].(*ast.GenDecl).Specs[0].(*ast.TypeSpec)
			require.Equal(t, "Service", spec.Name.Name)
			methods := spec.Type.(*ast.InterfaceType).Methods.List
			require.Len(t, methods, 1, buf.String())
			require.Contains(t, strings.ReplaceAll(buf.String(), "\t", ""), tc.comment)
		})
	}
}
