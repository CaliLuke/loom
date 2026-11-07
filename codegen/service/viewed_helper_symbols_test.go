package service

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestViewedHelpersShareServicePackage(t *testing.T) {
	for _, collision := range []string{"ProjectFoo", "NewFoo", "NewViewedFoo", "NewFooFromFooView", "cross-result", "ordinary"} {
		t.Run(collision, func(t *testing.T) {
			root := codegen.RunDSL(t, func() {
				member := dsl.Type(collision, func() {
					dsl.Attribute("value", dsl.String)
				})
				foo := dsl.ResultType("application/vnd.foo", "Foo", func() {
					dsl.Attribute("member", member)
					for _, view := range []string{"default", "bar"} {
						dsl.View(view, func() {
							dsl.Attribute("member")
						})
					}
				})
				bar := dsl.ResultType("application/vnd.foobar", "FooBar", func() {
					dsl.Attribute("value", dsl.String)
					dsl.View("default", func() {
						dsl.Attribute("value")
					})
				})
				dsl.Service("viewroles", func() {
					dsl.Method("read", func() {
						dsl.Result(foo)
					})
					if collision == "cross-result" {
						dsl.Method("other", func() {
							dsl.Result(bar)
						})
					}
				})
			})
			services := NewServicesData(root)
			source := renderServiceFile(t, root, services)
			file, err := parser.ParseFile(token.NewFileSet(), "service.go", "package viewroles\n"+source, 0)
			require.NoError(t, err)
			require.Empty(t, servicePackageDuplicates(file))
			data := services.Get("viewroles")
			for _, viewed := range data.viewedResultTypes {
				require.Contains(t, source, "func "+viewed.Init.Name+"(")
				require.Contains(t, source, "func "+viewed.ResultInit.Name+"(")
				matched := false
				for _, projected := range data.projectedTypes {
					if projected.Type.Hash() != expr.AsObject(viewed.Type).Attribute("projected").Type.Hash() {
						continue
					}
					matched = true
					for _, init := range projected.TypeInits {
						require.Contains(t, viewed.ResultInit.Code, init.Name+"(")
					}
					for _, projection := range projected.Projections {
						require.Contains(t, viewed.Init.Code, projection.Name+"(")
					}
				}
				require.True(t, matched, "each wrapper must be checked against its conversion declarations")
			}
			if collision == "ordinary" {
				for _, name := range []string{"ProjectFoo", "NewFoo", "NewViewedFoo", "NewFooFromFooView"} {
					require.Contains(t, source, "func "+name+"(")
				}
			}
		})
	}
}

func TestViewedHelperIdentityAfterProjectionNaming(t *testing.T) {
	scope, viewScope := codegen.NewNameScope(), codegen.NewNameScope()
	result := &expr.ResultTypeExpr{
		UserTypeExpr: &expr.UserTypeExpr{TypeName: "Foo", UID: "foo", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{}}},
	}
	for _, name := range []string{"default", "bar"} {
		result.Views = append(result.Views, &expr.ViewExpr{
			Name:          name,
			AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{}},
			Parent:        result,
		})
	}
	projected := result.Dup(expr.DupAtt(result.Attribute()))
	projected.Rename("FooView")
	att := &expr.AttributeExpr{Type: result}
	projection := &expr.AttributeExpr{Type: projected}
	viewScope.Unique("FooView")
	require.Equal(t, "FooView2", viewScope.GoValueTypeName(projection))
	scope.Unique("ProjectFoo")
	scope.Unique("NewFooFromFooView2")
	names := allocateViewConversions(att, projection, scope, viewScope)
	require.Equal(t, "ProjectFoo2", names.projections["default"])
	require.Equal(t, "NewFooFromFooView22", names.reconstructions["default"])
	require.Equal(t, names, allocateViewConversions(att, projection, scope, viewScope))
	att.Nullable = true
	require.Equal(t, names, allocateViewConversions(att, projection, scope, viewScope), "occurrence wrappers share declared conversion helpers")
}

func TestViewedWrapperOccurrenceIdentity(t *testing.T) {
	result := &expr.ResultTypeExpr{
		UserTypeExpr: &expr.UserTypeExpr{TypeName: "Foo", UID: "foo", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{}}},
	}
	att := &expr.AttributeExpr{Type: result}
	envelope := &expr.AttributeExpr{Type: result.Dup(expr.DupAtt(result.Attribute()))}
	scope := codegen.NewNameScope()
	scope.Unique("NewViewedFoo")
	scope.Unique("NewFoo")
	plain := allocateViewedWrappers(att, envelope, scope, "Foo")
	require.Equal(t, "NewViewedFoo2", plain.toViewed)
	require.Equal(t, "NewFoo2", plain.toResult)
	att.Meta = expr.MetaExpr{expr.ViewMetaKey: []string{"tiny"}}
	require.Equal(t, plain, allocateViewedWrappers(att, envelope, scope, "Foo"))
	att.Nullable = true
	envelope.Type.(expr.UserType).Rename("FooNullable")
	nullable := allocateViewedWrappers(att, envelope, scope, "FooNullable")
	require.Equal(t, "NewViewedFooNullable", nullable.toViewed)
	require.Equal(t, "NewFooNullable", nullable.toResult)
	require.Equal(t, nullable, allocateViewedWrappers(att, envelope, scope, "FooNullable"))
}
