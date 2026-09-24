package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// UnionBranchCollisionDSL declares messages whose union fields have branch or
// oneof names that collide with the names of other fields, oneofs or oneof
// branches of the same message. The fields, oneofs and oneof branches of a
// message share one namespace, so the colliding names must be made unique.
//
// Envelope has three unions that share an Int64 branch, in constructor and
// block form, a regular field named like an object branch of a constructor
// union, of a block union with validations and of a named union. Holder uses
// two named unions twice each, one with a branch that has validations. Tree has a union field named like one of its
// branches, next to a named union whose branches collide with it. Named has a
// named union field named like one of its branches, next to a regular field
// named like the renamed oneof. Pair has a oneof named like a branch of
// another oneof.
var UnionBranchCollisionDSL = func() {
	var Leaf = Type("Leaf", func() {
		Field(1, "name", String)
	})
	var Other = Type("Other", func() {
		Field(1, "count", Int)
	})
	var Code = Type("Code", String, func() {
		MinLength(2)
	})
	var Choice = Type("Choice", OneOf(Leaf, Other))
	var Tagged = Type("Tagged", OneOf(Code, Other))
	var Holder = Type("Holder", func() {
		Field(1, "first", Choice)
		Field(3, "second", Choice)
		Field(5, "tag", Tagged)
		Field(7, "alt", Tagged)
		Required("first")
	})
	var Envelope = Type("Envelope", func() {
		Field(1, "a", OneOf(String, Int64))
		Field(3, "b", OneOf(Boolean, Int64))
		Field(5, "c", OneOf(Int64, Leaf))
		Field(7, "leaf", String)
		OneOf("d", func() {
			Field(8, "Int64", Int64)
			Field(9, "leaf", String, func() {
				MinLength(2)
			})
		})
		Field(10, "pick", Choice)
		Field(12, "holder", Holder)
		Field(13, "holders", ArrayOf(Holder))
		Required("a")
	})
	var Tree = Type("Tree", func() {
		Field(1, "leaf", OneOf(Leaf, Other))
		Field(3, "choice", Choice)
		Required("leaf")
	})
	var Named = Type("Named", func() {
		Field(1, "leaf", Choice)
		Field(3, "leaf_oneof", String)
		Required("leaf")
	})
	var Pair = Type("Pair", func() {
		Field(1, "u", OneOf(Leaf, Other))
		OneOf("w", func() {
			Field(3, "u", Leaf)
			Field(4, "v", Other)
		})
	})
	Service("collision", func() {
		Method("echo", func() {
			Payload(Envelope)
			Result(Envelope)
			GRPC(func() {})
		})
		Method("grow", func() {
			Payload(Tree)
			Result(Tree)
			GRPC(func() {})
		})
		Method("match", func() {
			Payload(Named)
			Result(Pair)
			GRPC(func() {})
		})
	})
}
