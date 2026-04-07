package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadHeaderCustomNameDSL = func() {
	Service("ServiceHeaderCustomName", func() {
		Method("MethodHeaderCustomName", func() {
			Payload(func() {
				Attribute("h", String, func() {
					Meta("struct:field:name", "Header")
				})
			})
			HTTP(func() {
				GET("/")
				Headers(func() {
					Header("h")
				})
			})
		})
	})
}


