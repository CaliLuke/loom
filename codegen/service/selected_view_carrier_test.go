package service

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// TestSelectedViewNullableFieldCarrier checks nullable field guards and physical helper calls.
func TestSelectedViewNullableFieldCarrier(t *testing.T) {
	for _, required := range []bool{false, true} {
		name := "optional"
		if required {
			name = "required"
		}
		t.Run(name, func(t *testing.T) {
			root := codegen.RunDSL(t, func() {
				child := dsl.ResultType("application/vnd.child", "Child", func() {
					dsl.Attribute("text", dsl.String)
					dsl.Attribute("label", dsl.String)
					dsl.Required("text")
					dsl.View("default", func() {
						dsl.Attribute("text")
					})
					dsl.View("tiny", func() {
						dsl.Attribute("label")
					})
				})
				parent := dsl.ResultType("application/vnd.parent", "Parent", func() {
					dsl.Attribute("plain", child)
					dsl.Attribute("child", child, func() {
						dsl.Nullable()
					})
					if required {
						dsl.Required("child")
					}
					dsl.View("default", func() {
						dsl.Attribute("plain")
						dsl.Attribute("child")
					})
					dsl.View("tiny", func() {
						dsl.Attribute("plain", func() {
							dsl.View("tiny")
						})
					})
				})
				dsl.Service("carrier", func() {
					dsl.Method("show", func() {
						dsl.Result(parent)
					})
				})
			})
			views := renderViewsFile(t, root, NewServicesData(root))
			body := generatedFunction(t, views, "ValidateParentView")
			t.Run("nullable operations", func(t *testing.T) {
				require.NotContains(t, body, "result.Child != nil")
				require.NotContains(t, body, "result.Child == nil")
				require.Contains(t, body, "result.Child.Value()")
				if required {
					require.Contains(t, body, "!result.Child.Present()")
				}
			})
			t.Run("ordinary and excluded controls", func(t *testing.T) {
				require.Contains(t, body, "result.Plain != nil")
				tiny := generatedFunction(t, views, "ValidateParentViewTiny")
				require.NotContains(t, tiny, "result.Child")
				require.Contains(t, tiny, "ValidateChildViewTiny(result.Plain,")
			})
		})
	}
}

// TestSelectedViewIntrinsicNullableCarrier checks intrinsic root and collection element operations.
func TestSelectedViewIntrinsicNullableCarrier(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		child := dsl.ResultType("application/vnd.child", "Child", func() {
			dsl.Attributes(func() {
				dsl.Nullable()
				dsl.Attribute("text", dsl.String)
				dsl.Required("text")
			})
			dsl.View("default", func() {
				dsl.Attribute("text")
			})
		})
		dsl.Service("carrier", func() {
			dsl.Method("show", func() {
				dsl.Result(child)
			})
			dsl.Method("list", func() {
				dsl.Result(dsl.CollectionOf(child))
			})
		})
	})
	views := renderViewsFile(t, root, NewServicesData(root))
	t.Run("root", func(t *testing.T) {
		body := generatedFunction(t, views, "ValidateChildView")
		require.Contains(t, body, "result.Value()", "intrinsic wrapper must unwrap before field access")
		require.NotContains(t, body, "result.Text")
		require.Contains(t, body, `loom.MissingFieldError("text", validationPath)`)
	})
	t.Run("collection element", func(t *testing.T) {
		body := generatedFunction(t, views, "ValidateChildCollectionView")
		require.NotContains(t, body, "item == nil", "nullable element uses a wrapper, not a pointer")
		require.Contains(t, body, "item.Present()", "collection positions cannot be omitted")
	})
}

// TestSelectedViewRootCarrierBoundary checks root admission, envelope references and projection entry.
func TestSelectedViewRootCarrierBoundary(t *testing.T) {
	for _, nullable := range []bool{false, true} {
		name := "ordinary"
		if nullable {
			name = "intrinsic"
		}
		t.Run(name, func(t *testing.T) {
			root := codegen.RunDSL(t, func() {
				child := dsl.ResultType("application/vnd.child", "Child", func() {
					dsl.Attributes(func() {
						if nullable {
							dsl.Nullable()
						}
						dsl.Attribute("text", dsl.String)
						dsl.Attribute("label", dsl.String)
						dsl.Required("text")
					})
					dsl.View("default", func() {
						dsl.Attribute("text")
					})
					dsl.View("tiny", func() {
						dsl.Attribute("label")
					})
				})
				dsl.Service("carrier", func() {
					dsl.Method("show", func() {
						dsl.Result(child)
					})
					dsl.Method("list", func() {
						dsl.Result(dsl.CollectionOf(child))
					})
				})
			})
			services := NewServicesData(root)
			service := services.Get("carrier")
			viewed := service.Method("show").ViewedResult
			t.Run("metadata envelope", func(t *testing.T) {
				require.Equal(t, "*carrierviews.Child", viewed.FullRef)
				require.Equal(t, "carrierviews.Child", viewed.FullName)
				require.Equal(t, "*Child", viewed.Validate.Ref)
				require.Equal(t, "*carrierviews.Child", viewed.Init.ReturnTypeRef)
				require.Equal(t, "*carrierviews.Child", viewed.ResultInit.Args[0].Ref)
				require.Equal(t, "carrierviews.ChildCollection", service.Method("list").ViewedResult.FullRef)
			})
			t.Run("root admission", func(t *testing.T) {
				code := viewed.Init.Code
				if nullable {
					require.Contains(t, code, "!res.Present()")
					require.Contains(t, viewed.Validate.Validate, "!result.Projected.Present()")
					require.NotContains(t, code, "res == nil")
					require.NotContains(t, code, "res.IsNull()", "explicit nullable null must reach selected-view dispatch")
				} else {
					require.Contains(t, code, "res == nil")
				}
				require.Contains(t, code, "switch view")
				require.Contains(t, code, "loom.InvalidEnumValueError")
			})
			t.Run("projection entries", func(t *testing.T) {
				code := renderServiceFile(t, root, services)
				for _, name := range []string{"ProjectChild", "ProjectChildTiny", "NewChildFromChildView", "NewChildFromChildViewTiny"} {
					body := generatedFunction(t, code, name)
					if nullable {
						require.NotContains(t, body, " == nil")
						require.Contains(t, body, ".Value()")
						require.Contains(t, body, ".IsNull()")
					} else {
						require.Contains(t, body, " == nil")
					}
				}
			})
		})
	}
}

// TestSelectedViewRootOccurrence preserves method-local presence across the viewed envelope.
func TestSelectedViewRootOccurrence(t *testing.T) {
	for _, nullableFirst := range []bool{false, true} {
		t.Run(fmt.Sprint(nullableFirst), func(t *testing.T) {
			root := codegen.RunDSL(t, func() {
				child := dsl.ResultType("application/vnd.child", "Child", func() {
					dsl.Attribute("text", dsl.String)
					dsl.Attribute("label", dsl.String)
					dsl.Required("text")
					dsl.View("default", func() {
						dsl.Attribute("text")
					})
					dsl.View("tiny", func() {
						dsl.Attribute("label")
					})
				})
				dsl.Service("carrier", func() {
					methods := []string{"plain", "show"}
					if nullableFirst {
						methods = []string{"show", "plain"}
					}
					for _, method := range methods {
						dsl.Method(method, func() {
							dsl.Result(child, func() {
								if method == "show" {
									dsl.Nullable()
								}
							})
						})
					}
				})
			})
			services := NewServicesData(root)
			viewed := services.Get("carrier").Method("show").ViewedResult
			require.True(t, expr.IsNullable(projectedViewedResultAttribute(viewed)))
			plain := services.Get("carrier").Method("plain").ViewedResult
			require.False(t, expr.IsNullable(projectedViewedResultAttribute(plain)))
			require.NotEqual(t, plain.Init.Args[0].Ref, viewed.Init.Args[0].Ref)
			require.Contains(t, viewed.Init.Code, "res.Value()")
			require.Contains(t, viewed.Init.Code, "ProjectChild(&actual)")
			require.Contains(t, viewed.ResultInit.Code, "vres.Projected.Value()")
			require.Contains(t, viewed.Validate.Validate, "result.Projected.Value()")
			code := renderServiceFile(t, root, services)
			require.Contains(t, generatedFunction(t, code, "ProjectChild"), "res == nil")
		})
	}
}
