package testdata

import . "github.com/CaliLuke/loom/dsl"

// UserTypePackageTransportsDSL declares user types placed in the "common"
// package with struct:pkg:path metadata. It uses them in payloads, results,
// errors, streaming results, nested fields, arrays, and maps over HTTP, gRPC,
// and JSON-RPC, including JSON-RPC SSE and WebSocket streams of local
// wrappers. Payloads also have a field named "common", so generated variables
// must not shadow the package name. Relocated type names also match method
// names in two services, so references must share destination allocations.
// An authored MovedUnionKind and an inline PickKind branch also compete with
// union discriminator names in that physical package. One raw union also belongs
// to two distinct destination packages and to one common package through
// different service/file candidates.
var UserTypePackageTransportsDSL = func() {
	API("usertypepkg", func() {
		JSONRPC(func() {})
	})
	inner := Type("Inner", func() {
		Meta("struct:pkg:path", "common")
		Field(1, "count", Int)
	})
	item := Type("Item", func() {
		Meta("struct:pkg:path", "common")
		Field(1, "name", String)
		Field(2, "inner", inner)
		Field(3, "inners", ArrayOf(inner))
		Field(4, "inner_index", MapOf(String, inner))
		Required("name")
	})
	fault := Type("Fault", func() {
		Meta("struct:pkg:path", "common")
		Field(1, "message", String)
		Field(2, "name", String)
		ErrorName("name")
		Required("message", "name")
	})
	wrap := Type("Wrap", func() {
		Field(1, "item", item)
		Field(2, "items", ArrayOf(item))
		Required("item")
	})
	// Explicit primitive representations keep this authored union raw when the
	// same shape is reached from two different generated packages.
	sharedChoice := OneOf(func() {
		Field(1, "text", String, func() {
			Meta("struct:field:type", "string")
		})
		Field(2, "count", Int, func() {
			Meta("struct:field:type", "int")
		})
	})
	left := Type("SharedLeft", func() {
		Meta("struct:pkg:path", "left")
		Field(1, "pick", sharedChoice)
	})
	right := Type("SharedRight", func() {
		Meta("struct:pkg:path", "right")
		Field(1, "pick", sharedChoice)
	})
	sharedDestinations := Type("SharedDestinations", func() {
		Field(1, "left", left)
		Field(2, "right", right)
	})
	globalLeft := Type("GlobalLeft", func() {
		Meta("struct:pkg:path", "common")
		Field(1, "pick", sharedChoice)
	})
	globalRight := Type("GlobalRight", func() {
		Meta("struct:pkg:path", "common")
		Field(1, "pick", sharedChoice)
	})
	relocated := []struct {
		name  string
		value any
	}{
		{"shared_destinations", sharedDestinations},
		{"moved_empty", Type("MovedEmpty", func() {
			Meta("struct:pkg:path", "common")
		})},
		{"moved_object", Type("MovedObject", item, func() {
			Meta("struct:pkg:path", "common")
		})},
		{"moved_array", Type("MovedArray", ArrayOf(item), func() {
			Meta("struct:pkg:path", "common")
		})},
		{"moved_map", Type("MovedMap", MapOf(String, item), func() {
			Meta("struct:pkg:path", "common")
		})},
		{"moved_primitive", Type("MovedPrimitive", String, func() {
			Meta("struct:pkg:path", "common")
		})},
		{"moved_inline_union", Type("MovedInlineUnion", func() {
			Meta("struct:pkg:path", "common")
			OneOf("pick", func() {
				Field(1, "Kind", String)
				Field(2, "other", Int)
			})
		})},
		{"moved_union", Type("MovedUnion", OneOf(inner, item), func() {
			Meta("struct:pkg:path", "common")
		})},
	}
	unionKind := Type("MovedUnionKind", String, func() {
		Meta("struct:pkg:path", "common")
	})
	payload := func() {
		Field(1, "item", item)
		Field(2, "common", String)
		Field(3, "items", ArrayOf(item))
		Field(4, "index", MapOf(String, item))
		Field(5, "kind", unionKind)
		Required("item")
	}
	Service("catalog", func() {
		Method("global_left", func() {
			Payload(globalLeft)
			Result(globalLeft)
			HTTP(func() {
				POST("/names/global_left")
			})
			GRPC(func() {})
		})
		for _, shape := range relocated {
			Method(shape.name, func() {
				Payload(shape.value)
				Result(shape.value)
				HTTP(func() {
					POST("/names/" + shape.name)
				})
				GRPC(func() {})
			})
		}
		Error("invalid", fault)
		Method("update", func() {
			Payload(payload)
			Result(item)
			HTTP(func() {
				POST("/update")
				Param("common")
				Response(StatusOK)
				Response("invalid", StatusBadRequest)
			})
			GRPC(func() {
				Response("invalid", CodeInvalidArgument)
			})
		})
		Method("index", func() {
			Payload(ArrayOf(item))
			Result(MapOf(String, item))
			HTTP(func() {
				POST("/index")
			})
		})
		Method("watch", func() {
			Payload(func() {
				Field(1, "common", String)
			})
			StreamingResult(item)
			HTTP(func() {
				GET("/watch")
				Param("common")
				ServerSentEvents()
			})
		})
		Method("follow", func() {
			Payload(func() {
				Field(1, "common", String)
			})
			StreamingResult(item)
			HTTP(func() {
				GET("/follow")
				Param("common")
			})
			GRPC(func() {})
		})
	})
	Service("catalogrpc", func() {
		Method("global_right", func() {
			Payload(func() {
				ID("id", String)
				Field(1, "value", globalRight)
			})
			Result(globalRight)
			JSONRPC(func() {})
		})
		for _, shape := range relocated {
			Method(shape.name, func() {
				Payload(func() {
					ID("id", String)
					Field(1, "value", shape.value)
				})
				Result(shape.value)
				JSONRPC(func() {})
			})
		}
		Error("invalid", fault)
		JSONRPC(func() {
			POST("/rpc")
		})
		Method("update", func() {
			Payload(func() {
				ID("id", String)
				payload()
			})
			Result(item)
			JSONRPC(func() {
				Response("invalid", func() {
					Code(-32001)
				})
			})
		})
	})
	Service("catalogsse", func() {
		Error("invalid", fault)
		JSONRPC(func() {
			POST("/sse")
		})
		Method("watch", func() {
			Payload(func() {
				ID("id", String)
				Field(1, "common", String)
			})
			StreamingResult(wrap)
			JSONRPC(func() {
				ServerSentEvents()
				Response("invalid", func() {
					Code(-32001)
				})
			})
		})
	})
	Service("catalogws", func() {
		Error("invalid", fault)
		JSONRPC(func() {
			GET("/ws")
		})
		Method("exchange", func() {
			StreamingPayload(wrap)
			StreamingResult(wrap)
			JSONRPC(func() {
				Response("invalid", func() {
					Code(-32001)
				})
			})
		})
	})
}
