package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

var GRPCEndpointWithUnionContainingAny = func() {
	var AnyAlias = Type("AnyAlias", func() {
		Attribute("value", Any)
	})

	var U = Type("U", func() {
		OneOf("choice", func() {
			Field(1, "plain_any", Any)
			Field(2, "array_any", ArrayOf(AnyAlias))
			Field(3, "map_any", MapOf(String, AnyAlias))
		})
	})

	Service("Service", func() {
		Method("MethodUnion", func() {
			Payload(U)
			GRPC(func() {})
		})
	})
}
