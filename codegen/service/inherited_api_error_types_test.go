package service

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	httpdata "github.com/CaliLuke/loom/http/codegen/testdata"
)

// TestInheritedAPIErrorTypes declares the custom error types referenced by
// inherited API mappings in the generated service package.
func TestInheritedAPIErrorTypes(t *testing.T) {
	for _, tc := range []struct {
		name   string
		design func()
		types  []string
	}{
		{"headers", httpdata.APINoBodyErrorResponseDSL, []string{"StringError"}},
		{"content type", httpdata.APINoBodyErrorResponseWithContentTypeDSL, []string{"StringError"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := codegen.RunDSL(t, tc.design)
			source := renderServiceFile(t, root, NewServicesData(root))
			for _, name := range tc.types {
				require.Contains(t, source, "type "+name+" struct")
			}
		})
	}
}

// TestInheritedAPIErrorScope keeps collection limited to evaluated mappings.
func TestInheritedAPIErrorScope(t *testing.T) {
	for _, override := range []bool{false, true} {
		t.Run(fmt.Sprint(override), func(t *testing.T) {
			root := codegen.RunDSL(t, func() {
				apiFailure := dsl.Type("APIFailure", func() {
					dsl.Attribute("message", dsl.String)
				})
				localFailure := dsl.Type("LocalFailure", func() {
					dsl.Attribute("message", dsl.String)
				})
				unused := dsl.Type("UnusedFailure", func() {
					dsl.Attribute("detail", dsl.String)
				})
				dsl.API("test", func() {
					dsl.Error("failure", apiFailure)
					dsl.Error("unused", unused)
					dsl.HTTP(func() {
						dsl.Response("failure", dsl.StatusBadRequest)
						dsl.Response("unused", dsl.StatusInternalServerError)
					})
				})
				dsl.Service("used", func() {
					if override {
						dsl.Error("failure", localFailure)
						dsl.HTTP(func() {
							dsl.Response("failure", dsl.StatusConflict)
						})
					} else {
						dsl.Error("failure")
					}
					dsl.Method("one", func() {
						dsl.HTTP(func() {
							dsl.GET("/one")
						})
					})
					dsl.Method("two", func() {
						dsl.HTTP(func() {
							dsl.GET("/two")
						})
					})
				})
				dsl.Service("unrelated", func() {
					dsl.Method("show", func() {
						dsl.HTTP(func() {
							dsl.GET("/unrelated")
						})
					})
				})
			})
			services := NewServicesData(root)
			data := services.Get("used")
			names := make([]string, len(data.errorTypes))
			for i, errorType := range data.errorTypes {
				names[i] = errorType.Name
			}
			require.NotContains(t, names, "UnusedFailure")
			if override {
				require.Equal(t, []string{"LocalFailure"}, names)
			} else {
				require.Contains(t, names, "APIFailure")
				require.Len(t, names, 2)
				require.Len(t, data.errorInits, 1)
				require.Equal(t, "MakeFailure", data.errorInits[0].Name)
			}
			require.Empty(t, services.Get("unrelated").errorTypes)
		})
	}
}

// TestInheritedAPIErrorOwnPackage rejects an inherited type placed in its
// service package, matching the rule for directly declared service types.
func TestInheritedAPIErrorOwnPackage(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		failure := dsl.Type("Failure", func() {
			dsl.Attribute("message", dsl.String)
			dsl.Meta("struct:pkg:path", "shared")
		})
		dsl.API("test", func() {
			dsl.Error("failure", failure)
			dsl.HTTP(func() {
				dsl.Response("failure", dsl.StatusBadRequest)
			})
		})
		dsl.Service("shared", func() {
			dsl.Error("failure")
			dsl.Method("show", func() {
				dsl.HTTP(func() {
					dsl.GET("/")
				})
			})
		})
	})
	data := NewServicesData(root).Get("shared")
	require.ErrorContains(t, checkUserTypePackages(data), `type "Failure" has struct:pkg:path "shared"`)
	require.ErrorContains(t, checkUserTypePackages(data), "the service package cannot import itself")
}

// TestInheritedAPIErrorDescriptor uses the mapped type even when a method
// declares an error of the same name in another package.
func TestInheritedAPIErrorDescriptor(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		apiError := dsl.Type("APIFailure", func() {
			dsl.Attribute("message", dsl.String)
			dsl.Meta("struct:pkg:path", "apierrors")
		})
		localError := dsl.Type("LocalFailure", func() {
			dsl.Attribute("message", dsl.String)
			dsl.Meta("struct:pkg:path", "localerrors")
		})
		dsl.API("test", func() {
			dsl.Error("failure", apiError)
			dsl.HTTP(func() {
				dsl.Response("failure", dsl.StatusBadRequest)
			})
		})
		dsl.Service("shared", func() {
			dsl.Method("show", func() {
				dsl.Error("failure", localError)
				dsl.HTTP(func() {
					dsl.GET("/")
				})
			})
		})
	})
	data := NewServicesData(root).Get("shared")
	desc := BuildErrorDescriptor(data, "failure", root.Errors[0].AttributeExpr)
	require.Equal(t, "*apierrors.APIFailure", desc.Type.Ref)
}
