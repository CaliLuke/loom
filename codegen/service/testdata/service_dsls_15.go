package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

var UnionCustomKeysMultiTypeDSL = func() {
	var TypeA = Type("TypeA", func() {
		Attribute("a", Int)
	})
	var TypeB = Type("TypeB", func() {
		Attribute("b", String)
	})
	var PaymentMethod = Type("PaymentMethod", func() {
		OneOf("method", func() {
			Meta("oneof:type:field", "paymentType")
			Meta("oneof:value:field", "details")
			Attribute("credit_card", TypeA)
			Attribute("paypal", TypeB)
		})
	})
	Service("PaymentService", func() {
		Method("ProcessPayment", func() {
			Payload(PaymentMethod)
			Result(PaymentMethod)
		})
	})
}

// UnionDefaultKeysDSL tests that unions without custom Meta tags still use "type" and "value".
