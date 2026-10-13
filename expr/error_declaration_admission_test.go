package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

func TestErrorDeclarationNamesRejectLocalDuplicates(t *testing.T) {
	for _, scope := range []string{"api", "service", "method"} {
		for _, different := range []bool{false, true} {
			name := scope + "/same-type"
			if different {
				name = scope + "/different-types"
			}
			t.Run(name, func(t *testing.T) {
				expr.SetupTestDSL(t)
				declare := func() {
					Error("boom", String)
					if different {
						Error("boom", Int)
					} else {
						Error("boom", String)
					}
				}
				require.True(t, eval.Execute(func() {
					if scope == "api" {
						API("duplicate", declare)
					}
					Service("svc", func() {
						if scope == "service" {
							declare()
						}
						Method("call", func() {
							if scope == "method" {
								declare()
							}
						})
					})
				}, nil))
				err := eval.RunDSL()
				require.Error(t, err)
				require.Contains(t, err.Error(), `error "boom" defined twice`)
				prefix := "design:"
				if scope == "service" {
					prefix = `service "svc":`
				} else if scope == "method" {
					prefix = `service "svc" method "call":`
				}
				require.Contains(t, err.Error(), prefix)
			})
		}
	}
}

func TestErrorDeclarationNamesPermitScopeReuse(t *testing.T) {
	root := expr.RunDSL(t, func() {
		API("reuse", func() {
			Error("boom", Boolean)
		})
		for _, name := range []string{"first", "second"} {
			Service(name, func() {
				Error("boom", String)
				Error("foo-bar")
				Error("foo_bar")
				Error("BOOM")
				Method("inherited", func() {
				})
				Method("overridden", func() {
					Error("boom", Int)
					Error("local")
				})
			})
		}
	})
	require.Equal(t, expr.Boolean, root.Error("boom").Type.(expr.UserType).Attribute().Type)
	for _, s := range root.Services {
		require.Equal(t, expr.String, s.Error("boom").Type.(expr.UserType).Attribute().Type)
		for _, m := range s.Methods {
			want := expr.String
			wantNames := []string{"boom", "foo-bar", "foo_bar", "BOOM"}
			if m.Name == "overridden" {
				want = expr.Int
				wantNames = []string{"boom", "local", "foo-bar", "foo_bar", "BOOM"}
			}
			names := make([]string, 0, len(m.Errors))
			for _, e := range m.Errors {
				names = append(names, e.Name)
			}
			require.Equal(t, wantNames, names, "method declarations precede inherited service errors")
			require.Equal(t, want, m.Error("boom").Type.(expr.UserType).Attribute().Type)
		}
	}
}

func TestErrorDeclarationNamesRejectUnusedAndShadowed(t *testing.T) {
	cases := map[string]func(){
		"unused-api": func() {
			API("unused", func() {
				Error("boom")
				Error("boom")
			})
		},
		"unused-service": func() {
			Service("unused", func() {
				Error("boom")
				Error("boom")
			})
		},
		"shadowed-service": func() {
			Service("shadowed", func() {
				Error("boom", String)
				Error("boom", Int)
				Method("call", func() {
					Error("boom", Boolean)
				})
			})
		},
		"mapped-method": func() {
			Service("mapped", func() {
				Method("call", func() {
					Result(String)
					Error("boom")
					Error("boom")
					HTTP(func() {
						GET("/call")
						Response("boom", StatusBadRequest)
					})
				})
			})
		},
	}
	for name, design := range cases {
		t.Run(name, func(t *testing.T) {
			expr.SetupTestDSL(t)
			require.True(t, eval.Execute(design, nil))
			err := eval.RunDSL()
			require.Error(t, err)
			require.Contains(t, err.Error(), `error "boom" defined twice`)
		})
	}
}

func TestErrorDeclarationNamesValidateAuthoredLists(t *testing.T) {
	for _, scope := range []string{"api", "service", "method"} {
		t.Run(scope, func(t *testing.T) {
			root := expr.SetupTestDSL(t)
			errors := make([]*expr.ErrorExpr, 0, 3)
			for range 3 {
				errors = append(errors, &expr.ErrorExpr{
					Name:          "boom",
					AttributeExpr: &expr.AttributeExpr{Type: expr.String},
				})
			}
			var err error
			switch scope {
			case "api":
				root.Errors = errors
				err = root.Validate()
			case "service":
				s := &expr.ServiceExpr{Name: "direct", Errors: errors}
				err = s.Validate()
			case "method":
				m := &expr.MethodExpr{Name: "direct", Errors: errors, Service: &expr.ServiceExpr{Name: "service"}}
				m.Prepare()
				err = m.Validate()
			}
			require.Error(t, err)
			var validation *eval.ValidationErrors
			require.ErrorAs(t, err, &validation)
			require.Len(t, validation.Errors, 2, "each declaration after the first must be diagnosed")
			require.Contains(t, err.Error(), `error "boom" defined twice`)
			require.Len(t, errors, 3, "admission must not delete authored declarations")
		})
	}
}
