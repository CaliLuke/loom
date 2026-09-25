package service

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen/service/testdata"
)

// TestInterceptorWrapperNames generates the interceptor files of services
// whose interceptor design names do not start with an upper case letter and
// checks that every interceptor wrapper that the endpoint wrappers call is
// declared, and that every declared wrapper is called.
func TestInterceptorWrapperNames(t *testing.T) {
	cases := []struct {
		Name string
		DSL  func()
		// Wrappers are the names of the declared interceptor wrappers.
		Wrappers []string
	}{
		{
			Name:     "api-server-interceptor",
			DSL:      testdata.SingleAPIServerInterceptorDSL,
			Wrappers: []string{"wrapMethodLogging", "wrapMethod2Logging"},
		},
		{
			Name:     "client-interceptor",
			DSL:      testdata.SingleClientInterceptorDSL,
			Wrappers: []string{"wrapClientMethodTracing"},
		},
		{
			Name: "lower-case-and-snake-case-names",
			DSL:  testdata.LowerCaseInterceptorNamesDSL,
			Wrappers: []string{
				"wrapDirectAudit",
				"wrapDirectAuditLog",
				"wrapClientDirectAudit",
				"wrapClientDirectAuditLog",
			},
		},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := runDSL(t, c.DSL)
			services := NewServicesData(root)
			require.Len(t, root.Services, 1)

			declared := map[string]bool{}
			called := map[string]bool{}
			fset := token.NewFileSet()
			for _, f := range InterceptorsFiles("github.com/CaliLuke/loom/example", root.Services[0], services) {
				buf := new(bytes.Buffer)
				for _, s := range f.AllSections() {
					require.NoError(t, s.Write(buf))
				}
				file, err := parser.ParseFile(fset, f.Path, buf.Bytes(), 0)
				require.NoError(t, err, buf.String())
				ast.Inspect(file, func(n ast.Node) bool {
					switch n := n.(type) {
					case *ast.FuncDecl:
						if n.Recv == nil && strings.HasPrefix(n.Name.Name, "wrap") {
							declared[n.Name.Name] = true
						}
					case *ast.CallExpr:
						if id, ok := n.Fun.(*ast.Ident); ok && strings.HasPrefix(id.Name, "wrap") {
							called[id.Name] = true
						}
					}
					return true
				})
			}
			assert.ElementsMatch(t, c.Wrappers, slices.Collect(maps.Keys(declared)))
			assert.ElementsMatch(t, c.Wrappers, slices.Collect(maps.Keys(called)))
		})
	}
}
