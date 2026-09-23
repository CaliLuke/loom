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

// UnionMessageDSL uses constructor OneOf unions directly as the payload and
// result of methods, anonymous and named with Type, in every unary and
// streaming position. Each request and response message wraps the union in
// a oneof whose branches are numbered consecutively from 1.
var UnionMessageDSL = func() {
	var Leaf = Type("Leaf", func() {
		Field(1, "name", String)
	})
	var Other = Type("Other", func() {
		Field(1, "count", Int)
	})
	var Choice = Type("Choice", OneOf(Leaf, Other))
	Service("pickunion", func() {
		Method("echo", func() {
			Payload(OneOf(Leaf, Other))
			Result(OneOf(Leaf, Other))
			GRPC(func() {})
		})
		Method("named", func() {
			Payload(Choice)
			Result(Choice)
			GRPC(func() {})
		})
		Method("watch", func() {
			Payload(OneOf(Leaf, Other))
			StreamingResult(OneOf(Leaf, Other))
			GRPC(func() {})
		})
		Method("upload", func() {
			StreamingPayload(OneOf(Leaf, Other))
			Result(OneOf(Leaf, Other))
			GRPC(func() {})
		})
		Method("relay", func() {
			StreamingPayload(OneOf(Leaf, Other))
			StreamingResult(OneOf(Leaf, Other))
			GRPC(func() {})
		})
	})
}

// UnionMessageBranchNameDSL uses unions directly as payloads and results
// whose branch names collide with the name of the oneof of the message that
// wraps them: a branch named "field", and branches named "field" and
// "field_oneof".
var UnionMessageBranchNameDSL = func() {
	var FieldType = Type("Field", func() {
		Field(1, "name", String)
	})
	var FieldOneofType = Type("FieldOneof", func() {
		Field(1, "count", Int)
	})
	var Other = Type("Other", func() {
		Field(1, "flag", Boolean)
	})
	Service("branchname", func() {
		Method("echo", func() {
			Payload(OneOf(FieldType, Other))
			Result(OneOf(FieldType, Other))
			GRPC(func() {})
		})
		Method("clash", func() {
			Payload(OneOf(FieldType, FieldOneofType))
			Result(OneOf(FieldType, FieldOneofType))
			GRPC(func() {})
		})
		Method("relay", func() {
			StreamingPayload(OneOf(FieldType, Other))
			StreamingResult(OneOf(FieldType, Other))
			GRPC(func() {})
		})
	})
}
