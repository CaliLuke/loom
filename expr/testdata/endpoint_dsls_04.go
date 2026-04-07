package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

var GRPCEndpointWithInheritErrorDSL = func() {
	API("API", func() {
		Error("not_found")
		GRPC(func() {
			Response("not_found", CodeNotFound)
		})
	})
	Service("Service", func() {
		Error("not_found")
		Method("Method", func() {
			GRPC(func() {
				Response(CodeOK)
			})
		})
	})
}
