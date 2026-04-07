package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

var PkgPathMultipleDSL = func() {
	var Bar = Type("Bar", func() {
		Attribute("IntField", Int)
		Meta("struct:pkg:path", "bar")
	})
	var Baz = Type("Baz", func() {
		Attribute("IntField", Int)
		Meta("struct:pkg:path", "baz")
	})

	Service("MultiplePkgPathMethod", func() {
		Method("A", func() {
			Payload(Bar)
			Result(Bar)
		})

		Method("B", func() {
			Payload(Baz)
			Result(Baz)
		})

		Method("EnvelopedB", func() {
			Payload(func() {
				Attribute("Baz", Baz)
			})
			Result(func() {
				Attribute("Baz", Baz)
			})
		})
	})
}
