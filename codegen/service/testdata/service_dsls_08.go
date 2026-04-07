package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

var PkgPathPayloadAttributeDSL = func() {
	var Foo = Type("Foo", func() {
		Attribute("IntField", Int)
		Meta("struct:pkg:path", "foo")
	})
	var Bar = Type("Bar", func() {
		Attribute("Foo", Foo)
	})

	Service("PkgPathPayloadAttributeDSL", func() {
		Method("Foo", func() {
			Payload(Bar)
			Result(Bar)
		})
	})
}
