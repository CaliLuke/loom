package service

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dsl "github.com/CaliLuke/loom/dsl"
)

// TestViewedResultInterceptorWrappers checks that the server wrapper of an
// interceptor that accesses the result of a method whose endpoint returns a
// viewed result converts the viewed result to the result type for the
// interceptor and projects the result that the interceptor returns with the
// view of the endpoint result, or with the view of the design, and that it
// returns a fault error for a result of another type or a nil result. The
// wrappers of an interceptor that does not access the result, of a method
// whose result is a user type and of a client interceptor call the
// interceptor with the endpoint.
func TestViewedResultInterceptorWrappers(t *testing.T) {
	root := runDSL(t, viewedResultInterceptorDSL)
	services := NewServicesData(root)
	bodies := make(map[string]string)
	fset := token.NewFileSet()
	for _, f := range InterceptorsFiles("example.com/viewed/gen", root.Services[0], services) {
		buf := new(bytes.Buffer)
		for _, s := range f.AllSections() {
			require.NoError(t, s.Write(buf))
		}
		file, err := parser.ParseFile(fset, f.Path, buf.Bytes(), 0)
		require.NoError(t, err, buf.String())
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || !strings.HasPrefix(fn.Name.Name, "wrap") {
				continue
			}
			var body bytes.Buffer
			require.NoError(t, printer.Fprint(&body, fset, fn.Body))
			bodies[fn.Name.Name] = body.String()
		}
	}
	cases := []struct {
		wrapper string
		want    []string
	}{
		{"wrapVgetStamp", []string{
			`view := ""`,
			"vres, ok := res.(*svcviews.VRT)",
			"if !ok || vres == nil {",
			`return nil, loom.Fault("invalid value expected *svcviews.VRT, got %v", res)`,
			"view = vres.View",
			"return NewVRT(vres)",
			"result, ok := res.(*VRT)",
			"if !ok || result == nil {",
			`return nil, loom.Fault("invalid value expected *VRT, got %v", res)`,
			"return NewViewedVRT(result, view)",
		}},
		{"wrapTinyStamp", []string{
			`view := "tiny"`,
			"result, ok := res.(*VRT)",
			"return NewViewedVRT(result, view)",
		}},
		{"wrapVgetPass", nil},
		{"wrapShowStamp", nil},
		{"wrapClientVgetStamp", nil},
	}
	for _, c := range cases {
		body, ok := bodies[c.wrapper]
		if !assert.True(t, ok, "no %s wrapper", c.wrapper) {
			continue
		}
		if len(c.want) == 0 {
			assert.Contains(t, body, "return i.", c.wrapper)
			assert.NotContains(t, body, "NewViewed", c.wrapper)
			continue
		}
		for _, want := range c.want {
			assert.Contains(t, body, want, c.wrapper)
		}
	}
}

func viewedResultInterceptorDSL() {
	dsl.Interceptor("pass")
	dsl.Interceptor("stamp", func() {
		dsl.ReadResult(func() {
			dsl.Attribute("id")
		})
		dsl.WriteResult(func() {
			dsl.Attribute("id")
		})
	})
	vrt := dsl.ResultType("application/vnd.vrt", "VRT", func() {
		dsl.Attribute("id", dsl.String)
		dsl.Attribute("name", dsl.String)
		dsl.View("default", func() {
			dsl.Attribute("id")
			dsl.Attribute("name")
		})
		dsl.View("tiny", func() {
			dsl.Attribute("id")
		})
	})
	ut := dsl.Type("UT", func() {
		dsl.Attribute("id", dsl.String)
	})
	dsl.Service("svc", func() {
		dsl.ServerInterceptor("pass")
		dsl.ServerInterceptor("stamp")
		dsl.ClientInterceptor("stamp")
		dsl.Method("vget", func() {
			dsl.Result(vrt)
			dsl.HTTP(func() {
				dsl.GET("/vget")
			})
		})
		dsl.Method("tiny", func() {
			dsl.Result(vrt, func() {
				dsl.View("tiny")
			})
			dsl.HTTP(func() {
				dsl.GET("/tiny")
			})
		})
		dsl.Method("show", func() {
			dsl.Result(ut)
			dsl.HTTP(func() {
				dsl.GET("/show")
			})
		})
	})
}
