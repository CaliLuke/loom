package codegen

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPathSectionCommentCannotInjectCode(t *testing.T) {
	long := "ShowPath returns the URL path to the ServiceWithAVeryLongName service ShowSomethingLong HTTP endpoint. "
	cases := map[string]struct {
		description string
		comment     string
	}{
		"long line not wrapped": {
			description: long,
			comment:     "// " + strings.TrimSpace(long) + "\nfunc ShowPath() string {",
		},
		"hostile newline": {
			description: "ShowPath returns\n*/ func Injected() {} /*",
			comment:     "// ShowPath returns\n// */ func Injected() {} /*\nfunc ShowPath() string {",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			data := &EndpointData{Routes: []*RouteData{{PathInit: &InitData{
				Name:          "ShowPath",
				Description:   tc.description,
				ReturnTypeRef: "string",
				ServerCode:    "return \"/show\"",
			}}}}
			var buf bytes.Buffer
			buf.WriteString("package p\n\n")
			require.NoError(t, pathSection(data).Write(&buf))
			src := buf.String()
			file, err := parser.ParseFile(token.NewFileSet(), "p.go", src, 0)
			require.NoError(t, err, src)
			require.Len(t, file.Decls, 1, src)
			require.Equal(t, "ShowPath", file.Decls[0].(*ast.FuncDecl).Name.Name)
			require.Contains(t, src, tc.comment)
		})
	}
}
