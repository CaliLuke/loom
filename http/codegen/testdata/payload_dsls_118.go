package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadBodyInlineMapUserDSL = func() {
	var KeyType = Type("KeyType", func() {
		Attribute("a", String, func() {
			Pattern("patterna")
		})
		Attribute("b", String, func() {
			Pattern("patternb")
		})
		Required("a")
	})
	var ElemType = Type("ElemType", func() {
		Attribute("a", String, func() {
			Pattern("patterna")
		})
		Attribute("b", String, func() {
			Pattern("patternb")
		})
		Required("a")
	})
	Service("ServiceBodyInlineMapUser", func() {
		Method("MethodBodyInlineMapUser", func() {
			Payload(MapOf(KeyType, ElemType))
			HTTP(func() {
				POST("/")
			})
		})
	})
}


