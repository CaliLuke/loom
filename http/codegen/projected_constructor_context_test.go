package codegen

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestProjectedConstructorTargetContext(t *testing.T) {
	for _, location := range []string{"header", "cookie"} {
		for _, viewed := range []bool{false, true} {
			for _, required := range []bool{false, true} {
				name := location
				if viewed {
					name += "/view"
				} else {
					name += "/service"
				}
				if required {
					name += "/required"
				} else {
					name += "/optional"
				}
				t.Run(name, func(t *testing.T) {
					root := RunHTTPDSL(t, projectedConstructorContextDSL(location, viewed, required))
					data := CreateHTTPServices(root).Get("mappedview")
					init := data.Endpoints[0].Result.Responses[0].ResultInit
					want := data.Service.PkgName
					if viewed {
						want = data.Service.ViewsPkg
					}
					require.Equal(t, want, init.ReturnTypePkg, "field casts must use the selected result declaration package")
					require.Contains(t, init.ReturnTypeRef, want+".")
					code := fieldCode(init, "client", data.Service.Scope).Code
					if viewed {
						require.Contains(t, code, want+".LabelView(")
						require.NotContains(t, code, data.Service.PkgName+".LabelView(")
					} else {
						require.Contains(t, code, want+".Label(")
					}
				})
			}
		}
	}
}

func TestProjectedConstructorTargetCompiles(t *testing.T) {
	for _, location := range []string{"header", "cookie"} {
		for _, viewed := range []bool{false, true} {
			for _, required := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/view=%t/required=%t", location, viewed, required), func(t *testing.T) {
					root := RunHTTPDSL(t, projectedConstructorContextDSL(location, viewed, required))
					data := CreateHTTPServices(root).Get("mappedview")
					init := data.Endpoints[0].Result.Responses[0].ResultInit
					arg := init.ClientArgs[0]
					dir := t.TempDir()
					renderHTTPModule(t, dir, "example.com/projectedcontext", root)
					input := "value"
					if arg.Pointer {
						input = "&value"
					}
					field := "result.Value"
					if arg.FieldPointer {
						field = "*result.Value"
					}
					absent := ""
					if !required {
						absent = fmt.Sprintf(`if %s(nil).Value != nil {
        t.Fatal("absent value became supplied")
    }`, init.Name)
					}
					runtime := fmt.Sprintf(`package client
import "testing"
func TestMappedConstructor(t *testing.T) {
    for _, value := range []string{"", "wire-value"} {
        result := %s(%s)
        if string(%s) != value {
            t.Fatalf("field mismatch: %%v", result.Value)
        }
    }
    %s
}
`, init.Name, input, field, absent)
					require.NoError(t, os.WriteFile(filepath.Join(dir, "gen/http/mappedview/client/constructor_test.go"), []byte(runtime), 0o600))
					runGoCommand(t, dir, "mod", "tidy")
					runGoCommand(t, dir, "test", "./...")
				})
			}
		}
	}
	t.Run("nominal-collision", func(t *testing.T) {
		root := RunHTTPDSL(t, projectedConstructorContextDSL("header", true, false, true))
		dir := t.TempDir()
		renderHTTPModule(t, dir, "example.com/projectedcontext", root)
		runGoCommand(t, dir, "mod", "tidy")
		runGoCommand(t, dir, "test", "./...")
	})
}

func projectedConstructorContextDSL(location string, viewed, required bool, collision ...bool) func() {
	return func() {
		var shadow expr.UserType
		labelName := "Label"
		if len(collision) > 0 && collision[0] {
			shadow = dsl.Type("Label", dsl.String)
			labelName = "label"
		}
		label := dsl.Type(labelName, dsl.String)
		fields := func() {
			if shadow != nil {
				dsl.Attribute("reserved", shadow)
			}
			dsl.Attribute("value", label)
			if required {
				dsl.Required("value")
			}
		}
		var result any
		if viewed {
			result = dsl.ResultType("application/vnd.record", func() {
				fields()
				dsl.View("default", func() {
					if shadow != nil {
						dsl.Attribute("reserved")
					}
					dsl.Attribute("value")
				})
			})
		} else {
			result = dsl.Type("Record", fields)
		}
		dsl.Service("mappedview", func() {
			dsl.Method("read", func() {
				dsl.Result(result)
				dsl.HTTP(func() {
					dsl.GET("/read")
					dsl.Response(dsl.StatusOK, func() {
						if location == "header" {
							dsl.Header("value:X-Value")
						} else {
							dsl.Cookie("value:value")
						}
					})
				})
			})
		})
	}
}

func TestProjectedConstructorNominalScope(t *testing.T) {
	root := RunHTTPDSL(t, projectedConstructorContextDSL("header", true, false, true))
	data := CreateHTTPServices(root).Get("mappedview")
	init := data.Endpoints[0].Result.Responses[0].ResultInit
	var field *expr.AttributeExpr
	for _, arg := range init.ClientArgs {
		if arg.FieldName == "Value" {
			field = &expr.AttributeExpr{Type: arg.FieldType}
		}
	}
	require.NotNil(t, field)
	target := data.Service.ViewScope.GoFullTypeRef(field, data.Service.ViewsPkg)
	wrong := data.Service.Scope.GoFullTypeRef(field, data.Service.ViewsPkg)
	require.NotEqual(t, target, wrong, "control must allocate different nominal identities")
	// Repairing only the package still uses the wrong nominal declaration.
	copy := *init
	copy.ReturnTypePkg = data.Service.ViewsPkg
	require.Contains(t, fieldCode(&copy, "client", data.Service.Scope).Code, target+"(")
}
