package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

var (
	// ExtendedMethodTypesDSL reuses a named type in plain and extended method
	// positions. The extension contributes a required, validated field.
	ExtendedMethodTypesDSL = func() {
		extendedMethodTypesDSL("direct")
	}
	// ExtendedAliasedMethodTypesDSL exercises the same contracts through two
	// named aliases, which must each retain an independent extended declaration.
	ExtendedAliasedMethodTypesDSL = func() {
		extendedMethodTypesDSL("aliased")
	}
	// ExtendedResultMethodTypesDSL verifies automatic views of extended results
	// include inherited fields and preserve their requiredness on the wire.
	ExtendedResultMethodTypesDSL = func() {
		extendedMethodTypesDSL("result-type")
	}
)

func extendedMethodTypesDSL(kind string) {
	base := Type("Base", func() {
		Attribute("left", String, func() {
			MinLength(1)
		})
		Required("left")
	})
	var definition any = func() {
		Attribute("right", String)
	}
	if kind == "aliased" {
		definition = Type("Inner", Type("Leaf", definition))
	}
	var value expr.UserType
	if kind == "result-type" {
		value = ResultType("application/vnd.value", "Value", func() {
			Attributes(definition.(func()))
		})
	} else {
		value = Type("Value", definition)
	}
	Service("fields", func() {
		Method("plain", func() {
			Payload(value)
			Result(value)
			HTTP(func() {
				POST("/plain")
			})
		})
		Method("send", func() {
			Payload(value, func() {
				Extend(base)
			})
			Result(value)
			HTTP(func() {
				POST("/send")
			})
		})
		Method("show", func() {
			Payload(value)
			Result(value, func() {
				Extend(base)
			})
			HTTP(func() {
				POST("/show")
			})
		})
		Method("upload", func() {
			StreamingPayload(value, func() {
				Extend(base)
			})
			Result(value)
			HTTP(func() {
				GET("/upload")
			})
		})
		Method("watch", func() {
			StreamingResult(value, func() {
				Extend(base)
			})
			HTTP(func() {
				GET("/watch")
			})
		})
	})
}
