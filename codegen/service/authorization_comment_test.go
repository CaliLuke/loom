package service

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestAuthorizerInterfaceCommentsCannotInjectCode(t *testing.T) {
	cases := map[string]struct {
		name    string
		comment string
	}{
		"plain": {
			name:    "document.edit",
			comment: "// AuthorizeDocumentEdit evaluates document.edit using current application access facts.\nAuthorizeDocumentEdit(",
		},
		"hostile newline": {
			name:    "x\n}\nfunc Injected() {}\ntype Z interface {",
			comment: "// AuthorizeDocumentEdit evaluates x\n// }\n// func Injected() {}\n// type Z interface { using current application access facts.\nAuthorizeDocumentEdit(",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var b strings.Builder
			writeAuthorizerInterface(&b, &authorizationServiceData{
				interfaceName: "Access",
				requirements: []*authorizationRequirementData{{
					expr:       &expr.AuthorizationExpr{Name: tc.name},
					methodName: "AuthorizeDocumentEdit",
				}},
			})
			src := "package p\n\n" + b.String()
			file, err := parser.ParseFile(token.NewFileSet(), "p.go", src, 0)
			require.NoError(t, err, src)
			require.Len(t, file.Decls, 1, src)
			spec := file.Decls[0].(*ast.GenDecl).Specs[0].(*ast.TypeSpec)
			require.Equal(t, "Access", spec.Name.Name)
			require.Len(t, spec.Type.(*ast.InterfaceType).Methods.List, 1, src)
			require.Contains(t, src, tc.comment)
		})
	}
}
