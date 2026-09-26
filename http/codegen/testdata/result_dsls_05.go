package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// Result(Header|Body)(Type)(Required|Default)?DSL
//
// Where Type is the type of the result or result.

var ExplicitBodyUserResultObjectDSL = func() {
	var UserType = Type("UserType", func() {
		Attribute("x", String)
		Attribute("y", Int)
	})
	var ResultType = ResultType("ResultType", func() {
		Attribute("a", UserType)
		Attribute("b", String)
		Attribute("c", String)
	})
	Service("ServiceExplicitBodyUserResultObject", func() {
		Method("MethodExplicitBodyUserResultObject", func() {
			Result(ResultType)
			HTTP(func() {
				POST("/")
				Response(StatusOK, func() {
					Header("c:Location")
					Header("b:Content-Type")
					Body(func() {
						Attribute("a")
					})
				})
			})
		})
	})
}

// ExplicitBodyResultTypeDSL declares response bodies with the result type of
// the method and with the result type of an error, such as Body(Listing) with
// Result(Listing). The listing nests another result type and a collection of
// it, and a second method returns the same listing with an implicit body.
var ExplicitBodyResultTypeDSL = func() {
	var Item = ResultType("application/vnd.item", "Item", func() {
		Attribute("id", String)
		Attribute("name", String)
		Required("id")
	})
	var Listing = ResultType("application/vnd.listing", "Listing", func() {
		Attribute("first", Item)
		Attribute("items", CollectionOf(Item))
		Attribute("next", String)
		Required("first", "items")
	})
	var Problem = ResultType("application/vnd.problem", "Problem", func() {
		Attribute("code", String)
		Attribute("detail", String)
		Required("code")
	})
	Service("ServiceExplicitBodyResultType", func() {
		Method("MethodExplicitBodyResultType", func() {
			NoSecurity()
			Result(Listing)
			Error("unavailable", Problem)
			HTTP(func() {
				POST("/listings")
				Response(StatusOK, func() {
					Body(Listing)
				})
				Response("unavailable", StatusServiceUnavailable, func() {
					Body(Problem)
				})
			})
		})
		Method("MethodImplicitBodyResultType", func() {
			NoSecurity()
			Result(Listing)
			HTTP(func() {
				GET("/listings")
			})
		})
	})
}
