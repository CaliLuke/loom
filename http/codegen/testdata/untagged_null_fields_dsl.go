package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// UntaggedNullFieldsDSL covers union-owned null and enclosing field presence.
func UntaggedNullFieldsDSL() {
	untaggedNullFieldsDSL(false)
}

// UntaggedNullFieldsJSONRPCDSL uses the same presence contract over JSON-RPC.
func UntaggedNullFieldsJSONRPCDSL() {
	untaggedNullFieldsDSL(true)
}

func untaggedNullFieldsDSL(rpc bool) {
	API("nullfields", func() {
		if rpc {
			JSONRPC(func() {
			})
		}
	})
	anyChoice := Type("AnyChoice", OneOf(Any, Boolean), Untagged)
	text := Type("NullableText", String, Nullable)
	nullableChoice := Type("NullableChoice", OneOf(text, Boolean), Untagged)
	ambiguous := Type("AmbiguousChoice", OneOf(Any, text), Untagged)
	plain := Type("PlainChoice", OneOf(String, Int), Untagged)
	record := Type("Record", func() {
		Attribute("required_any", anyChoice)
		Attribute("optional_any", anyChoice)
		Attribute("required_nullable", nullableChoice)
		Attribute("optional_nullable", nullableChoice)
		Attribute("list", ArrayOf(anyChoice))
		Attribute("map", MapOf(String, nullableChoice))
		Attribute("outer", plain, Nullable)
		Required("required_any", "required_nullable")
	})
	Service("nullfields", func() {
		if rpc {
			JSONRPC(func() {
				POST("/rpc")
			})
		}
		untaggedNullFieldMethod("echo", "/echo", record, rpc)
		untaggedNullFieldMethod("root_any", "/root-any", anyChoice, rpc)
		untaggedNullFieldMethod("root_nullable", "/root-nullable", nullableChoice, rpc)
		untaggedNullFieldMethod("ambiguous", "/ambiguous", ambiguous, rpc)
	})
}

func untaggedNullFieldMethod(name, path string, value expr.UserType, rpc bool) {
	Method(name, func() {
		Payload(value)
		Result(value)
		if rpc {
			JSONRPC(func() {
			})
		} else {
			HTTP(func() {
				POST(path)
				Response(StatusOK)
			})
		}
	})
}
