package testdata

import (
	"slices"

	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// InterceptorAccessorCoverageDSL applies one interceptor to independently
// selected server and client methods with distinct, compatible value types.
func InterceptorAccessorCoverageDSL(server, client []string, streaming bool) func() {
	return func() {
		audit := dsl.Interceptor("audit", func() {
			dsl.ReadPayload(accessorCoverageField)
			dsl.WritePayload(accessorCoverageField)
			if streaming {
				dsl.ReadStreamingPayload(accessorCoverageField)
				dsl.WriteStreamingPayload(accessorCoverageField)
				dsl.ReadStreamingResult(accessorCoverageField)
				dsl.WriteStreamingResult(accessorCoverageField)
			} else {
				dsl.ReadResult(accessorCoverageField)
				dsl.WriteResult(accessorCoverageField)
			}
		})
		dsl.Service("coverage", func() {
			for _, name := range []string{"a", "b", "c"} {
				dsl.Method(name, func() {
					dsl.Payload(accessorCoverageValue)
					if streaming {
						dsl.StreamingPayload(accessorCoverageValue)
						dsl.StreamingResult(accessorCoverageValue)
					} else {
						dsl.Result(accessorCoverageValue)
					}
					if slices.Contains(server, name) {
						dsl.ServerInterceptor(audit)
					}
					if slices.Contains(client, name) {
						dsl.ClientInterceptor(audit)
					}
					dsl.HTTP(func() {
						if streaming {
							dsl.GET("/" + name)
							dsl.Param("value")
						} else {
							dsl.POST("/" + name)
						}
					})
				})
			}
		})
	}
}

func accessorCoverageField() {
	dsl.Attribute("value")
}

func accessorCoverageValue() {
	dsl.Attribute("value", dsl.String)
	dsl.Required("value")
}

// InterceptorAccessorAliasDSL places the payloads and their accessed field type
// in a package that collides with the standard context import.
func InterceptorAccessorAliasDSL() {
	value := dsl.Type("Value", func() {
		dsl.Meta("struct:pkg:path", "types/context")
		dsl.Attribute("text", dsl.String)
		dsl.Required("text")
	})
	var payloads [3]expr.UserType
	for index, name := range []string{"AInput", "BInput", "CInput"} {
		payloads[index] = dsl.Type(name, func() {
			dsl.Meta("struct:pkg:path", "types/context")
			dsl.Attribute("value", value)
			dsl.Required("value")
		})
	}
	audit := dsl.Interceptor("audit", func() {
		dsl.ReadPayload(accessorCoverageField)
		dsl.WritePayload(accessorCoverageField)
	})
	dsl.Service("coverage", func() {
		for index, name := range []string{"a", "b", "c"} {
			dsl.Method(name, func() {
				dsl.Payload(payloads[index])
				dsl.Result(dsl.String)
				if name == "a" {
					dsl.ServerInterceptor(audit)
				}
				if name == "c" {
					dsl.ClientInterceptor(audit)
				}
				dsl.HTTP(func() {
					dsl.POST("/" + name)
				})
			})
		}
	})
}
