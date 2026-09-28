package testdata

import . "github.com/CaliLuke/loom/dsl"

// TypeIdentityDSL reuses a result type in request, success, collection, and
// error bodies, which retain its design identifier but have distinct Go types.
// Its collision services give endpoint bodies and nested types the same name.
var TypeIdentityDSL = func() {
	API("identity", func() {})
	leaf := Type("Leaf", func() {
		Attribute("name", String)
		Required("name")
	})
	other := Type("Other", func() {
		Attribute("count", Int)
		Required("count")
	})
	uidCollision := Type("unioncollision#Other", func() {
		Attribute("value", String)
		Required("value")
	})
	choice := Type("Choice", OneOf(leaf, other, uidCollision))
	result := ResultType("application/vnd.identity", "Record", func() {
		Attribute("id", String)
		Attribute("optional", choice)
		Attribute("required", choice)
		Required("id", "required")
	})
	Service("probe", func() {
		Method("put", func() {
			Payload(ArrayOf(result))
			Result(result)
			Error("bad", result)
			HTTP(func() {
				POST("/put")
				Response("bad", StatusBadRequest)
			})
		})
		Method("puts", func() {
			Payload(ArrayOf(result))
			Result(CollectionOf(result))
			Error("bad", result)
			HTTP(func() {
				POST("/puts")
				Response("bad", StatusBadRequest)
			})
		})
	})
	envelope := Type("Envelope", func() {
		Attribute("other", other)
		Required("other")
	})
	Service("bodycollision", func() {
		Method("other", func() {
			Payload(envelope)
			Result(envelope)
			HTTP(func() {
				POST("/object")
			})
		})
	})
	Service("unioncollision", func() {
		Method("other", func() {
			Payload(choice)
			Result(choice)
			HTTP(func() {
				POST("/union")
			})
		})
	})
	Service("streamcollision", func() {
		Method("other", func() {
			StreamingPayload(other)
			StreamingResult(envelope)
			HTTP(func() {
				GET("/stream")
			})
		})
	})
}
