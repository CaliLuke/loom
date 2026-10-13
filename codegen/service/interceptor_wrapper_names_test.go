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
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// TestInterceptorWrapperNames checks case normalization, concatenated names and
// client/server prefix collisions. Wrapper and accessor declarations must be
// unique, every called wrapper must be declared, every declared wrapper must be
// called, and each wrapper's GoDoc must name the function it documents.
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
		{
			Name: "concatenated-method-interceptor-pairs",
			DSL:  interceptorWrapperCollisionDSL("pairs"),
			Wrappers: []string{
				"wrapListAllAuth",
				"wrapListAllAuth2",
			},
		},
		{
			Name: "client-server-prefix",
			DSL:  interceptorWrapperCollisionDSL("sides"),
			Wrappers: []string{
				"wrapClientListTrace",
				"wrapClientListTrace2",
			},
		},
		{
			Name: "concatenated-accessor-pairs",
			DSL:  interceptorWrapperCollisionDSL("accessors"),
			Wrappers: []string{
				"wrapLogReadAudit",
				"wrapReadAuditLog",
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
			types := map[string]bool{}
			fset := token.NewFileSet()
			for _, f := range InterceptorsFiles("github.com/CaliLuke/loom/example", root.Services[0], services) {
				buf := new(bytes.Buffer)
				for _, s := range f.AllSections() {
					require.NoError(t, s.Write(buf))
				}
				file, err := parser.ParseFile(fset, f.Path, buf.Bytes(), parser.ParseComments)
				require.NoError(t, err, buf.String())
				ast.Inspect(file, func(n ast.Node) bool {
					switch n := n.(type) {
					case *ast.TypeSpec:
						assert.NotContains(t, types, n.Name.Name, "accessor declarations must be unique")
						types[n.Name.Name] = true
					case *ast.FuncDecl:
						if n.Recv == nil && strings.HasPrefix(n.Name.Name, "wrap") {
							assert.NotContains(t, declared, n.Name.Name, "wrapper declarations must be unique")
							declared[n.Name.Name] = true
							assert.True(t, strings.HasPrefix(n.Doc.Text(), n.Name.Name+" applies "),
								"GoDoc for %s must name its function: %q", n.Name.Name, n.Doc.Text())
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
			if c.Name == "concatenated-accessor-pairs" {
				assert.Contains(t, types, "auditLogReadPayload")
				assert.Contains(t, types, "auditLogReadPayload2")
			}
		})
	}
}

func interceptorWrapperCollisionDSL(kind string) func() {
	return func() {
		type binding struct {
			method, interceptor string
			client              bool
		}
		var bindings []binding
		switch kind {
		case "pairs":
			bindings = []binding{{"List", "AllAuth", false}, {"ListAll", "Auth", false}}
		case "sides":
			bindings = []binding{{"List", "Trace", true}, {"ClientList", "Trace", false}}
		case "accessors":
			bindings = []binding{{"LogRead", "Audit", false}, {"Read", "AuditLog", false}}
		}
		interceptors := make(map[string]*expr.InterceptorExpr)
		for _, b := range bindings {
			if interceptors[b.interceptor] != nil {
				continue
			}
			interceptors[b.interceptor] = dsl.Interceptor(b.interceptor, func() {
				if kind == "accessors" {
					dsl.ReadPayload(func() {
						dsl.Attribute("value")
					})
				}
			})
		}
		dsl.Service("collisions_"+kind, func() {
			for _, b := range bindings {
				dsl.Method(b.method, func() {
					if b.client {
						dsl.ClientInterceptor(interceptors[b.interceptor])
					} else {
						dsl.ServerInterceptor(interceptors[b.interceptor])
					}
					if kind == "accessors" {
						dsl.Payload(func() {
							dsl.Attribute("value", dsl.String)
						})
					}
					dsl.HTTP(func() {
						dsl.POST("/" + kind + "/" + b.method)
					})
				})
			}
		})
	}
}
