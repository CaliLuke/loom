package codegen

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestMultipartEncoderBranchTermination(t *testing.T) {
	for _, tc := range []struct {
		name    string
		design  func()
		service string
	}{
		{name: "nested request and response", design: testdata.UntaggedMultipartDSL, service: "multipartchoices"},
		{name: "nested content application", design: testdata.EncodingSourceDSL, service: "encoding_source"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			services := CreateHTTPServices(RunHTTPDSL(t, tc.design))
			for _, endpoint := range services.Get(tc.service).Endpoints {
				request := endpoint.Payload.Request
				if !request.MultipartGenerated {
					continue
				}
				t.Run(endpoint.Method.Name, func(t *testing.T) {
					code := "package generated\nfunc encode() (any, error) {\n" + renderMultipartEncode(request.ClientBody, request.multipart) + "}\n"
					file, err := parser.ParseFile(token.NewFileSet(), "encoder.go", code, 0)
					require.NoError(t, err)
					ast.Inspect(file, func(node ast.Node) bool {
						block, ok := node.(*ast.BlockStmt)
						if !ok {
							return true
						}
						for index, statement := range block.List {
							if _, terminal := statement.(*ast.ReturnStmt); terminal {
								require.Equal(t, len(block.List)-1, index, "a returning branch must not emit following statements")
							}
						}
						return true
					})
				})
			}
		})
	}
}
