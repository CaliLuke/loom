package codegen

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/require"
)

func TestHTTPPackageSymbols(t *testing.T) {
	for _, tc := range []struct {
		name      string
		unions    []string
		validated bool
	}{
		{"fixed", []string{"Server", "MountPoint", "New", "Mount", "Client", "NewClient"}, false},
		{"endpoint", []string{"DecodeReadRequest", "NewReadHandler", "MountReadHandler", "EncodeReadResponse", "DecodeReadResponse"}, false},
		{"validator", []string{"ValidateCatRequestBody"}, true},
		{"union companions", []string{"Choice", "NewChoiceCat", "ChoiceKindCat"}, false},
		{"ordinary", []string{"Choice", "Other"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := RunHTTPDSL(t, packageBodySymbolDSL(tc.unions, tc.validated, false, false))
			dir := t.TempDir()
			renderHTTPModuleFiles(t, dir, "example.com/carrier/gen", root)
			for _, pkg := range []string{"server", "client"} {
				require.Empty(t, packageSymbolDuplicates(t, filepath.Join(dir, "gen", "http", "carrier", pkg)), "%s package", pkg)
			}
		})
	}
}

func packageBodySymbolDSL(names []string, validated, viewed, roles bool) func() {
	return func() {
		values := make([]any, len(names))
		var nativeNew any
		for i, name := range names {
			if viewed && name == "New" {
				values[i] = packageSymbolNativeAny(name)
				nativeNew = values[i]
			} else {
				values[i] = packageSymbolChoice(name, String, Int)
			}
		}
		var cat any
		if validated {
			cat, _ = packageSymbolPets()
		}
		var result, nativeChoice any
		if viewed {
			nativeChoice = packageSymbolNativeAny("NativeChoice")
			choice := Type("ViewedChoice", OneOf(func() {
				Untagged()
				Attribute("catView", String)
				Attribute("dog", Int)
			}))
			collision := Type("NewViewedChoiceViewCat", func() {
				Attribute("name", String)
			})
			result = ResultType("application/vnd.package-symbols", "PackageSymbols", func() {
				Attribute("choice", choice)
				Attribute("collision", collision)
				Attribute("nativeChoice", nativeChoice)
				Attribute("nativeNew", nativeNew)
				for _, view := range []string{"default", "tiny-view", "tiny_view", "tiny_view2"} {
					View(view, func() {
						Attribute("choice")
						Attribute("collision")
						Attribute("nativeChoice")
						Attribute("nativeNew")
					})
				}
			})
		}
		Service("carrier", func() {
			Method("read", func() {
				Payload(func() {
					for i, value := range values {
						Attribute(names[i], value)
					}
					if validated {
						Attribute("cat", cat)
					}
					if viewed {
						Attribute("nativeChoice", nativeChoice)
					}
				})
				if viewed {
					Result(result)
				}
				HTTP(func() {
					if roles {
						POST("/items/read")
						POST("/items/read-alt")
					} else {
						POST("/")
					}
					if viewed {
						Response(StatusOK)
					} else {
						Response(StatusNoContent)
					}
				})
			})
			if roles {
				packageRoleMethods()
			}
		})
		if viewed {
			packageViewedSymbolService(true)
		}
	}
}

// packageViewedSymbolService exercises the service-package roles identified in
// review G1, including helper/type and cross-result helper collisions.
func packageViewedSymbolService(colliding bool) {
	names := []string{"OrdinaryMember"}
	if colliding {
		names = []string{"ProjectFoo", "NewFoo", "NewViewedFoo", "NewFooFromFooView"}
	}
	members := make([]any, len(names))
	for i, name := range names {
		members[i] = Type(name, func() {
			Attribute("value", String)
		})
	}
	foo := ResultType("application/vnd.naming-foo", "Foo", func() {
		for i, member := range members {
			Attribute(names[i], member)
		}
		for _, view := range []string{"default", "bar"} {
			View(view, func() {
				for _, name := range names {
					Attribute(name)
				}
			})
		}
	})
	var extraResults []any
	if colliding {
		for _, name := range []string{"FooBar", "FooView", "FooMap"} {
			extraResults = append(extraResults, ResultType("application/vnd.naming-"+name, name, func() {
				Attribute("value", String)
				View("default", func() {
					Attribute("value")
				})
			}))
		}
	}
	Service("viewroles", func() {
		Method("read", func() {
			Result(foo)
			HTTP(func() {
				GET("/views/read")
			})
		})
		for i, result := range extraResults {
			method := []string{"other", "projected", "metadata"}[i]
			Method(method, func() {
				Result(result)
				HTTP(func() {
					GET("/views/" + method)
				})
			})
		}
	})
}

func packageSymbolChoice(name string, cat, dog any) any {
	return Type(name, OneOf(func() {
		Untagged()
		Attribute("cat", cat)
		Attribute("dog", dog)
	}))
}

func packageSymbolNativeAny(name string) any {
	return Type(name, OneOf(func() {
		for _, branch := range []string{"Kind", "Kind2", "Kind3", "Cat", "KindCat", "NewCat"} {
			Attribute(branch, Any)
		}
	}))
}

func packageSymbolPets() (any, any) {
	cat := Type("Cat", func() {
		Attribute("name", String, func() {
			MinLength(1)
		})
		Required("name")
	})
	dog := Type("Dog", func() {
		Attribute("age", Int)
		Required("age")
	})
	return cat, dog
}

func packageSymbolDuplicates(t *testing.T, dir string) []string {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(dir, "*.go"))
	require.NoError(t, err)
	counts := make(map[string]int)
	for _, path := range paths {
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		require.NoError(t, err)
		for _, declaration := range file.Decls {
			switch declaration := declaration.(type) {
			case *ast.FuncDecl:
				if declaration.Recv == nil {
					counts[declaration.Name.Name]++
				}
			case *ast.GenDecl:
				for _, specification := range declaration.Specs {
					switch specification := specification.(type) {
					case *ast.TypeSpec:
						counts[specification.Name.Name]++
					case *ast.ValueSpec:
						for _, name := range specification.Names {
							if name.Name != "_" {
								counts[name.Name]++
							}
						}
					}
				}
			}
		}
	}
	var duplicates []string
	for name, count := range counts {
		if count > 1 {
			duplicates = append(duplicates, name)
		}
	}
	sort.Strings(duplicates)
	return duplicates
}

func TestHTTPPackageSymbolReferenceIdentity(t *testing.T) {
	sd := &ServiceData{Scope: codegen.NewNameScope()}
	sd.Scope.SetTypeFamilies(func(_ expr.DataType, name string) []string {
		return []string{"Validate" + name}
	})
	sd.Scope.Unique("Server")
	typeOf := func(name string) *expr.UserTypeExpr {
		return &expr.UserTypeExpr{TypeName: name, UID: name, AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{}}}
	}
	for _, name := range []string{"Server2", "Server", "ValidateCat", "Cat", "Cat2"} {
		typ := typeOf(name)
		allocated := sd.Scope.GoValueTypeName(&expr.AttributeExpr{Type: typ})
		require.Equal(t, allocated, sd.Scope.GoValueTypeName(&expr.AttributeExpr{Type: typ}))
		require.Equal(t, "*"+allocated, sd.Scope.GoTypeRef(&expr.AttributeExpr{Type: typ}))
		require.Equal(t, allocated, strings.TrimPrefix(codegen.NamedValidationFunction(&expr.AttributeExpr{Type: typ}, httpContext(sd.Scope, true, true), "body").Name, "Validate"))
	}
	require.Equal(t, "Server3", sd.Scope.GoValueTypeName(&expr.AttributeExpr{Type: typeOf("Server")}))
	require.Equal(t, "Cat2", sd.Scope.GoValueTypeName(&expr.AttributeExpr{Type: typeOf("Cat")}))
	require.Equal(t, "Cat22", sd.Scope.GoValueTypeName(&expr.AttributeExpr{Type: typeOf("Cat2")}))
}

func TestHTTPPackageSymbolMethodIndependence(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		choice := packageSymbolChoice("Server", String, Int)
		Service("carrier", func() {
			for _, name := range []string{"Client", "Mount", "Service"} {
				Method(name, func() {
					Payload(func() {
						Attribute("choice", choice)
					})
					HTTP(func() {
						POST("/" + name)
						Response(StatusNoContent)
					})
				})
			}
		})
	})
	sd := CreateHTTPServices(root).Get("carrier")
	for _, endpoint := range sd.Endpoints {
		require.Equal(t, endpoint.Method.Name, endpoint.Method.VarName)
	}
}

func TestHTTPPackageSymbolGenerated(t *testing.T) {
	names := slices.Concat([]string{"Server", "MountPoint", "New", "Mount", "Client", "NewClient", "DecodeReadRequest", "NewReadHandler", "Choice", "NewChoiceCat", "ChoiceKindCat", "ValidateCatRequestBody"}, packageRoleCollisionNames())
	root := RunHTTPDSL(t, packageBodySymbolDSL(names, true, true, true))
	dir := t.TempDir()
	if artifacts := os.Getenv("LOOM_PACKAGE_SYMBOL_ARTIFACT_DIR"); artifacts != "" {
		dir = filepath.Join(artifacts, "compiled")
		require.NoError(t, os.MkdirAll(dir, 0o755))
	}
	renderHTTPModule(t, dir, "example.com/carrier", root)
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}

func TestHTTPPackageSymbolArtifacts(t *testing.T) {
	dir := os.Getenv("LOOM_PACKAGE_SYMBOL_ARTIFACT_DIR")
	if dir == "" {
		t.Skip("set LOOM_PACKAGE_SYMBOL_ARTIFACT_DIR to retain the bounded specimen outputs")
	}
	cases := []struct {
		name   string
		design func()
	}{
		{"fixed", packageBodySymbolDSL([]string{"Server", "MountPoint", "New", "Mount", "Client", "NewClient"}, false, false, false)},
		{"endpoint", packageBodySymbolDSL([]string{"DecodeReadRequest", "NewReadHandler", "MountReadHandler", "EncodeReadResponse", "DecodeReadResponse"}, false, false, false)},
		{"validator", packageBodySymbolDSL([]string{"ValidateCatRequestBody"}, true, false, false)},
		{"union", packageBodySymbolDSL([]string{"Choice", "NewChoiceCat", "ChoiceKindCat"}, false, false, false)},
		{"ordinary", packageBodySymbolDSL([]string{"Choice", "Other"}, true, false, false)},
		{"viewed", packageBodySymbolDSL([]string{"Server", "MountPoint", "New", "Mount", "Client", "NewClient", "DecodeReadRequest", "NewReadHandler", "Choice", "NewChoiceCat", "ChoiceKindCat", "ValidateCatRequestBody"}, true, true, false)},
		{"roles", packageBodySymbolDSL(packageRoleCollisionNames(), false, false, true)},
		{"roles-ordinary", packageBodySymbolDSL([]string{"Choice"}, false, false, true)},
		{"view-ordinary", func() {
			packageViewedSymbolService(false)
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := RunHTTPDSL(t, c.design)
			target := filepath.Join(dir, c.name)
			require.NoError(t, os.MkdirAll(target, 0o755))
			renderHTTPModuleFiles(t, target, "example.com/carrier/gen", root)
		})
	}
}
