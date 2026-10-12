package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// RecursiveEffectiveCompositionDSL distinguishes recursive members from Extend
// dependencies in both declaration orders, across HTTP and gRPC generation.
var RecursiveEffectiveCompositionDSL = func() {
	var base, node, reverseBase, reverseNode expr.UserType
	base = Type("RecursiveBase", func() {
		Field(1, "next", node)
	})
	node = Type("RecursiveNode", func() {
		Extend(base)
		Field(2, "n", Int)
	})
	reverseNode = Type("ReverseNode", func() {
		Extend(reverseBase)
		Field(2, "n", Int)
	})
	reverseBase = Type("ReverseBase", func() {
		Field(1, "next", reverseNode)
	})
	Service("recursive_composition", func() {
		Method("forward", func() {
			Payload(node)
			Result(node)
			HTTP(func() {
				POST("/forward")
			})
			GRPC(func() {
			})
		})
		Method("reverse", func() {
			Payload(reverseNode)
			Result(reverseNode)
			HTTP(func() {
				POST("/reverse")
			})
			GRPC(func() {
			})
		})
	})
}
