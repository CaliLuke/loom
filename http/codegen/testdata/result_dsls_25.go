package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// Result(Header|Body)(Type)(Required|Default)?DSL
//
// Where Type is the type of the result or result.


var EmptyServerResponseWithTagsDSL = func() {
	Service("ServiceEmptyServerResponseWithTags", func() {
		Method("MethodEmptyServerResponseWithTags", func() {
			Result(func() {
				Attribute("h", String)
				Required("h")
			})
			HTTP(func() {
				GET("/")
				Response(StatusNoContent, func() {
					Body(Empty)
				})
				Response(StatusNotModified, func() {
					Tag("h", "true")
					Body(Empty)
				})
			})
		})
	})
}


