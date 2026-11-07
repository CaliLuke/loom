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

func TestViewedEnvelopeDeclarationIdentity(t *testing.T) {
	for _, second := range []string{"FooBar", "FooView", "FooMap", "ordinary"} {
		t.Run(second, func(t *testing.T) {
			root := codegen.RunDSL(t, func() {
				foo := dsl.ResultType("application/vnd.foo", "Foo", func() {
					dsl.Attribute("value", dsl.String)
					for _, view := range []string{"default", "bar"} {
						dsl.View(view, func() {
							dsl.Attribute("value")
						})
					}
				})
				other := dsl.ResultType("application/vnd.other", second, func() {
					dsl.Attribute("value", dsl.String)
					dsl.View("default", func() {
						dsl.Attribute("value")
					})
				})
				dsl.Service("envelopes", func() {
					dsl.Method("first", func() {
						dsl.Result(foo)
					})
					dsl.Method("second", func() {
						dsl.Result(other)
					})
				})
			})
			services := NewServicesData(root)
			views := renderViewsFile(t, root, services)
			service := renderServiceFile(t, root, services)
			file, err := parser.ParseFile(token.NewFileSet(), "view.go", "package views\n"+views, 0)
			require.NoError(t, err)
			require.Empty(t, servicePackageDuplicates(file))
			data := services.Get("envelopes")
			for _, envelope := range data.viewedResultTypes {
				name := envelope.VarName
				require.Contains(t, views, "type "+name+" struct")
				require.Equal(t, envelope.ViewsPkg+"."+name, envelope.FullName)
				require.Equal(t, "*"+envelope.FullName, envelope.FullRef)
				require.Equal(t, "*"+name, envelope.Validate.Ref)
				require.Contains(t, views, "func "+envelope.Validate.Name+"(result *"+name+",")
				require.Contains(t, service, envelope.FullName+"{Projected: p,")
				require.Contains(t, envelope.Init.Code, envelope.ViewsPkg+"."+envelope.Validate.Name+"(")
			}
			require.NotContains(t, views, "loom:viewed-envelope")
			require.NotContains(t, service, "loom:viewed-envelope")
			require.NotContains(t, views, "LoomViewedEnvelope")
			require.NotContains(t, service, "LoomViewedEnvelope")
			for _, metadata := range collectViewedTypes(data) {
				require.Contains(t, views, metadata.MapName+" = map[string][]string{")
			}
			if second == "FooMap" {
				require.Contains(t, views, "type FooMap2 struct")
				require.Contains(t, views, "FooMap = map[string][]string{")
			}
			if second == "FooView" {
				require.Contains(t, views, "type FooView struct")
				require.Contains(t, views, "type FooView2 struct")
				require.Contains(t, service, "envelopesviews.FooView2")
			}
		})
	}
}

func TestViewedEnvelopePhysicalIdentity(t *testing.T) {
	scope := codegen.NewNameScope()
	scope.SetTypeFamilies(viewTypeFamilies)
	projected := &expr.AttributeExpr{Type: &expr.UserTypeExpr{
		TypeName:      "FooView",
		AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{}},
	}}
	require.Equal(t, "FooView", scope.GoValueTypeName(projected))
	result := &expr.ResultTypeExpr{
		UserTypeExpr: &expr.UserTypeExpr{TypeName: "FooView", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{}}},
		Identifier:   "application/vnd.fooview",
	}
	envelope := allocateViewedEnvelope(&expr.AttributeExpr{Type: result}, projected, scope)
	require.NotEqual(t, projected.Type.Hash(), envelope.Type.Hash())
	require.Equal(t, "FooView2", scope.GoValueTypeName(envelope))
	require.Equal(t, "*views.FooView2", scope.GoFullTypeRef(envelope, "views"))
	require.Equal(t, "FooView2", scope.GoValueTypeName(expr.DupAtt(envelope)))
	repeated := allocateViewedEnvelope(&expr.AttributeExpr{Type: result}, projected, scope)
	require.Equal(t, envelope.Type.Hash(), repeated.Type.Hash())
	require.Equal(t, "FooView2", scope.GoValueTypeName(repeated))
	require.Equal(t, []string{"ValidateFooView2"}, viewTypeFamilies(envelope.Type, "FooView2"))
}
