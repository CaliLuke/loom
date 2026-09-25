package testdata

import . "github.com/CaliLuke/loom/dsl"

// ProtoStructMetaDSL declares gRPC types generated in the "menu" package with
// struct:pkg:path metadata, most of which also name their protocol buffer
// message with struct:name:proto. The types are used directly as payloads,
// results, streaming results, stream items and errors, where Loom names
// their messages after the struct:name:proto metadata, and nested as message
// fields, array elements, map values and union branches, where their
// messages keep the names of the types. Choice is a named union and Tags a
// named array, whose wrapper messages keep the names of the types. Node is
// recursive, and its struct:name:proto name "node_tree" is not the Go name
// protoc-gen-go generates. Event stays in the service package because the
// service code of streaming payloads of relocated types does not compile yet
// (#372).
var ProtoStructMetaDSL = func() {
	inner := Type("Inner", func() {
		Meta("struct:pkg:path", "menu")
		Meta("struct:name:proto", "InnerProto")
		Field(1, "count", Int)
	})
	leaf := Type("Leaf", func() {
		Meta("struct:pkg:path", "menu")
		Meta("struct:name:proto", "LeafProto")
		Field(1, "name", String)
	})
	other := Type("Other", func() {
		Meta("struct:pkg:path", "menu")
		Field(1, "flag", Boolean)
	})
	choice := Type("Choice", OneOf(leaf, other), func() {
		Meta("struct:pkg:path", "menu")
		Meta("struct:name:proto", "ChoiceProto")
	})
	tags := Type("Tags", ArrayOf(String), func() {
		Meta("struct:pkg:path", "menu")
		Meta("struct:name:proto", "TagList")
	})
	node := Type("Node", func() {
		Meta("struct:pkg:path", "menu")
		Meta("struct:name:proto", "node_tree")
		Field(1, "label", String)
		Field(2, "children", ArrayOf("Node"))
	})
	event := Type("Event", func() {
		Meta("struct:name:proto", "EventProto")
		Field(1, "id", String)
	})
	menu := Type("Menu", func() {
		Meta("struct:pkg:path", "menu")
		Meta("struct:name:proto", "MenuProto")
		Field(1, "name", String)
		Field(2, "inner", inner)
		Field(3, "inners", ArrayOf(inner))
		Field(4, "inner_index", MapOf(String, inner))
		Field(5, "choice", choice)
		Field(7, "tags", tags)
		Field(8, "tree", node)
		Required("name")
	})
	fault := Type("Fault", func() {
		Meta("struct:pkg:path", "menu")
		Meta("struct:name:proto", "FaultProto")
		Field(1, "message", String)
		Field(2, "name", String)
		ErrorName("name")
		Required("message", "name")
	})
	Service("menusvc", func() {
		Error("invalid", fault)
		Method("show", func() {
			Payload(menu)
			Result(menu)
			GRPC(func() {
				Response("invalid", CodeInvalidArgument)
			})
		})
		Method("pick", func() {
			Payload(choice)
			Result(choice)
			GRPC(func() {})
		})
		Method("branch", func() {
			Payload(OneOf(choice, inner))
			Result(OneOf(choice, inner))
			GRPC(func() {})
		})
		Method("tree", func() {
			Payload(node)
			Result(node)
			GRPC(func() {})
		})
		Method("tags", func() {
			Payload(tags)
			Result(func() {
				Field(1, "tags", tags)
			})
			GRPC(func() {})
		})
		Method("bidi", func() {
			Payload(func() {
				Field(1, "q", String)
			})
			StreamingPayload(event)
			StreamingResult(event)
			GRPC(func() {})
		})
		Method("relay", func() {
			Payload(func() {
				Field(1, "q", String)
			})
			StreamingPayload(event)
			StreamingResult(func() {
				Field(1, "event", event)
			})
			GRPC(func() {})
		})
		Method("watch", func() {
			Payload(inner)
			StreamingResult(menu)
			GRPC(func() {})
		})
	})
}
