package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// MappedNamesDSL declares payload and result attributes with a transport
// element name suffix, such as "n:m", in a type, a nested type, unions and an
// inline payload: optional, required and defaulted primitives, a validated
// string, an object, an array, a map, a constructor OneOf and a OneOf block
// with suffixed branches. The HTTP bodies use the suffix as the JSON name of
// the field.
var MappedNamesDSL = func() {
	var Leaf = Type("Leaf", func() {
		Attribute("leaf:l", String)
		Attribute("count:c", Int)
		Required("count:c")
	})
	var Envelope = Type("Envelope", func() {
		Attribute("n:m", String, func() {
			MinLength(2)
		})
		Attribute("req:r", Int)
		Attribute("def:d", Int, func() {
			Default(3)
		})
		Attribute("pick:p", OneOf(String, Int))
		Attribute("obj:o", Leaf)
		Attribute("list:ls", ArrayOf(String))
		Attribute("index:ix", MapOf(String, Leaf))
		OneOf("choice:ch", func() {
			Attribute("text:t", String)
			Attribute("leaf_branch:lb", Leaf)
		})
		Required("req:r", "pick:p", "obj:o")
	})
	Service("mappednames", func() {
		Method("echo", func() {
			Payload(Envelope)
			Result(Envelope)
			HTTP(func() {
				POST("/echo")
			})
		})
		Method("inline", func() {
			Payload(func() {
				Attribute("id:i", String)
				Attribute("count:c", Int)
				Required("id:i")
			})
			Result(func() {
				Attribute("id:i", String)
				Attribute("count:c", Int)
				Required("id:i")
			})
			HTTP(func() {
				PUT("/inline")
			})
		})
	})
}
