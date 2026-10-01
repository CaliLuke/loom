package testdata

import . "github.com/CaliLuke/loom/dsl"

// CLIIdentifierCollisionDSL declares one adversarial aggregate CLI namespace
// for HTTP, JSON-RPC, and gRPC, plus an importable service named main.
func CLIIdentifierCollisionDSL() {
	API("cli identifiers", func() {
		JSONRPC(func() {})
	})
	cliIdentifierHTTPService("en", "show")
	cliIdentifierHTTPService("hCkRaw", "show")
	cliIdentifierHTTPService("hCk", "raw")
	cliIdentifierHTTPService("main", "show")
	cliIdentifierJSONRPCService("de", "show")
	cliIdentifierJSONRPCService("jCkRaw", "show")
	cliIdentifierJSONRPCService("jCk", "raw")
	cliIdentifierGRPCService("c", "show")
	cliIdentifierGRPCService("gCkRaw", "show")
	cliIdentifierGRPCService("gCk", "raw")
	cliIdentifierHTTPGRPCPrimitiveService("fooBar", "baz")
	cliIdentifierHTTPGRPCPrimitiveService("foo", "barBaz")
}

func cliIdentifierHTTPService(serviceName, methodName string) {
	Service(serviceName, func() {
		Method(methodName, func() {
			Result(String)
			HTTP(func() {
				GET("/" + serviceName + "/" + methodName)
			})
		})
	})
}

func cliIdentifierJSONRPCService(serviceName, methodName string) {
	Service(serviceName, func() {
		JSONRPC(func() {
			POST("/rpc/" + serviceName)
		})
		Method(methodName, func() {
			Payload(func() {
				ID("id", String)
				Required("id")
			})
			Result(func() {
				ID("id", String)
				Attribute("value", String)
				Required("id")
			})
			JSONRPC(func() {})
		})
	})
}

func cliIdentifierGRPCService(serviceName, methodName string) {
	Service(serviceName, func() {
		Method(methodName, func() {
			Payload(func() {
				Field(1, "id", String)
				Required("id")
			})
			Result(func() {
				Field(1, "value", String)
			})
			GRPC(func() {})
		})
	})
}

func cliIdentifierHTTPGRPCPrimitiveService(serviceName, methodName string) {
	Service(serviceName, func() {
		Method(methodName, func() {
			Payload(String)
			Result(String)
			HTTP(func() {
				POST("/" + serviceName + "/" + methodName)
			})
			GRPC(func() {})
		})
	})
}
