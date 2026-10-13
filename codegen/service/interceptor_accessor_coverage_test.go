package service

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service/testdata"
	"github.com/CaliLuke/loom/expr"
)

func TestInterceptorAccessorCoverage(t *testing.T) {
	cases := []struct {
		name           string
		server, client []string
	}{
		{"server-only", []string{"a", "b"}, nil},
		{"client-only", nil, []string{"b", "c"}},
		{"identical", []string{"a", "b"}, []string{"a", "b"}},
		{"overlap", []string{"a", "b"}, []string{"b", "c"}},
		{"disjoint-single", []string{"a"}, []string{"c"}},
		{"disjoint-multiple", []string{"a", "b"}, []string{"c"}},
		{"client-first", []string{"c"}, []string{"a"}},
	}
	for _, streaming := range []bool{false, true} {
		kind := "unary"
		if streaming {
			kind = "streaming"
		}
		for _, c := range cases {
			t.Run(kind+"/"+c.name, func(t *testing.T) {
				root := runDSL(t, testdata.InterceptorAccessorCoverageDSL(c.server, c.client, streaming))
				services := NewServicesData(root)
				data := services.Get("coverage")
				// Sharing declarations must never broaden actual invocation membership.
				for _, method := range data.Methods {
					require.Equal(t, slices.Contains(c.server, method.Name), len(method.ServerInterceptors) != 0)
					require.Equal(t, slices.Contains(c.client, method.Name), len(method.ClientInterceptors) != 0)
				}
				for index, side := range [][]*InterceptorData{data.ServerInterceptors, data.ClientInterceptors} {
					want := c.server
					if index == 1 {
						want = c.client
					}
					if len(want) == 0 {
						require.Empty(t, side)
						continue
					}
					require.Len(t, side, 1)
					require.Len(t, side[0].Methods, len(want))
					for _, method := range side[0].Methods {
						require.Contains(t, want, strings.ToLower(method.MethodName))
					}
				}
				for _, side := range [][]*InterceptorData{data.interceptorDeclarations.server, data.interceptorDeclarations.client} {
					for _, declaration := range side {
						var names []string
						for _, method := range declaration.Methods {
							names = append(names, method.MethodName)
						}
						require.True(t, slices.IsSorted(names), "declarations follow canonical service method order")
					}
				}
				declared := make(map[string]int)
				for _, file := range InterceptorsFiles("example.com/coverage/gen", root.Services[0], services) {
					var source bytes.Buffer
					for _, section := range file.AllSections() {
						require.NoError(t, section.Write(&source))
					}
					parsed, err := parser.ParseFile(token.NewFileSet(), file.Path, source.Bytes(), 0)
					require.NoError(t, err)
					ast.Inspect(parsed, func(node ast.Node) bool {
						if spec, ok := node.(*ast.TypeSpec); ok {
							declared[spec.Name.Name]++
						}
						return true
					})
				}
				require.Equal(t, 1, declared["AuditInfo"])
				wanted := make(map[string]bool)
				for _, side := range [][]*InterceptorData{data.ServerInterceptors, data.ClientInterceptors} {
					for _, interceptor := range side {
						for _, method := range interceptor.Methods {
							for _, name := range []string{method.PayloadAccess, method.ResultAccess, method.StreamingPayloadAccess, method.StreamingResultAccess} {
								if name != "" {
									wanted[name] = true
								}
							}
						}
					}
				}
				for name := range wanted {
					if declared[name] != 1 {
						t.Errorf("accessor %s: got %d declarations, want one for the server/client union", name, declared[name])
					}
				}
			})
		}
	}
}

func TestInterceptorAccessorAliases(t *testing.T) {
	const genpkg = "example.com/coverage/gen"
	root := runDSL(t, testdata.InterceptorAccessorAliasDSL)
	root.Services[0].Methods[2].Payload.Find("value").Meta = expr.MetaExpr{"struct:field:name": {"StoredValue"}}
	services := NewServicesData(root)
	data := services.Get("coverage")
	require.NoError(t, SetUserTypeImports(genpkg, data))
	base := data.interceptorDeclarations.server[0]
	require.Equal(t, "*context.Value", base.ReadPayload[0].TypeRef)
	imports := append([]*codegen.ImportSpec{codegen.SimpleImport("context")}, data.UserTypeImports...)
	renamed, _ := services.fileData("coverage", imports)
	shared := renamed.interceptorDeclarations.server[0]
	require.NotSame(t, base, shared)
	require.Equal(t, "*context2.Value", shared.ReadPayload[0].TypeRef)
	require.Equal(t, "*context2.AInput", shared.Methods[0].PayloadRef)
	require.Equal(t, "*context2.CInput", shared.Methods[1].PayloadRef)
	require.Equal(t, "*context2.Value", shared.Methods[1].attributes[interceptorReadPayload][0].TypeRef)
	require.Equal(t, "StoredValue", shared.Methods[1].attributes[interceptorReadPayload][0].selector)
	require.Equal(t, "*context.Value", base.ReadPayload[0].TypeRef, "reanalysis must not mutate the base declaration")
	files := InterceptorsFiles(genpkg, root.Services[0], services)
	var rendered string
	for _, file := range files {
		rendered += codegen.SectionsCode(t, file.AllSections()[1:])
	}
	require.Contains(t, rendered, "Value() *context2.Value")
	require.Contains(t, rendered, "payload *context2.CInput")
	require.Contains(t, rendered, "return p.payload.StoredValue")
}
