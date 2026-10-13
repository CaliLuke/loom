package service

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

// TestInterceptorContractSignatureIdentity records actual emitted getter and
// storage types; structural DSL equality cannot decide shared Go signatures.
func TestInterceptorContractSignatureIdentity(t *testing.T) {
	for _, tc := range []struct {
		name    string
		getter  string
		storage string
	}{
		{"optional", "string", "*string"},
		{"required", "string", "string"},
		{"nominal-first", "*FirstValue", "*FirstValue"},
		{"nominal-second", "*SecondValue", "*SecondValue"},
		{"custom", "time.Duration", "*time.Duration"},
		{"nullable", "loom.Nullable[string]", "loom.Nullable[string]"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := runDSL(t, func() {
				var typ any = dsl.String
				if tc.name == "custom" {
					typ = dsl.Int64
				}
				if tc.name == "nominal-first" || tc.name == "nominal-second" {
					name := "FirstValue"
					if tc.name == "nominal-second" {
						name = "SecondValue"
					}
					typ = dsl.Type(name, func() {
						dsl.Attribute("text", dsl.String)
					})
				}
				audit := dsl.Interceptor("Audit", func() {
					dsl.ReadPayload(func() {
						dsl.Attribute("value")
					})
				})
				dsl.Service("contracts", func() {
					dsl.ServerInterceptor(audit)
					dsl.Method("show", func() {
						dsl.Payload(func() {
							dsl.Attribute("value", typ, func() {
								if tc.name == "custom" {
									dsl.Meta("struct:field:type", "time.Duration", "time")
								}
								if tc.name == "nullable" {
									dsl.Nullable()
								}
							})
							if tc.name == "required" {
								dsl.Required("value")
							}
						})
					})
				})
			})
			services := NewServicesData(root)
			var getter, storage string
			for _, file := range Files("example.com/interceptorcontracts/gen", root.Services[0], services, make(map[string][]string)) {
				var source bytes.Buffer
				for _, section := range file.AllSections() {
					require.NoError(t, section.Write(&source))
				}
				fset := token.NewFileSet()
				parsed, err := parser.ParseFile(fset, file.Path, source.Bytes(), 0)
				require.NoError(t, err)
				ast.Inspect(parsed, func(node ast.Node) bool {
					typ, ok := node.(*ast.TypeSpec)
					if !ok {
						return true
					}
					var value ast.Expr
					if typ.Name.Name == "AuditPayload" {
						value = typ.Type.(*ast.InterfaceType).Methods.List[0].Type.(*ast.FuncType).Results.List[0].Type
					} else if typ.Name.Name == "ShowPayload" {
						value = typ.Type.(*ast.StructType).Fields.List[0].Type
					}
					if value != nil {
						var rendered bytes.Buffer
						require.NoError(t, format.Node(&rendered, fset, value))
						if typ.Name.Name == "AuditPayload" {
							getter = rendered.String()
						} else {
							storage = rendered.String()
						}
					}
					return true
				})
			}
			t.Logf("emitted getter %s; field %s", getter, storage)
			require.Equal(t, tc.getter, getter)
			require.Equal(t, tc.storage, storage)
		})
	}
}
