package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// NamedOneOfFieldDSL is ConstructorOneOfFieldDSL with every union declared
// as a named type with Type and passed to Field. Field numbers the branches of
// each named union consecutively from the field number, as it does for a
// constructor OneOf.
var NamedOneOfFieldDSL = func() {
	var Leaf = Type("Leaf", func() {
		Field(1, "name", String)
	})
	var Other = Type("Other", func() {
		Field(1, "count", Int)
	})
	var Extra = Type("Extra", func() {
		Field(1, "flag", Boolean)
	})
	var Pick = Type("Pick", OneOf(Leaf, Other))
	var Holder = Type("Holder", func() {
		Field(1, "label", String)
		Field(2, "inner", Pick)
	})
	var Must = Type("Must", OneOf(Extra, Holder))
	var Envelope = Type("Envelope", func() {
		Field(1, "id", String)
		Field(2, "pick", Pick)
		Field(4, "must", Must)
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

// NamedUnionFieldReuseDSL uses one named union as an optional field of a
// type, as a required field of another type with a different field number,
// directly as the payload and result of a method, and as a field of a type
// used as array elements. Choice is the wrapper message of the direct payload
// and result, and each field holds the branches in a oneof of its own
// message.
var NamedUnionFieldReuseDSL = func() {
	var Leaf = Type("Leaf", func() {
		Field(1, "name", String)
	})
	var Other = Type("Other", func() {
		Field(1, "count", Int)
	})
	var Choice = Type("Choice", OneOf(Leaf, Other))
	var Holder = Type("Holder", func() {
		Field(1, "label", String)
		Field(2, "choice", Choice)
	})
	var Envelope = Type("Envelope", func() {
		Field(1, "id", String)
		Field(3, "choice", Choice)
		Field(5, "holder", Holder)
		Field(6, "holders", ArrayOf(Holder))
		Required("choice")
	})
	Service("reuse", func() {
		Method("echo", func() {
			Payload(Envelope)
			Result(Holder)
			GRPC(func() {})
		})
		Method("named", func() {
			Payload(Choice)
			Result(Choice)
			GRPC(func() {})
		})
		Method("relay", func() {
			StreamingPayload(Holder)
			StreamingResult(Envelope)
			GRPC(func() {})
		})
	})
}
