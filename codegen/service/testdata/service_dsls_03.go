package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

var PkgPathArrayDSL = func() {
	var Foo = Type("Foo", func() {
		Attribute("IntField", Int)
		Meta("struct:pkg:path", "foo")
	})
	Service("PkgPathArrayMethod", func() {
		Method("A", func() {
			Payload(ArrayOf(Foo))
			Result(ArrayOf(Foo))
		})
	})
}
