package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

var RawObjectPayloadTypeNameCollisionDSL = func() {
	var FooPayload = Type("FooPayload", func() {
		Attribute("x", String)
	})
	var FooPayload2 = Type("FooPayload2", func() {
		Attribute("y", String)
	})

	Service("RawObjectPayloadTypeNameCollision", func() {
		// Reserve FooPayload and FooPayload2 in the NameScope *before* raw-object
		// payload wrapping occurs (wrapObject runs before method data is built).
		Error("reserve_foo_payload", FooPayload)
		Error("reserve_foo_payload2", FooPayload2)

		Method("Foo", func() {
			Payload(func() {
				Attribute("a", String)
				Required("a")
			})
		})
	})
}

// UnionCustomKeysDSL tests union with custom type and value keys via Meta tags.
