package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

var EndpointWithClientInterceptorDSL = func() {
	Interceptor("tracing")
	Service("ServiceWithClientInterceptor", func() {
		Method("Method", func() {
			ClientInterceptor("tracing")
			Payload(String)
			Result(String)
			HTTP(func() {
				POST("/")
			})
		})
	})
}
