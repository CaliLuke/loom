package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

var GRPCEndpointWithReferenceTypes = func() {
	var EntityReference = Type("EntityReference", func() {
		Field(1, "name", String)
	})

	var Entity = Type("Entity", func() {
		Reference(EntityReference)
		Field(1, "id", String)
		Field(2, "name")
	})

	Service("Service", func() {
		Method("Method", func() {
			Payload(Entity)
			GRPC(func() {})
		})
	})
}
