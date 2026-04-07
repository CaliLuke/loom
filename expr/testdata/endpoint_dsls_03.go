package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

var GRPCEndpointWithExtendedTypes = func() {
	var EntityExtended = Type("EntityExtended", func() {
		Field(1, "name", String)
		Field(2, "id", String)
	})

	var Entity = Type("Entity", func() {
		Extend(EntityExtended)
	})

	Service("Service", func() {
		Method("Method", func() {
			Payload(Entity)
			Result(func() {
				Extend(Entity)
			})
			GRPC(func() {
				Metadata(func() {
					Attribute("name")
				})
				Message(func() {
					Attribute("id")
				})
				Response(func() {
					Headers(func() {
						Attribute("name")
					})
					Message(func() {
						Attribute("id")
					})
				})
			})
		})
	})
}
