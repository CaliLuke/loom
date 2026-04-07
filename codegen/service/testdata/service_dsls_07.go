package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

var PkgPathDupeDSL = func() {
	var Foo = Type("Foo", func() {
		Attribute("IntField", Int)
		Meta("struct:pkg:path", "foo")
	})

	Service("PkgPathDupeMethod", func() {
		Method("A", func() {
			Payload(Foo)
			Result(Foo)
		})
		Method("B", func() {
			Payload(Foo)
			Result(Foo)
		})
	})
	Service("PkgPathDupeMethod2", func() {
		Method("A", func() {
			Payload(Foo)
			Result(Foo)
		})
		Method("B", func() {
			Payload(Foo)
			Result(Foo)
		})
	})
}
