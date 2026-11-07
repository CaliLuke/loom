package service

import (
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestServicePackageTypeFamilies(t *testing.T) {
	for _, tc := range []struct {
		name      string
		names     []string
		validated bool
	}{
		{"union companions", []string{"Choice", "ChoiceKind", "ChoiceKindCat", "NewChoiceCat"}, false},
		{"validator", []string{"Choice", "ValidateCat"}, true},
		{"ordinary", []string{"Choice", "Other"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := codegen.RunDSL(t, func() {
				var cat any = dsl.String
				if tc.validated {
					cat = dsl.Type("Cat", func() {
						dsl.Attribute("name", dsl.String, func() {
							dsl.MinLength(1)
						})
						dsl.Required("name")
					})
				}
				values := make([]any, len(tc.names))
				for i, name := range tc.names {
					values[i] = dsl.Type(name, dsl.OneOf(func() {
						dsl.Untagged()
						dsl.Attribute("cat", cat)
						dsl.Attribute("dog", dsl.Int)
					}))
				}
				dsl.Service("families", func() {
					dsl.Method("read", func() {
						dsl.Payload(func() {
							for i, value := range values {
								dsl.Attribute(tc.names[i], value)
							}
						})
					})
				})
			})
			services := NewServicesData(root)
			source := renderServiceFile(t, root, services)
			file, err := parser.ParseFile(token.NewFileSet(), "service.go", "package families\n"+source, 0)
			require.NoError(t, err)
			require.Empty(t, servicePackageDuplicates(file), "emitted service declaration families must be distinct")
			data := services.Get("families")
			for _, union := range data.unions {
				require.Contains(t, source, "type "+union.Name+" struct")
				for _, field := range union.Fields {
					require.Contains(t, source, "func New"+union.Name+field.MethodName+"(")
				}
			}
		})
	}
}

func TestViewPackageTypeFamilies(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		choice := dsl.Type("Choice", dsl.OneOf(func() {
			dsl.Untagged()
			dsl.Attribute("catView", dsl.String)
			dsl.Attribute("dog", dsl.Int)
		}))
		collision := dsl.Type("NewChoiceViewCat", func() {
			dsl.Attribute("name", dsl.String)
		})
		result := dsl.ResultType("application/vnd.family", "Family", func() {
			dsl.Attribute("choice", choice)
			dsl.Attribute("collision", collision)
			dsl.View("default", func() {
				dsl.Attribute("choice")
				dsl.Attribute("collision")
			})
		})
		dsl.Service("families", func() {
			dsl.Method("read", func() {
				dsl.Result(result)
			})
		})
	})
	services := NewServicesData(root)
	source := renderViewsFile(t, root, services)
	file, err := parser.ParseFile(token.NewFileSet(), "views.go", "package familiesviews\n"+source, 0)
	require.NoError(t, err)
	require.Empty(t, servicePackageDuplicates(file), "projected declarations must own their complete helper families")
}

func servicePackageDuplicates(file *ast.File) []string {
	counts := make(map[string]int)
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
	var duplicates []string
	for name, count := range counts {
		if count > 1 {
			duplicates = append(duplicates, name)
		}
	}
	sort.Strings(duplicates)
	return duplicates
}

func TestNativeUnionAliasFamilies(t *testing.T) {
	for _, primary := range []string{"Choice", "New"} {
		for _, view := range []bool{false, true} {
			t.Run(primary+map[bool]string{false: "/service", true: "/view"}[view], func(t *testing.T) {
				union := &expr.Union{TypeName: primary}
				for _, name := range []string{"Kind", "Kind2", "Kind3", "Cat", "KindCat", "NewCat"} {
					union.Values = append(union.Values, &expr.NamedAttributeExpr{Name: name, Attribute: &expr.AttributeExpr{Type: expr.Any}})
				}
				scope := codegen.NewNameScope()
				scope.SetTypeFamilies(func(typ expr.DataType, name string) []string {
					return nativeTypeDeclarations(typ, name, nil).names
				})
				var data *UnionTypeData
				require.NotPanics(t, func() {
					if view {
						data = buildViewUnionTypeData(union, scope, nil)
					} else {
						data = buildUnionTypeData(union, scope, nil)
					}
				}, "a primary suffix cannot fix a collision inside its own family")
				require.NotNil(t, data)
				symbols := nativeTypeDeclarations(union, data.Name, nil)
				require.Equal(t, data.Name+"Kind2", symbols.aliases["Kind2"])
				require.Equal(t, data.Name+"Kind3", symbols.aliases["Kind3"])
				require.Equal(t, data.Name+"Kind4", symbols.aliases["Kind"])
				slices.Reverse(union.Values)
				require.Equal(t, symbols.aliases, nativeTypeDeclarations(union, data.Name, nil).aliases)
				occupied := map[string]bool{data.Name: true, data.KindName: true}
				for _, field := range data.Fields {
					for _, name := range []string{field.KindConst, "New" + data.Name + field.MethodName} {
						require.False(t, occupied[name], name)
						occupied[name] = true
					}
				}
				for _, field := range data.Fields {
					require.True(t, field.EmitPrimitiveAlias)
					require.False(t, occupied[field.FieldType], field.FieldType)
					occupied[field.FieldType] = true
				}
			})
		}
	}
}

func TestViewSanitizedTypeFamilies(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		result := dsl.ResultType("application/vnd.viewnames", "ViewNames", func() {
			dsl.Attribute("name", dsl.String)
			for _, view := range []string{"default", "tiny-view", "tiny_view", "tiny_view2"} {
				dsl.View(view, func() {
					dsl.Attribute("name")
				})
			}
		})
		dsl.Service("viewnames", func() {
			dsl.Method("read", func() {
				dsl.Result(result)
			})
		})
	})
	services := NewServicesData(root)
	views := renderViewsFile(t, root, services)
	service := renderServiceFile(t, root, services)
	for _, source := range []string{views, service} {
		file, err := parser.ParseFile(token.NewFileSet(), "generated.go", "package viewnames\n"+source, 0)
		require.NoError(t, err)
		require.Empty(t, servicePackageDuplicates(file))
		for _, view := range []string{"tiny-view", "tiny_view", "tiny_view2"} {
			require.Contains(t, source, `case "`+view+`":`)
		}
	}
	for view, suffix := range map[string]string{"tiny-view": "TinyView", "tiny_view": "TinyView3", "tiny_view2": "TinyView2"} {
		validator := "ValidateViewNamesView" + suffix
		projection := "ProjectViewNames" + suffix
		constructor := "NewViewNamesFromViewNamesView" + suffix
		require.Contains(t, views, "func "+validator+"(")
		require.Contains(t, service, "func "+projection+"(")
		require.Contains(t, service, "func "+constructor+"(")
		for _, check := range []struct{ source, function, call string }{
			{views, "ValidateViewNames", validator},
			{service, "NewViewedViewNames", projection},
			{service, "NewViewNames", constructor},
		} {
			body := generatedFunction(t, check.source, check.function)
			start := strings.Index(body, `case "`+view+`":`)
			require.NotEqual(t, -1, start)
			branch := strings.SplitN(body[start:], "break", 2)[0]
			branch = strings.SplitN(branch, "case ", 3)[1]
			require.Contains(t, branch, check.call+"(")
		}
	}
}

func TestViewHelperSuffixIdentity(t *testing.T) {
	rt := &expr.ResultTypeExpr{}
	for _, name := range []string{"default", "tiny-view", "tiny_view", "tiny_view2"} {
		rt.Views = append(rt.Views, &expr.ViewExpr{Name: name})
	}
	expected := map[string]string{"tiny-view": "TinyView", "tiny_view": "TinyView3", "tiny_view2": "TinyView2"}
	require.Equal(t, expected, viewHelperSuffixes(rt))
	slices.Reverse(rt.Views)
	require.Equal(t, expected, viewHelperSuffixes(rt))
	rt.UserTypeExpr = &expr.UserTypeExpr{TypeName: "ChildView", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{}}}
	carrier := newViewValidationCarrier(&expr.AttributeExpr{Type: rt}, projectedTypeContext("", true, codegen.NewNameScope()), "tiny_view")
	require.Equal(t, "ValidateChildViewTinyView3", carrier.function.Name)
}
