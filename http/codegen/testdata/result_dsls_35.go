package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// Result(Header|Body)(Type)(Required|Default)?DSL
//
// Where Type is the type of the result or result.


var ResultBodyUnionCustomKeysMultiDSL = func() {
	var TypeA = Type("TypeA", func() {
		Attribute("a", Int)
	})
	var TypeB = Type("TypeB", func() {
		Attribute("b", String)
	})
	var PaymentResponse = Type("PaymentResponse", func() {
		OneOf("status", func() {
			Meta("oneof:type:field", "statusType")
			Meta("oneof:value:field", "statusDetails")
			Attribute("success", TypeA)
			Attribute("failure", TypeB)
		})
	})
	Service("ServiceResultUnionCustomKeysMulti", func() {
		Method("MethodResultUnionCustomKeysMulti", func() {
			Result(PaymentResponse)
			HTTP(func() {
				GET("/")
			})
		})
	})
}

