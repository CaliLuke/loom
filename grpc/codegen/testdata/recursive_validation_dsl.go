package testdata

import . "github.com/CaliLuke/loom/dsl"

// RecursiveValidationDSL covers the independent unary, stream, error and view
// validation call forms with the same recursive named message.
var RecursiveValidationDSL = func() {
	API("recursivevalidation", func() {
	})
	node := Type("Node", func() {
		Field(1, "value", String, func() {
			Pattern("^[a-z]+$")
		})
		Field(2, "next", "Node")
		Required("value")
	})
	failure := Type("Failure", func() {
		ErrorName(1, "name", String)
		Field(2, "tree", node)
		Required("name", "tree")
	})
	view := ResultType("application/vnd.recursive", func() {
		TypeName("Viewed")
		Field(1, "tree", node)
		Required("tree")
		View("default", func() {
			Attribute("tree")
		})
	})
	Service("trees", func() {
		Error("failure", failure)
		Method("check", func() {
			Payload(node)
			Result(node)
			GRPC(func() {
				Response(CodeOK)
				Response("failure", CodeInvalidArgument)
			})
		})
		Method("exchange", func() {
			StreamingPayload(node)
			StreamingResult(node)
			GRPC(func() {
				Response(CodeOK)
				Response("failure", CodeInvalidArgument)
			})
		})
		Method("watch", func() {
			StreamingResult(view)
			GRPC(func() {
				Response(CodeOK)
				Response("failure", CodeInvalidArgument)
			})
		})
	})
}
