package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

var PkgPathRecursiveDSL = func() {
	var Foo = Type("Foo", func() {
		Attribute("IntField", Int)
		Meta("struct:pkg:path", "foo")
	})
	var RecursiveFoo = Type("RecursiveFoo", func() {
		Attribute("Foo", Foo)
		Meta("struct:pkg:path", "foo")
	})

	Service("PkgPathRecursiveMethod", func() {
		Method("A", func() {
			Payload(RecursiveFoo)
			Result(RecursiveFoo)
		})
	})
}
