package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

var NoMethodClientInterceptorDSL = func() {
	Interceptor("tracing")
	Service("NoMethodClientInterceptor", func() {
		ClientInterceptor("tracing")
	})
}
