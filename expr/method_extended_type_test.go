package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// TestExtendedMethodTypesOwnTheirShape checks each method type position and
// both declaration orders against a plain use of the same named type.
func TestExtendedMethodTypesOwnTheirShape(t *testing.T) {
	for _, slot := range []struct {
		name      string
		define    func(any, ...any)
		attribute func(*expr.MethodExpr) *expr.AttributeExpr
	}{
		{"Payload", Payload, func(m *expr.MethodExpr) *expr.AttributeExpr {
			return m.Payload
		}},
		{"Result", Result, func(m *expr.MethodExpr) *expr.AttributeExpr {
			return m.Result
		}},
		{"StreamingPayload", StreamingPayload, func(m *expr.MethodExpr) *expr.AttributeExpr {
			return m.StreamingPayload
		}},
		{"StreamingResult", StreamingResult, func(m *expr.MethodExpr) *expr.AttributeExpr {
			return m.StreamingResult
		}},
	} {
		for _, kind := range []string{"direct", "aliased", "result-type"} {
			for _, extended := range []bool{false, true} {
				for _, plainFirst := range []bool{false, true} {
					name := slot.name + "/" + kind
					if extended {
						name += "/extended"
					} else {
						name += "/description"
					}
					if plainFirst {
						name += "/plain-first"
					} else {
						name += "/custom-first"
					}
					t.Run(name, func(t *testing.T) {
						root := expr.RunDSL(t, func() {
							base := Type("Base", func() {
								Attribute("left", String)
								Required("left")
							})
							var definition any = func() {
								Attribute("right", String)
							}
							if kind == "aliased" {
								definition = Type("Inner", Type("Leaf", definition))
							}
							var value expr.UserType
							if kind == "result-type" {
								value = ResultType("application/vnd.value", "Value", func() {
									Attributes(definition.(func()))
								})
							} else {
								value = Type("Value", definition)
							}
							Service("fields", func() {
								plain := func() {
									Method("plain", func() {
										Payload(value)
										Result(value)
									})
								}
								if plainFirst {
									plain()
								}
								Method("custom", func() {
									slot.define(value, func() {
										Description("custom description")
										if extended {
											Extend(base)
										}
									})
								})
								if !plainFirst {
									plain()
								}
							})
						})
						method := root.Service("fields").Method("custom")
						custom := slot.attribute(method)
						original := root.Service("fields").Method("plain").Payload
						require.Nil(t, original.Find("left"), "customization must not widen plain uses")
						require.Equal(t, "Value", original.Type.Name())
						require.Equal(t, "custom description", custom.Description)
						if !extended {
							require.Equal(t, "Value", custom.Type.Name())
							require.Nil(t, custom.Find("left"))
							return
						}
						if want := "Value_custom_" + slot.name; custom.Type.Name() != want {
							t.Errorf("custom type name = %q, want %q", custom.Type.Name(), want)
						}
						require.NotNil(t, custom.Find("left"))
						require.NotNil(t, custom.Find("right"))
						require.True(t, custom.IsRequired("left"))
						require.True(t, custom.Type.(expr.UserType).Attribute().IsRequired("left"), "the emitted type must own inherited requiredness")
						if result, ok := custom.Type.(*expr.ResultTypeExpr); ok {
							view := result.View(expr.DefaultView)
							require.NotNil(t, expr.AsObject(view.Type).Attribute("left"), "the implicit default view must include inherited fields")
							require.True(t, view.IsRequired("left"))
						}
					})
				}
			}
		}
	}
}

// TestExtendedEmptyMethodTypesPreserveSentinel covers the shared Empty object,
// which expr.Dup intentionally retains instead of copying.
func TestExtendedEmptyMethodTypesPreserveSentinel(t *testing.T) {
	for _, aliased := range []bool{false, true} {
		t.Run(map[bool]string{false: "direct", true: "aliased"}[aliased], func(t *testing.T) {
			root := expr.RunDSL(t, func() {
				base := Type("Base", func() {
					Attribute("left", String)
					Required("left")
				})
				var value expr.UserType = Empty
				if aliased {
					value = Type("Value", Empty)
				}
				Service("fields", func() {
					Method("extended", func() {
						Payload(value, func() {
							Extend(base)
						})
					})
				})
			})
			require.Equal(t, "Empty", expr.Empty.Name())
			require.Empty(t, *expr.AsObject(expr.Empty))
			payload := root.Service("fields").Method("extended").Payload
			require.NotSame(t, expr.Empty, payload.Type)
			require.NotNil(t, payload.Find("left"))
			require.True(t, payload.IsRequired("left"))
		})
	}
}

func TestExtendedResultViewSelection(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		name := "implicit"
		if explicit {
			name = "explicit"
		}
		t.Run(name, func(t *testing.T) {
			root := expr.RunDSL(t, func() {
				base := Type("Base", func() {
					Attribute("left", String)
					Required("left")
				})
				value := ResultType("application/vnd.value", "Value", func() {
					Attributes(func() {
						Attribute("right", String)
					})
					if explicit {
						View("default", func() {
							Attribute("right")
						})
					}
				})
				Service("fields", func() {
					Method("plain", func() {
						Result(value)
					})
					Method("custom", func() {
						Result(value, func() {
							Extend(base)
							Required("right")
						})
					})
				})
			})
			custom := root.Service("fields").Method("custom").Result.Type.(*expr.ResultTypeExpr)
			projected, err := expr.Project(custom, expr.DefaultView)
			require.NoError(t, err)
			left := expr.AsObject(projected.Type).Attribute("left")
			if explicit {
				require.Nil(t, left, "explicit field selection must remain authoritative")
				require.False(t, projected.IsRequired("left"))
			} else {
				require.NotNil(t, left)
				require.True(t, projected.IsRequired("left"))
			}
			require.True(t, projected.IsRequired("right"), "method requirements must survive projection")
			original := root.Service("fields").Method("plain").Result
			require.Nil(t, original.Find("left"))
			require.False(t, original.IsRequired("right"))
		})
	}
}
