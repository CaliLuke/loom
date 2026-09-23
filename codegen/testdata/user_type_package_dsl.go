package testdata

import . "github.com/CaliLuke/loom/dsl"

// UserTypePackageTransportsDSL declares user types placed in the "common"
// package with struct:pkg:path metadata. It uses them in payloads, results,
// errors, streaming results, nested fields, arrays, and maps over HTTP, gRPC,
// and JSON-RPC, including JSON-RPC SSE and WebSocket streams of local
// wrappers. Payloads also have a field named "common", so generated variables
// must not shadow the package name.
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
	payload := func() {
		Field(1, "item", item)
		Field(2, "common", String)
		Field(3, "items", ArrayOf(item))
		Field(4, "index", MapOf(String, item))
		Required("item")
	}
	Service("catalog", func() {
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
