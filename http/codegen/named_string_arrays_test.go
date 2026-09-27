package codegen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

const namedStringArraysHarness = `package client

import (
 "context"
 "net/http"
 "net/http/httptest"
 "reflect"
 "slices"
 "strings"
 "testing"

 namedarrays "example.com/namedarrays/gen/namedarrays"
 server "example.com/namedarrays/gen/http/namedarrays/server"
 loomhttp "github.com/CaliLuke/loom/http"
)

func arrayPayload[T any]() T {
 var value T
 v := reflect.ValueOf(&value).Elem()
 if v.Kind() == reflect.Pointer {
  v.Set(reflect.New(v.Type().Elem()))
  v = v.Elem().Field(0)
 }
 v.Set(reflect.MakeSlice(v.Type(), 2, 2))
 v.Index(0).SetString("first")
 v.Index(1).SetString("second")
 return value
}

func TestNamedArrays(t *testing.T) {
 c := &Client{scheme: "https", host: "example.com"}
 %s
}
`

func TestNamedStringArrays(t *testing.T) {
	root := RunHTTPDSL(t, testdata.NamedStringArraysDSL)
	data := CreateHTTPServices(root).Get("namedarrays")
	require.Len(t, data.Endpoints, 18)
	for _, endpoint := range data.Endpoints {
		t.Run(endpoint.Method.Name, func(t *testing.T) {
			init := endpoint.Payload.Request.PayloadInit
			if init == nil || init.ReturnIsStruct {
				return
			}
			require.NotEmpty(t, init.ServerCode)
			require.NotEmpty(t, init.ClientCode)
		})
	}
}

func TestNamedStringArraysGeneratedModule(t *testing.T) {
	for _, tc := range []struct {
		name   string
		design func()
	}{{"headers and queries", testdata.NamedStringArraysDSL}, {"paths", testdata.NamedStringArrayPathsDSL}} {
		t.Run(tc.name, func(t *testing.T) {
			testNamedStringArrayModule(t, tc.design)
		})
	}
}

func testNamedStringArrayModule(t *testing.T, design func()) {
	t.Helper()
	root := RunHTTPDSL(t, design)
	data := CreateHTTPServices(root).Get("namedarrays")
	var cases strings.Builder
	for _, endpoint := range data.Endpoints {
		name := endpoint.Method.Name
		encode := ""
		if endpoint.RequestEncoder != "" {
			encode = fmt.Sprintf("if err := %s(nil)(req, p); err != nil {\n t.Fatal(err)\n}\n", endpoint.RequestEncoder)
		}
		fmt.Fprintf(&cases, `t.Run(%q, func(t *testing.T) {
 p := arrayPayload[%s]()
 req, err := c.%s(context.Background(), p)
 if err != nil { t.Fatal(err) }
 %s
 switch {
 case strings.HasSuffix(%q, "header"):
  if got := req.Header.Values("values"); !slices.Equal(got, []string{"first", "second"}) { t.Errorf("headers = %%v", got) }
 case strings.HasSuffix(%q, "query"):
  if got := req.URL.Query()["values"]; !slices.Equal(got, []string{"first", "second"}) { t.Errorf("query = %%v", got) }
 default:
  if !strings.HasSuffix(req.URL.Path, "/first,second") { t.Errorf("path = %%q", req.URL.Path) }
 }
 mux := loomhttp.NewMuxer()
 called := false
 mux.Handle("GET", %q, func(w http.ResponseWriter, r *http.Request) {
  called = true
  got, err := server.%s(mux, nil)(r)
  if err != nil { t.Fatal(err) }
  if !reflect.DeepEqual(got, p) { t.Errorf("decoded = %%#v, want %%#v", got, p) }
 })
 mux.ServeHTTP(httptest.NewRecorder(), req)
 if !called { t.Error("decoder was not called") }
`, name, endpoint.Payload.Ref, endpoint.RequestInit.Name, encode, name, name, endpoint.Routes[0].Path, endpoint.RequestDecoder)
		if init := endpoint.Payload.Request.PayloadInit; init != nil {
			fmt.Fprintf(&cases, `got, err := Build%sPayload(%q)
 if err != nil { t.Fatal(err) }
 if !reflect.DeepEqual(got, p) { t.Errorf("CLI constructor = %%#v, want %%#v", got, p) }
`, endpoint.Method.VarName, `["first","second"]`)
		}
		cases.WriteString("})\n")
	}
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/namedarrays", root)
	harness := fmt.Sprintf(namedStringArraysHarness, cases.String())
	require.NoError(t, os.WriteFile(filepath.Join(dir, "gen", "http", "namedarrays", "client", "arrays_test.go"), []byte(harness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-count=1", "./...")
}

func TestNamedStringArrayTransportTypes(t *testing.T) {
	label := &expr.UserTypeExpr{TypeName: "Label", AttributeExpr: &expr.AttributeExpr{Type: expr.String}}
	chain := &expr.UserTypeExpr{TypeName: "Chain", AttributeExpr: &expr.AttributeExpr{Type: label}}
	list := &expr.UserTypeExpr{TypeName: "List", AttributeExpr: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: chain}}}}
	object := &expr.Object{
		{Name: "first", Attribute: &expr.AttributeExpr{Type: list}},
		{Name: "second", Attribute: &expr.AttributeExpr{Type: list}},
		{Name: "third", Attribute: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: chain}}}},
		{Name: "fourth", Attribute: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: chain}}}},
	}
	input := &expr.AttributeExpr{Type: object}
	output := makeHTTPType(input)
	for _, name := range []string{"first", "second", "third", "fourth"} {
		t.Run(name, func(t *testing.T) {
			array, ok := output.Find(name).Type.(*expr.Array)
			require.True(t, ok)
			require.Equal(t, expr.String, array.ElemType.Type)
			require.Same(t, chain, expr.AsArray(input.Find(name).Type).ElemType.Type)
		})
	}
	require.Same(t, label, chain.Attribute().Type)
	require.Same(t, chain, expr.AsArray(list).ElemType.Type)
}

func TestNamedStringAliasTransportDefaults(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		label := dsl.Type("DefaultLabel", dsl.String, func() {
			dsl.Default("inner")
			dsl.Example("inner example")
		})
		chain := dsl.Type("DefaultChain", label, func() {
			dsl.Default("fallback")
			dsl.Example("outer example")
		})
		dsl.Service("defaults", func() {
			dsl.Method("read", func() {
				dsl.Payload(func() {
					dsl.Attribute("value", chain)
				})
				dsl.HTTP(func() {
					dsl.GET("/")
					dsl.Header("value")
				})
			})
		})
	})
	header := CreateHTTPServices(root).Get("defaults").Endpoint("read").Payload.Request.Headers[0]
	require.Equal(t, "fallback", header.DefaultValue)
	require.Equal(t, "outer example", header.Example)
	require.Equal(t, expr.String, header.Type)
}

func TestNamedStringArrayPathTransforms(t *testing.T) {
	data := CreateHTTPServices(RunHTTPDSL(t, testdata.NamedStringArrayPathsDSL)).Get("namedarrays")
	require.Len(t, data.Endpoints, 9)
	for _, endpoint := range data.Endpoints {
		t.Run(endpoint.Method.Name, func(t *testing.T) {
			if strings.HasPrefix(endpoint.Method.Name, "plain") {
				return
			}
			require.Contains(t, endpoint.RequestInit.ClientCode, "make([]string")
			require.NotEqual(t, "val", endpoint.Routes[0].PathInit.ClientArgs[0].VarName)
		})
	}
}
