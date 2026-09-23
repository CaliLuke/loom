package codegen

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	cg "github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	. "github.com/CaliLuke/loom/dsl"
)

// transportVarNameCollisionDSL declares HTTP path params, query params,
// headers, and cookies whose attribute names Goify to the same Go identifier,
// to an identifier that the generated transport functions use themselves, or
// to a local derived from another attribute variable, such as fooRaw for the
// raw form of foo. It covers the request, a success response, and an error
// response. The struct field names are distinct so that only the transport
// variable names collide. A second method uses some of the same names alone.
var transportVarNameCollisionDSL = func() {
	var Fault = Type("Fault", func() {
		ErrorName("name", String)
		Attribute("res_id", String)
		Attribute("resId", String, func() {
			Meta("struct:field:name", "ResIDSecond")
		})
		Attribute("err", String)
		Attribute("w", String)
		Required("name")
	})
	Service("VarCollision", func() {
		Method("Colliding", func() {
			Payload(func() {
				Attribute("fooBar", String)
				Attribute("foo_bar", String, func() {
					Meta("struct:field:name", "FooBarQuery")
				})
				Attribute("foo__bar", String, func() {
					Meta("struct:field:name", "FooBarHeader")
				})
				Attribute("Foo_Bar", String, func() {
					Meta("struct:field:name", "FooBarCookie")
				})
				Attribute("count", Int)
				Attribute("Count", Int, func() {
					Meta("struct:field:name", "CountHeader")
				})
				Attribute("p", String)
				Attribute("err", String)
				Attribute("payload", String)
				Attribute("params", String)
				Attribute("raw", String)
				Attribute("v", Int)
				Attribute("req", String)
				Attribute("resp", String)
				Attribute("c", String)
				Attribute("qp", String)
				Attribute("data", String)
				Attribute("items", ArrayOf(Int))
				Attribute("itemsSlice", String)
				Attribute("foo", Int)
				Attribute("fooRaw", String)
				Attribute("ids", ArrayOf(Int))
				Attribute("idsRaw", ArrayOf(String))
				Attribute("tokenRaw", String)
				Attribute("token", String)
				Attribute("val", MapOf(String, Int))
				Attribute("Val", Int, func() {
					Meta("struct:field:name", "ValHeader")
				})
				Attribute("utf8", String, func() {
					MinLength(2)
				})
				Attribute("strconv", Int)
				Attribute("url", String)
				Attribute("http", String)
				Required("fooBar", "p", "items")
			})
			Result(func() {
				Attribute("res_id", String)
				Attribute("resId", String, func() {
					Meta("struct:field:name", "ResIDCookie")
				})
				Attribute("Res_Id", String, func() {
					Meta("struct:field:name", "ResIDCookieSecond")
				})
				Attribute("err", String)
				Attribute("resp", String)
				Attribute("res", String)
				Attribute("ctx", String)
				Attribute("v", Int)
				Attribute("data", String)
				Attribute("tag", Int)
				Attribute("tags", String)
				Attribute("tagsSlice", String)
				Attribute("nums", ArrayOf(Int))
				Attribute("numss", String)
				Attribute("count", Int)
				Attribute("countraw", String)
				Attribute("countSlice", String)
				Attribute("levelRaw", String)
				Attribute("level", Int)
				Attribute("utf8", String, func() {
					MinLength(2)
				})
				Attribute("fmt", Int)
				Attribute("strings", Int)
			})
			Error("fault", Fault)
			HTTP(func() {
				POST("/items/{fooBar}/{p}/{items}/{itemsSlice}")
				Param("foo_bar")
				Param("count")
				Param("err")
				Param("params")
				Param("raw")
				Param("qp")
				Param("ids")
				Param("idsRaw")
				Param("val")
				Param("strconv")
				Param("url")
				Header("Val:X-Val")
				Header("utf8:X-Utf8")
				Cookie("http:http_cookie")
				Header("foo:X-Foo")
				Header("fooRaw:X-Foo-Raw")
				Cookie("tokenRaw:token_raw")
				Cookie("token:token")
				Header("foo__bar:X-Foo-Bar")
				Header("Count:X-Count")
				Header("payload:X-Payload")
				Header("v:X-V")
				Header("req:X-Req")
				Cookie("Foo_Bar:foo_bar_cookie")
				Cookie("resp:resp_cookie")
				Cookie("c:c_cookie")
				Body("data")
				Response(StatusOK, func() {
					Header("res_id:X-Res-Id")
					Header("err:X-Err")
					Header("resp:X-Resp")
					Header("v:X-V")
					Header("tag:X-Tag")
					Header("tags:X-Tags")
					Header("tagsSlice:X-Tags-Slice")
					Header("nums:X-Nums")
					Header("numss:X-Numss")
					Header("levelRaw:X-Level-Raw")
					Header("level:X-Level")
					Header("utf8:X-Utf8")
					Header("fmt:X-Fmt")
					Cookie("strings:strings_cookie")
					Cookie("count:count_cookie")
					Cookie("countraw:countraw_cookie")
					Cookie("countSlice:count_slice_cookie")
					Cookie("resId:res_id_cookie")
					Cookie("Res_Id:res_id_cookie2")
					Cookie("res:res_cookie")
					Cookie("ctx:ctx_cookie")
					Body("data")
				})
				Response("fault", StatusBadRequest, func() {
					Header("res_id:X-Fault-Res-Id")
					Header("resId:X-Fault-ResId")
					Cookie("err:fault_err")
					Cookie("w:fault_w")
				})
			})
		})
		Method("Isolated", func() {
			Payload(func() {
				Attribute("foo_bar", String)
			})
			Result(func() {
				Attribute("res_id", String)
			})
			HTTP(func() {
				GET("/isolated")
				Header("foo_bar:X-Foo-Bar")
				Response(StatusOK, func() {
					Header("res_id:X-Res-Id")
				})
			})
		})
	})
	var Rt = ResultType("application/vnd.rt", func() {
		Attributes(func() {
			Attribute("value", String)
			Attribute("calcviews", String)
			Attribute("calc", String)
		})
		View("default", func() {
			Attribute("value")
			Attribute("calcviews")
			Attribute("calc")
		})
	})
	Service("Calc", func() {
		Method("Compute", func() {
			Payload(func() {
				Attribute("calc", String)
				Attribute("calcviews", String)
			})
			Result(Rt)
			HTTP(func() {
				GET("/compute")
				Param("calc")
				Header("calcviews:X-Views")
				Response(StatusOK, func() {
					Header("calcviews:X-Views")
					Header("calc:X-Calc")
				})
			})
		})
	})
}

func TestTransportVarNamesAreUniquePerFunction(t *testing.T) {
	root := RunHTTPDSL(t, transportVarNameCollisionDSL)
	services := CreateHTTPServices(root)
	files := append(ServerFiles("", services), ClientFiles("", services)...)
	files = append(files, PathFiles(services)...)
	files = append(files, ServerTypeFiles("", services)...)
	files = append(files, ClientTypeFiles("", services)...)
	files = append(files, ClientCLIFiles("", services)...)

	funcs := renderTransportFuncs(t, files)

	cases := []struct {
		Name     string
		Func     string
		Contains []string
		Excludes []string
	}{
		{
			Name: "server request decoder",
			Func: "DecodeCollidingRequest",
			Contains: []string{
				`fooBar = params["fooBar"]`,
				`qp2["foo_bar"]`,
				`r.Header.Get("X-Foo-Bar")`,
				`r.Cookie("foo_bar_cookie")`,
				`fooRaw := r.Header.Get("X-Foo")`,
				`fooRaw2Raw := r.Header.Get("X-Foo-Raw")`,
				`for i, rv := range idsRaw {`,
				`idsRaw2 = qp2["idsRaw"]`,
				"val2[keya] = vala",
				"tokenRaw = &tokenRawRaw",
				"token = &tokenRaw2",
				"utf8.RuneCountInString(*utf82)",
				"val3 = &pv",
				"NewCollidingPayload(body, fooBar, p, items, itemsSlice, fooBar2, count, err3, params2, raw2, qp, ids, idsRaw2, val2, strconv2, url_, val3, utf82, foo, fooRaw2, fooBar3, count2, payload2, v2, req2, http_, tokenRaw, token, fooBar4, resp, c2)",
			},
		},
		{
			Name:     "server payload initializer",
			Func:     "NewCollidingPayload",
			Contains: []string{"v := body", "res.FooBarQuery = fooBar2", "res.FooBarHeader = fooBar3", "res.FooBarCookie = fooBar4", "res.V = v2"},
		},
		{
			Name:     "client request builder",
			Func:     "BuildCollidingRequest",
			Contains: []string{"CollidingVarCollisionPath(fooBar, p2, items, itemsSlice2)"},
		},
		{
			Name:     "path builder",
			Func:     "CollidingVarCollisionPath",
			Contains: []string{"fooBar string, p2 string, items []int, itemsSlice2 string", `strings.Join(itemsSlice, ",")`},
		},
		{
			Name: "client response decoder",
			Func: "DecodeCollidingResponse",
			Contains: []string{
				"levelRaw = &levelRawRaw",
				"levelRaw2 := resp.Header.Get(\"X-Level\")",
				"NewCollidingResultOK(body, resID, err3, resp2, v2, tag, tags2, tagsSlice2, nums, numss2, levelRaw, level, utf82, fmt_, strings2, count, countraw2, countSlice2, resID2, resID3, res2, ctx2)",
				"NewCollidingFault(&body, resID, resID2, err3, w2)",
			},
		},
		{
			Name: "server response encoder",
			Func: "EncodeCollidingResponse",
			Contains: []string{
				"tags := strconv.Itoa(*val)",
				"numssSlice := make([]string, len(val))",
				"countraw := res.Count",
				"count := strconv.Itoa(*countraw)",
			},
		},
		{
			Name:     "CLI payload builder",
			Func:     "BuildCollidingPayload",
			Contains: []string{"var val3 *int", "val3 = &val"},
		},
		{
			Name:     "viewed response avoids service and views aliases",
			Func:     "DecodeComputeResponse",
			Contains: []string{"calcviews2Raw := resp.Header.Get(\"X-Views\")", "calc2Raw := resp.Header.Get(\"X-Calc\")"},
		},
		{
			Name:     "request avoids service and views aliases",
			Func:     "DecodeComputeRequest",
			Contains: []string{`calc2 = &raw`, `calcviews2Raw := r.Header.Get("X-Views")`},
		},
		{
			Name:     "isolated endpoint keeps base names",
			Func:     "DecodeIsolatedRequest",
			Contains: []string{"fooBar"},
			Excludes: []string{"fooBar2"},
		},
		{
			Name:     "isolated response keeps base names",
			Func:     "DecodeIsolatedResponse",
			Contains: []string{"NewIsolatedResultOK(resID)"},
		},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			code, ok := funcs[c.Func]
			require.True(t, ok, "function %s not generated", c.Func)
			for _, s := range c.Contains {
				require.Contains(t, code, s)
			}
			for _, s := range c.Excludes {
				require.NotContains(t, code, s)
			}
		})
	}
}

func TestTransportVarNameCollisionsCompile(t *testing.T) {
	root := RunHTTPDSL(t, transportVarNameCollisionDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/varcollision", root)
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
}

func TestTransportFunctionImportNamesCoverFileImports(t *testing.T) {
	const genpkg = "example.com/gen"
	data := &ServiceData{Service: &service.Data{PathName: "svc", PkgName: "svc", ViewsPkg: "svcviews"}}
	lists := map[string][]*cg.ImportSpec{
		"server encode_decode.go": serverEncodeDecodeImports(genpkg, "svc", data),
		"server types.go":         serverTypeImports(genpkg, "svc", data),
		"client encode_decode.go": clientEncodeDecodeImports(genpkg, "svc", data),
		"client types.go":         clientTypeImports(genpkg, "svc", data),
		"client cli.go":           clientCLIImports(genpkg, data),
		"paths.go":                pathImports(),
	}
	// serverEncodeDecodeImports adds these only for services that use them.
	lists["server conditional"] = []*cg.ImportSpec{{Path: "net/url"}, {Path: "mime/multipart"}}
	for file, specs := range lists {
		for _, spec := range specs {
			if strings.HasPrefix(spec.Path, genpkg) {
				continue
			}
			name := importName(spec)
			if !slices.Contains(transportFunctionImportNames, name) {
				t.Errorf("%s imports %q as %q, which transportFunctionImportNames does not reserve", file, spec.Path, name)
			}
		}
	}
}

// renderTransportFuncs renders files, reports every function that declares a
// variable twice in the same scope, and returns the source of each rendered
// function by name.
func renderTransportFuncs(t *testing.T, files []*cg.File) map[string]string {
	t.Helper()
	funcs := map[string]string{}
	for _, file := range files {
		renderedPath, err := file.Render(t.TempDir())
		require.NoError(t, err)
		src, err := os.ReadFile(renderedPath)
		require.NoError(t, err)
		fset := token.NewFileSet()
		parsed, err := parser.ParseFile(fset, renderedPath, src, 0)
		require.NoError(t, err)
		for _, decl := range parsed.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			funcs[fn.Name.Name] = string(src[fset.Position(fn.Pos()).Offset:fset.Position(fn.End()).Offset])
			for _, dup := range duplicateScopeVars(fn) {
				t.Errorf("%s: %s declares %q more than once in the same scope", file.Path, fn.Name.Name, dup)
			}
		}
	}
	return funcs
}

// duplicateScopeVars returns the identifiers that fn declares more than once
// in the same scope. Parameters share the scope of the function body, and
// function literals start their own parameter scope.
func duplicateScopeVars(fn *ast.FuncDecl) []string {
	var dups []string
	declareFields := func(seen map[string]bool, lists ...*ast.FieldList) {
		for _, list := range lists {
			if list == nil {
				continue
			}
			for _, field := range list.List {
				for _, id := range field.Names {
					if id.Name != "_" && seen[id.Name] {
						dups = append(dups, id.Name)
					}
					seen[id.Name] = true
				}
			}
		}
	}
	var visitList func(stmts []ast.Stmt, seen map[string]bool)
	visitNested := func(n ast.Node) {
		ast.Inspect(n, func(node ast.Node) bool {
			switch x := node.(type) {
			case *ast.FuncLit:
				seen := map[string]bool{}
				declareFields(seen, x.Type.Params, x.Type.Results)
				visitList(x.Body.List, seen)
				return false
			case *ast.BlockStmt:
				visitList(x.List, map[string]bool{})
				return false
			case *ast.CaseClause:
				visitList(x.Body, map[string]bool{})
				return false
			case *ast.CommClause:
				visitList(x.Body, map[string]bool{})
				return false
			}
			return true
		})
	}
	visitList = func(stmts []ast.Stmt, seen map[string]bool) {
		for _, stmt := range stmts {
			switch s := stmt.(type) {
			case *ast.DeclStmt:
				if gen, ok := s.Decl.(*ast.GenDecl); ok && gen.Tok == token.VAR {
					for _, spec := range gen.Specs {
						for _, id := range spec.(*ast.ValueSpec).Names {
							if id.Name != "_" && seen[id.Name] {
								dups = append(dups, id.Name)
							}
							seen[id.Name] = true
						}
					}
				}
			case *ast.AssignStmt:
				if s.Tok == token.DEFINE {
					fresh := false
					for _, lhs := range s.Lhs {
						if id, ok := lhs.(*ast.Ident); ok && id.Name != "_" && !seen[id.Name] {
							fresh = true
							seen[id.Name] = true
						}
					}
					if !fresh {
						dups = append(dups, "no new variable on left side of :=")
					}
				}
			}
			visitNested(stmt)
		}
	}
	seen := map[string]bool{}
	declareFields(seen, fn.Recv, fn.Type.Params, fn.Type.Results)
	visitList(fn.Body.List, seen)
	return dups
}
