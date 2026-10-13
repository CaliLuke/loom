package service

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen/service/testdata"
)

func TestInterceptorContractServerOperations(t *testing.T) {
	for _, transport := range []string{"grpc", "rpc-sse", "rpc-bidi", "rpc-request-reply"} {
		t.Run(transport, func(t *testing.T) {
			root := runDSL(t, testdata.InterceptorContractStreamDSL(transport))
			services := NewServicesData(root)
			data := services.Get("contracts")
			interfaceName := data.Methods[0].ServerStream.Interface
			wrapperName := data.ServerInterceptors[0].Methods[0].ServerStream.WrapperName
			files := Files("example.com/interceptorcontracts/gen", root.Services[0], services, make(map[string][]string))
			expected := make(map[string]string)
			actual := make(map[string]string)
			for _, file := range files {
				var source bytes.Buffer
				for _, section := range file.AllSections() {
					require.NoError(t, section.Write(&source))
				}
				fset := token.NewFileSet()
				parsed, err := parser.ParseFile(fset, file.Path, source.Bytes(), 0)
				require.NoError(t, err, "accepted design must render syntactically valid adapters")
				ast.Inspect(parsed, func(node ast.Node) bool {
					switch node := node.(type) {
					case *ast.TypeSpec:
						if node.Name.Name == interfaceName {
							for _, field := range node.Type.(*ast.InterfaceType).Methods.List {
								expected[field.Names[0].Name] = interceptorOperationSignature(t, field.Type.(*ast.FuncType))
							}
						}
					case *ast.FuncDecl:
						if node.Recv == nil {
							return true
						}
						var receiver bytes.Buffer
						require.NoError(t, format.Node(&receiver, fset, node.Recv.List[0].Type))
						if receiver.String() == "*"+wrapperName {
							require.NotContains(t, actual, node.Name.Name, "adapter must declare each operation once")
							actual[node.Name.Name] = interceptorOperationSignature(t, node.Type)
						}
					}
					return true
				})
			}
			require.NotEmpty(t, expected, "canonical service interface must be present")
			if data.Methods[0].ServerStream.EndpointStruct == "" {
				require.Empty(t, actual, "per-message endpoints have no stream adapter")
				return
			}
			delete(actual, "Unwrap")
			require.Equal(t, expected, actual, "adapter must implement precisely the canonical stream operations")
		})
	}
}

func interceptorOperationSignature(t *testing.T, fn *ast.FuncType) string {
	t.Helper()
	var signature bytes.Buffer
	for _, list := range []*ast.FieldList{fn.Params, fn.Results} {
		signature.WriteByte('(')
		if list != nil {
			for _, field := range list.List {
				count := max(1, len(field.Names))
				for range count {
					require.NoError(t, format.Node(&signature, token.NewFileSet(), field.Type))
					signature.WriteByte(',')
				}
			}
		}
		signature.WriteByte(')')
	}
	return signature.String()
}
