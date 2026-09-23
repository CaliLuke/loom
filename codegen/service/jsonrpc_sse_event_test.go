package service

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
)

// TestJSONRPCSSEEventTypesDeclareMethodsOnlyOnLocalTypes covers the JSON-RPC
// SSE event types of the service package. Go only allows methods on named
// types declared in the same package, so the per-method event of a
// primitive, collection, Any, Any-based alias, or externally located streamed
// result is an alias of that type, and the service-level Event accepts any
// value when one of its streamed types cannot carry a marker method. Local
// user types keep their marker methods. The Event set holds only streamed
// types, one per distinct Go type, so the transport Send type switch has
// exactly one case for each.
func TestJSONRPCSSEEventTypesDeclareMethodsOnlyOnLocalTypes(t *testing.T) {
	// userTypes holds the user types declared by the running DSL.
	type userTypes struct {
		note, item, doc, external any
	}
	cases := []struct {
		name         string
		results      map[string]func(userTypes) any
		unary        func(userTypes) any
		mixedResult  func(userTypes) any
		wantAliases  map[string]string
		wantMarkers  []string
		noMarkers    []string
		wantEvents   []string
		wantEventAny bool
	}{
		{
			name: "built-in results",
			results: map[string]func(userTypes) any{
				"StreamString": func(userTypes) any { return dsl.String },
				"StreamInt":    func(userTypes) any { return dsl.Int },
				"StreamBool":   func(userTypes) any { return dsl.Boolean },
				"StreamBytes":  func(userTypes) any { return dsl.Bytes },
				"StreamAny":    func(userTypes) any { return dsl.Any },
				"StreamArray":  func(userTypes) any { return dsl.ArrayOf(dsl.String) },
				"StreamMap":    func(userTypes) any { return dsl.MapOf(dsl.String, dsl.Int) },
				"StreamNotes":  func(u userTypes) any { return dsl.ArrayOf(u.note) },
			},
			wantAliases: map[string]string{
				"StreamStringEvent": "string",
				"StreamIntEvent":    "int",
				"StreamBoolEvent":   "bool",
				"StreamBytesEvent":  "[]byte",
				"StreamAnyEvent":    "loom.JSONValue",
				"StreamArrayEvent":  "[]string",
				"StreamMapEvent":    "map[string]int",
				"StreamNotesEvent":  "[]*Note",
				"Event":             "any",
			},
			wantEventAny: true,
		},
		{
			name: "user type and built-in results",
			results: map[string]func(userTypes) any{
				"StreamString": func(userTypes) any { return dsl.String },
				"StreamNote":   func(u userTypes) any { return u.note },
			},
			wantAliases: map[string]string{
				"StreamStringEvent": "string",
				"Event":             "any",
			},
			wantMarkers:  []string{"Note.isStreamNoteEvent"},
			wantEventAny: true,
		},
		{
			name: "user type stream with built-in unary result",
			results: map[string]func(userTypes) any{
				"StreamNote": func(u userTypes) any { return u.note },
			},
			unary:       func(userTypes) any { return dsl.String },
			wantMarkers: []string{"Note.isStreamNoteEvent", "Note.isjSONRPCSSEEventsEvent"},
			wantEvents:  []string{"*Note"},
		},
		{
			name: "externally located user type",
			results: map[string]func(userTypes) any{
				"StreamExternal": func(u userTypes) any { return u.external },
			},
			wantAliases: map[string]string{
				"StreamExternalEvent": "*types.External",
				"Event":               "any",
			},
			wantEventAny: true,
		},
		{
			name: "local result with built-in streaming result",
			results: map[string]func(userTypes) any{
				"Watch": func(userTypes) any { return dsl.String },
			},
			mixedResult: func(u userTypes) any { return u.note },
			wantAliases: map[string]string{
				"WatchEvent": "string",
				"Event":      "any",
			},
			wantEventAny: true,
		},
		{
			name: "built-in result with local streaming result",
			results: map[string]func(userTypes) any{
				"Watch": func(u userTypes) any { return u.note },
			},
			mixedResult: func(userTypes) any { return dsl.String },
			wantMarkers: []string{"Note.isWatchEvent", "Note.isjSONRPCSSEEventsEvent"},
			wantEvents:  []string{"*Note"},
		},
		{
			name: "local result with local streaming result",
			results: map[string]func(userTypes) any{
				"Watch": func(u userTypes) any { return u.item },
			},
			mixedResult: func(u userTypes) any { return u.note },
			wantMarkers: []string{"Item.isWatchEvent", "Item.isjSONRPCSSEEventsEvent"},
			noMarkers:   []string{"Note.isjSONRPCSSEEventsEvent"},
			wantEvents:  []string{"*Item"},
		},
		{
			name: "Any-based user type",
			results: map[string]func(userTypes) any{
				"StreamDoc": func(u userTypes) any { return u.doc },
			},
			wantAliases: map[string]string{
				"Doc":            "loom.JSONValue",
				"StreamDocEvent": "Doc",
				"Event":          "any",
			},
			wantEvents:   []string{"Doc"},
			wantEventAny: true,
		},
		{
			name: "Any and Any-based user type share one Event type",
			results: map[string]func(userTypes) any{
				"StreamAny":  func(userTypes) any { return dsl.Any },
				"StreamDoc":  func(u userTypes) any { return u.doc },
				"StreamAny2": func(userTypes) any { return dsl.Any },
				"StreamStr":  func(userTypes) any { return dsl.String },
			},
			wantEventAny: true,
			wantEvents:   []string{"loom.JSONValue|Doc", "string"},
		},
		{
			name: "local user types only",
			results: map[string]func(userTypes) any{
				"StreamNote": func(u userTypes) any { return u.note },
			},
			wantMarkers: []string{"Note.isStreamNoteEvent", "Note.isjSONRPCSSEEventsEvent"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := codegen.RunDSL(t, func() {
				dsl.API("jsonrpc-sse-events", func() {
					dsl.JSONRPC(func() {})
				})
				u := userTypes{
					note: dsl.Type("Note", func() {
						dsl.Attribute("text", dsl.String)
						dsl.Required("text")
					}),
					item: dsl.Type("Item", func() {
						dsl.Attribute("id", dsl.Int)
						dsl.Required("id")
					}),
					doc: dsl.Type("Doc", dsl.Any),
					external: dsl.Type("External", func() {
						dsl.Attribute("text", dsl.String)
						dsl.Meta("struct:pkg:path", "types")
					}),
				}
				dsl.Service("JSONRPCSSEEvents", func() {
					dsl.JSONRPC(func() {
						dsl.POST("/rpc")
					})
					for name, result := range tc.results {
						dsl.Method(name, func() {
							dsl.Payload(func() {
								dsl.ID("id", dsl.String)
							})
							dsl.StreamingResult(result(u))
							if tc.mixedResult != nil {
								dsl.Result(tc.mixedResult(u))
							}
							dsl.JSONRPC(func() {
								dsl.ServerSentEvents()
							})
						})
					}
					if tc.unary != nil {
						dsl.Method("Get", func() {
							dsl.Payload(func() {
								dsl.ID("id", dsl.String)
							})
							dsl.Result(tc.unary(u))
							dsl.JSONRPC(func() {})
						})
					}
				})
			})
			services := NewServicesData(root)
			files := Files("example.com/events/gen", root.Services[0], services, make(map[string][]string))
			require.NotEmpty(t, files)
			buf := new(bytes.Buffer)
			for _, s := range files[0].AllSections() {
				require.NoError(t, s.Write(buf))
			}
			code := buf.String()
			file, err := parser.ParseFile(token.NewFileSet(), "service.go", code, 0)
			require.NoError(t, err, code)

			aliases, locals := typeDecls(file)
			for name, want := range tc.wantAliases {
				require.Equalf(t, want, aliases[name], "alias %s in\n%s", name, code)
			}
			markers := make(map[string]bool)
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Recv == nil {
					continue
				}
				recv := receiverName(fn.Recv.List[0].Type)
				require.Truef(t, locals[recv], "method %s declared on %s, which is not a local named type:\n%s", fn.Name.Name, types.ExprString(fn.Recv.List[0].Type), code)
				markers[recv+"."+fn.Name.Name] = true
			}
			for _, marker := range tc.wantMarkers {
				require.Truef(t, markers[marker], "missing marker %s in\n%s", marker, code)
			}
			for _, marker := range tc.noMarkers {
				require.Falsef(t, markers[marker], "unexpected marker %s in\n%s", marker, code)
			}
			if tc.wantEvents != nil {
				refs, _ := jsonrpcSSEEventTypes(services.Get("JSONRPCSSEEvents"))
				require.Lenf(t, refs, len(tc.wantEvents), "event types %v", refs)
				for _, want := range tc.wantEvents {
					require.Truef(t, slices.ContainsFunc(strings.Split(want, "|"), func(ref string) bool {
						return slices.Contains(refs, ref)
					}), "event types %v missing %s", refs, want)
				}
			}
			if tc.wantEventAny {
				require.NotContains(t, code, "isjSONRPCSSEEventsEvent")
			}
		})
	}
}

// typeDecls returns the aliases declared in file keyed by name with their
// target type, and the set of non-alias, non-interface named types.
func typeDecls(file *ast.File) (map[string]string, map[string]bool) {
	aliases := make(map[string]string)
	locals := make(map[string]bool)
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			ts := spec.(*ast.TypeSpec)
			if ts.Assign.IsValid() {
				aliases[ts.Name.Name] = types.ExprString(ts.Type)
				continue
			}
			if _, isInterface := ts.Type.(*ast.InterfaceType); !isInterface {
				locals[ts.Name.Name] = true
			}
		}
	}
	return aliases, locals
}

func receiverName(expr ast.Expr) string {
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name
	}
	return ""
}
