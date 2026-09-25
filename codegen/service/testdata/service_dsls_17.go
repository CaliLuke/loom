package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// CustomizedResultCopiesDSL uses one result type with views as the result of
// three methods. Two methods customize the requiredness of the result type,
// so each gets a renamed copy that keeps the identifier of the result type.
var CustomizedResultCopiesDSL = func() {
	var Menu = ResultType("application/vnd.menu", "Menu", func() {
		Attributes(func() {
			Attribute("x", String)
			Attribute("y", String)
		})
		View("default", func() {
			Attribute("x")
			Attribute("y")
		})
		View("tiny", func() {
			Attribute("x")
		})
	})
	Service("CustomizedResultCopies", func() {
		Method("M1", func() {
			Result(Menu, func() {
				Required("x")
			})
		})
		Method("M2", func() {
			Result(Menu, func() {
				Required("x")
				View("tiny")
			})
		})
		Method("M3", func() {
			Result(Menu)
		})
	})
}
