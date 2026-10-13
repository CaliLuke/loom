package service

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/testdata"
	"github.com/CaliLuke/loom/dsl"
)

func TestErrorConstructorIdentity(t *testing.T) {
	for _, pair := range [][2]string{
		{"foo_bar", "FooBar"}, {"get", "Get"}, {"a_b", "aB"},
		{"x1", "x_1"}, {"foo", "foo_"}, {"user", "User"},
	} {
		t.Run(pair[0]+"/"+pair[1], func(t *testing.T) {
			root := codegen.RunDSL(t, func() {
				dsl.Service("errors", func() {
					for _, name := range pair {
						dsl.Error(name)
					}
					for _, method := range []string{"one", "two"} {
						dsl.Method(method, func() {
							for _, name := range pair {
								dsl.Error(name)
							}
						})
					}
				})
			})
			data := NewServicesData(root).Get("errors")
			require.Len(t, data.errorInits, 2)
			if data.errorInits[0].Name == data.errorInits[1].Name {
				t.Errorf("distinct wire errors %q and %q share constructor %s", pair[0], pair[1], data.errorInits[0].Name)
			}
			byWire := make(map[string]string)
			for _, init := range data.errorInits {
				byWire[init.ErrName] = init.Name
			}
			for _, method := range data.Methods {
				for _, init := range method.Errors {
					require.Equal(t, byWire[init.ErrName], init.Name, "method %s must reference the emitted factory", method.Name)
				}
			}
		})
	}
}

func TestErrorConstructorExternalAndCustom(t *testing.T) {
	root := codegen.RunDSL(t, testdata.ErrorConstructorExternalDSL)
	data := NewServicesData(root).Get("errors")
	require.Len(t, data.errorInits, 1)
	require.Equal(t, "MakeFooBar", data.errorInits[0].Name)
	method := data.Method("show")
	require.Equal(t, "*domain.MakeFooBar", method.PayloadRef)
	require.Len(t, method.Errors, 2)
	for _, er := range method.Errors {
		if er.ErrName == "custom" {
			require.Empty(t, er.Name, "a custom error has no generated factory")
		} else {
			require.Equal(t, data.errorInits[0].Name, er.Name)
		}
	}
}

func TestErrorConstructorDoesNotReserveAbsentFactory(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		custom := dsl.Type("Failure", func() {
			dsl.Attribute("message", dsl.String)
		})
		payload := dsl.Type("MakeCustom", func() {
			dsl.Attribute("value", dsl.String)
		})
		dsl.Service("errors", func() {
			dsl.Error("custom", custom)
			dsl.Method("show", func() {
				dsl.Payload(payload)
			})
		})
	})
	data := NewServicesData(root).Get("errors")
	require.Empty(t, data.errorInits)
	require.Equal(t, "MakeCustom", data.Method("show").Payload)
}

func TestErrorConstructorSources(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		dsl.API("errors", func() {
			dsl.Error("api_failure")
		})
		dsl.Service("errors", func() {
			dsl.Error("api_failure")
			dsl.Error("service_failure")
			for _, name := range []string{"one", "two"} {
				dsl.Method(name, func() {
					dsl.Error("method_failure")
				})
			}
		})
	})
	data := NewServicesData(root).Get("errors")
	require.Len(t, data.errorInits, 3)
	factories := map[string]string{
		"api_failure":     "MakeAPIFailure",
		"service_failure": "MakeServiceFailure",
		"method_failure":  "MakeMethodFailure",
	}
	for _, er := range data.errorInits {
		require.Equal(t, factories[er.ErrName], er.Name)
	}
	for _, method := range data.Methods {
		require.Len(t, method.Errors, 3)
		for _, er := range method.Errors {
			require.Equal(t, factories[er.ErrName], er.Name)
		}
	}
}

func TestErrorConstructorSharesPackageScope(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		payload := dsl.Type("MakeFailure", func() {
			dsl.Attribute("value", dsl.String)
		})
		dsl.Service("errors", func() {
			dsl.Error("failure")
			dsl.Method("one", func() {
				dsl.Payload(payload)
				dsl.Error("failure")
			})
		})
	})
	data := NewServicesData(root).Get("errors")
	require.Len(t, data.errorInits, 1)
	require.NotEmpty(t, data.Method("one").Payload)
	require.NotEqual(t, data.errorInits[0].Name, data.Method("one").Payload, "error factory and nominal type share the service package")
}
