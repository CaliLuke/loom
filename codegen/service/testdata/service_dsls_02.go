package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

var PkgPathDSL = func() {
	var Foo = Type("Foo", func() {
		Attribute("IntField", Int)
		Meta("struct:pkg:path", "foo")
	})
	Service("PkgPathMethod", func() {
		Method("A", func() {
			Payload(Foo)
			Result(Foo)
		})
	})
}
