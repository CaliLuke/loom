package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// ConstructorOneOfFieldDSL passes constructor OneOf unions to Field: an
// optional union, a required union, and a union in a type used directly and
// as array elements. Field numbers the branches of each union consecutively
// from the field number.
var ConstructorOneOfFieldDSL = func() {
	var Leaf = Type("Leaf", func() {
		Field(1, "name", String)
	})
	var Other = Type("Other", func() {
		Field(1, "count", Int)
	})
	var Extra = Type("Extra", func() {
		Field(1, "flag", Boolean)
	})
	var Holder = Type("Holder", func() {
		Field(1, "label", String)
		Field(2, "inner", OneOf(Leaf, Other))
	})
	var Envelope = Type("Envelope", func() {
		Field(1, "id", String)
		Field(2, "pick", OneOf(Leaf, Other))
		Field(4, "must", OneOf(Extra, Holder))
		Field(6, "holders", ArrayOf(Holder))
		Required("must")
	})
	Service("pickunion", func() {
		Method("echo", func() {
			Payload(Envelope)
			Result(Envelope)
			GRPC(func() {})
		})
	})
}

// BlockOneOfFieldDSL is ConstructorOneOfFieldDSL with every union defined
// with the block form of OneOf, with the branch names and field numbers that
// the constructor form derives.
var BlockOneOfFieldDSL = func() {
	var Leaf = Type("Leaf", func() {
		Field(1, "name", String)
	})
	var Other = Type("Other", func() {
		Field(1, "count", Int)
	})
	var Extra = Type("Extra", func() {
		Field(1, "flag", Boolean)
	})
	var Holder = Type("Holder", func() {
		Field(1, "label", String)
		OneOf("inner", func() {
			Field(2, "Leaf", Leaf)
			Field(3, "Other", Other)
		})
	})
	var Envelope = Type("Envelope", func() {
		Field(1, "id", String)
		OneOf("pick", func() {
			Field(2, "Leaf", Leaf)
			Field(3, "Other", Other)
		})
		OneOf("must", func() {
			Field(4, "Extra", Extra)
			Field(5, "Holder", Holder)
		})
		Field(6, "holders", ArrayOf(Holder))
		Required("must")
	})
	Service("pickunion", func() {
		Method("echo", func() {
			Payload(Envelope)
			Result(Envelope)
			GRPC(func() {})
		})
	})
}
