package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var MultipleServicesSamePayloadAndResultDSL = func() {
	Service("ServiceA", func() {
		Method("list", func() {
			Payload(func() {
				Attribute("name", String)
			})
			StreamingPayload(func() {
				Attribute("name", String)
			})
			Result(func() {
				Attribute("id", Int)
				Attribute("name", String)
				Required("id", "name")
			})
			Error("something_went_wrong")
			HTTP(func() {
				GET("/{name}")
				Response(StatusOK)
				Response("something_went_wrong", StatusInternalServerError)
			})
		})
	})
	Service("ServiceB", func() {
		Method("list", func() {
			Payload(func() {
				Attribute("name", String)
			})
			StreamingPayload(func() {
				Attribute("name", String)
			})
			Result(func() {
				Attribute("id", Int)
				Attribute("name", String)
				Required("id", "name")
			})
			Error("something_went_wrong")
			HTTP(func() {
				GET("/{name}")
				Response(StatusOK)
				Response(StatusInternalServerError, "something_went_wrong")
			})
		})
	})
}


