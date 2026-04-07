package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadDeepUserDSL = func() {
	var DeepChild = ResultType("DeepChild", func() {
		Attribute("name", String)
	})
	var ImmediateChild = ResultType("ImmediateChild", func() {
		Attributes(func() {
			Attribute("deep_child", DeepChild)
		})
		View("other", func() {
			Attribute("deep_child")
		})
	})
	var ImmediateChildExtender = ResultType("ImmediateChildExtender", func() {
		Extend(ImmediateChild)
		Attributes(func() {
			Attribute("deep_child", DeepChild)
		})
		View("other", func() {
			Attribute("deep_child")
		})
	})
	var TopLevel = ResultType("TopLevel", func() {
		Attributes(func() {
			Attribute("immediate_child", ImmediateChildExtender)
		})
	})
	Service("ServiceDeepUser", func() {
		Method("MethodDeepUser", func() {
			Result(TopLevel)
			HTTP(func() { GET("/") })
		})
	})
}


