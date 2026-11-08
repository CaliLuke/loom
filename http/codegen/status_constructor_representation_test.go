package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

func TestResponseStatusNativeReplacement(t *testing.T) {
	for _, viewed := range []bool{false, true} {
		name := "native"
		if viewed {
			name = "viewed"
		}
		t.Run(name, func(t *testing.T) {
			root := RunHTTPDSL(t, func() {
				fields := func() {
					Attribute("status", Int, func() { Meta("struct:field:type", "int64") })
					Attribute("message", String)
					Required("status", "message")
				}
				var value any
				if viewed {
					value = ResultType("application/vnd.replacement", func() {
						fields()
						View("default", func() { Attribute("status"); Attribute("message") })
					})
				} else {
					value = Type("Value", fields)
				}
				Service("replacement", func() {
					Method("read", func() {
						Result(value)
						HTTP(func() { GET("/"); Response(StatusOK, func() { StatusField("status"); Body("message") }) })
					})
				})
			})
			data := CreateHTTPServices(root).Get("replacement")
			response := data.Endpoint("read").Result.Responses[0]
			args := response.ResultInit.ClientArgs
			require.Equal(t, "int", args[len(args)-1].TypeRef)
			body := fieldCode(response.ResultInit, "client", data.Service.Scope).Code
			require.Contains(t, body, "int64(statusCode)")
			if viewed {
				require.Contains(t, body, ".Status = &tmpstatusCode")
			}
		})
	}
}
