package testdata

import . "github.com/CaliLuke/loom/dsl"

// RecursiveNamedArrayDSL uses a named array of objects that reach the named
// array through array elements and map values as a request and response
// body. Generating its examples used to panic, and its rendered examples must
// hold no null object placeholder, since the schemas do not admit null.
var RecursiveNamedArrayDSL = func() {
	var Node = Type("Node", func() {
		Attribute("id", String)
		Attribute("grid", ArrayOf("Nodes"))
		Attribute("idx", MapOf(String, "Nodes"))
	})
	var Nodes = Type("Nodes", ArrayOf(Node))
	var _ = API("test", func() {
		Server("test", func() {
			Host("localhost", func() {
				URI("https://loom.design")
			})
		})
	})
	Service("testService", func() {
		Method("walk", func() {
			Payload(Nodes)
			Result(Nodes)
			HTTP(func() {
				POST("/")
			})
		})
	})
}
