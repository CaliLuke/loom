package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadBodyUnionCustomKeysValidateDSL = func() {
	var PaymentMethod = Type("PaymentMethod", func() {
		OneOf("method", func() {
			Meta("oneof:type:field", "paymentType")
			Meta("oneof:value:field", "details")
			Attribute("CreditCard", String)
			Attribute("PayPal", String)
		})
	})
	Service("ServiceBodyUnionCustomKeysValidate", func() {
		Method("MethodBodyUnionCustomKeysValidate", func() {
			Payload(func() {
				Attribute("payment", PaymentMethod)
				Required("payment")
			})
			HTTP(func() {
				POST("/")
			})
		})
	})
}

