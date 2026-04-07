package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

var InterceptorsDSL = func() {
	var LogInterceptor = Interceptor("Log", func() {
		Description("Logs request and response details")
	})
	var MetricsInterceptor = Interceptor("Metrics", func() {
		Description("Collects metrics for the operation")
	})

	API("TestAPI", func() {
		Title("Test API")
		Server("Test", func() {
			Description("Test server")
		})
	})

	Service("ServiceWithInterceptors", func() {
		ClientInterceptor(LogInterceptor)
		Method("MethodA", func() {
			ClientInterceptor(MetricsInterceptor)
			Payload(String)
			Result(String)
			GRPC(func() {})
		})
		Method("MethodB", func() {
			Payload(Int)
			Result(Int)
			GRPC(func() {})
		})
	})
}
