package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
)

func TestProjectedCarrierPackageReferences(t *testing.T) {
	for _, pkg := range []string{"common", "context", "svcviews"} {
		t.Run(pkg, func(t *testing.T) {
			root := codegen.RunDSL(t, func() {
				child := dsl.ResultType("application/vnd.package.child", func() {
					dsl.TypeName("Child")
					dsl.Meta("struct:pkg:path", "types/"+pkg)
					dsl.Attributes(func() {
						dsl.Attribute("id", dsl.String)
						dsl.Attribute("secret", dsl.String)
						dsl.Attribute("next", "Child")
					})
					dsl.View("default", func() {
						dsl.Attribute("id")
						dsl.Attribute("next")
					})
				})
				parent := dsl.ResultType("application/vnd.package.parent", func() {
					dsl.TypeName("Parent")
					dsl.Meta("struct:pkg:path", "types/"+pkg)
					dsl.Attributes(func() {
						dsl.Attribute("child", child)
						dsl.Attribute("children", dsl.ArrayOf(child))
					})
					dsl.View("default", func() {
						dsl.Attribute("child")
						dsl.Attribute("children")
					})
				})
				dsl.Service("svc", func() {
					dsl.Method("show", func() {
						dsl.Result(parent)
					})
				})
			})
			services := NewServicesData(root)
			for _, names := range []map[string]string{nil, {"types/" + pkg: "external2"}} {
				data := services.WithPackageNames(names).Get("svc")
				canonical := pkg
				if names != nil {
					canonical = "external2"
				}
				require.Equal(t, "*"+canonical+".Parent", data.Method("show").ResultRef)
				var conversions strings.Builder
				for _, projected := range data.projectedTypes {
					for _, conversion := range append(projected.Projections, projected.TypeInits...) {
						conversions.WriteString(conversion.Code)
						for _, helper := range conversion.Helpers {
							conversions.WriteString(helper.Code)
						}
					}
				}
				code := conversions.String()
				require.Contains(t, code, data.ViewsPkg+".ParentView")
				require.Contains(t, code, data.ViewsPkg+".ChildView")
				if canonical != data.ViewsPkg {
					require.NotContains(t, code, canonical+".ParentView", "canonical placement must not leak into view conversion snapshots")
					require.NotContains(t, code, canonical+".ChildView")
				}
				require.NotContains(t, code, "Secret:", "default child view excludes secret")
			}
		})
	}
}
