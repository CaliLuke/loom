package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestResponseConstructorUsesPreparedBodyLayout(t *testing.T) {
	for _, tc := range []struct {
		name                       string
		mapping, selected, failure bool
	}{
		{name: "success-array"},
		{name: "success-map", mapping: true},
		{name: "selected-array", selected: true},
		{name: "selected-map", mapping: true, selected: true},
		{name: "error-whole-array", failure: true},
		{name: "error-whole-map", mapping: true, failure: true},
		{name: "error-array", selected: true, failure: true},
		{name: "error-map", mapping: true, selected: true, failure: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := RunHTTPDSL(t, responseBodyContextDSL(tc.mapping, tc.selected, tc.failure))
			endpoint := CreateHTTPServices(root).Get("layouts").Endpoint("read")
			var response *ResponseData
			if tc.failure {
				response = endpoint.Errors[0].Errors[0].Response
			} else {
				response = endpoint.Result.Responses[0]
			}
			require.NotNil(t, response.ClientBody)
			require.NotNil(t, response.ResultInit)
			require.NotEmpty(t, response.ResultInit.ClientArgs)
			require.Contains(t, response.ClientBody.ValueRef, "loom.Optional[")
			require.Equal(t, response.ClientBody.ValueRef, response.ResultInit.ClientArgs[0].TypeRef)
			require.Contains(t, response.ResultInit.ClientCode, ".Value()")
		})
	}
}

func TestResponseBodylessCollectionUsesBindingLayout(t *testing.T) {
	root := RunHTTPDSL(t, responseBodylessCollectionDSL)
	response := CreateHTTPServices(root).Get("headers").Endpoint("read").Result.Responses[0]
	require.Nil(t, response.ClientBody)
	require.NotNil(t, response.ResultInit)
	require.Len(t, response.Headers, 1)
	require.Equal(t, response.Headers[0].TypeRef, response.ResultInit.ClientArgs[0].TypeRef)
	require.NotContains(t, response.ResultInit.ClientCode, ".Value()")
	require.NotContains(t, response.ResultInit.ClientCode, "loom.Optional[")
}

func TestResponseBindingPointerAndDefaultStorage(t *testing.T) {
	for _, location := range []string{"header", "cookie"} {
		for _, mode := range []string{"optional", "required", "default"} {
			t.Run(location+"/"+mode, func(t *testing.T) {
				root := RunHTTPDSL(t, func() {
					Service("bindings", func() {
						Method("read", func() {
							Result(func() {
								Attribute("value", String, func() {
									if mode == "default" {
										Default("fallback")
									}
								})
								if mode == "required" {
									Required("value")
								}
							})
							HTTP(func() {
								GET("/read")
								Response(StatusOK, func() {
									if location == "header" {
										Header("value:X-Value")
									} else {
										Cookie("value:value")
									}
								})
							})
						})
					})
				})
				services := CreateHTTPServices(root)
				service := services.Get("bindings")
				response := service.Endpoint("read").Result.Responses[0]
				require.Nil(t, response.ClientBody)
				require.Len(t, response.ResultInit.ClientArgs, 1)
				argument := response.ResultInit.ClientArgs[0]
				require.Equal(t, mode == "optional", argument.FieldPointer)
				if mode == "default" {
					files := ClientFiles("", services)
					require.Contains(t, codegen.SectionCode(t, files[1].AllSections()[2]), `"fallback"`)
				}
			})
		}
	}
}

func responseBodyContextDSL(mapping, selected, failure bool) func() {
	return func() {
		item := Type("Item", func() {
			Attribute("name", String)
			Required("name")
		})
		var collection expr.DataType = ArrayOf(item)
		if mapping {
			collection = MapOf(String, item)
		}
		fields := func() {
			Attribute("values", collection)
		}
		Service("layouts", func() {
			Method("read", func() {
				switch {
				case failure && !selected:
					Error("oops", collection)
				case failure:
					Error("oops", fields)
				case selected:
					Result(fields)
				default:
					Result(collection)
				}
				HTTP(func() {
					GET("/values")
					body := func() {
						if selected {
							Body("values")
						}
					}
					if failure {
						Response("oops", StatusBadRequest, body)
					} else {
						Response(StatusOK, body)
					}
				})
			})
		})
	}
}

func responseBodylessCollectionDSL() {
	label := Type("Label", String)
	Service("headers", func() {
		Method("read", func() {
			Result(ArrayOf(label))
			HTTP(func() {
				GET("/values")
				Response(StatusOK, func() {
					Header("values")
				})
			})
		})
	})
}
