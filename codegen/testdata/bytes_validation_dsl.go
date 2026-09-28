package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// BytesValidationHTTPDSL retains named byte slices across request, response,
// view, parameter and collection boundaries that use distinct Go layouts.
func BytesValidationHTTPDSL() {
	bytesValidationDSL(true, false)
}

// BytesValidationHTTPUnpreservedDSL is the control whose byte aliases are
// normalized to native slices in HTTP bodies. Its service views still retain
// named byte pointers and exercise the same validation defect.
func BytesValidationHTTPUnpreservedDSL() {
	bytesValidationDSL(false, false)
}

// BytesValidationJSONRPCDSL covers the same named-byte validation boundary in
// JSON-RPC request and response bodies.
func BytesValidationJSONRPCDSL() {
	bytesValidationDSL(true, true)
}

// BytesValidationHTTPMinimalDSL is the passing parent control: a byte alias
// without canonical metadata or service views is unwrapped in the request body.
func BytesValidationHTTPMinimalDSL() {
	API("bytesvalidation", func() {})
	blob := Type("Blob", Bytes, func() {
		MinLength(2)
	})
	Service("bytescheck", func() {
		Method("check", func() {
			Payload(func() {
				Attribute("data", blob)
				Required("data")
			})
			HTTP(func() {
				POST("/check")
			})
		})
	})
}

func bytesValidationDSL(canonical, rpc bool) {
	API("bytesvalidation", func() {
		if rpc {
			JSONRPC(func() {})
		}
	})
	blob := Type("Blob", Bytes, func() {
		MinLength(2)
		MaxLength(3)
		if canonical {
			Meta("openapi:typename:canonical", "true")
		}
	})
	chain := Type("BlobAlias", blob)
	choice := Type("ByteChoice", Bytes, func() {
		Enum([]byte("hi"), []byte("hey"))
		if canonical {
			Meta("openapi:typename:canonical", "true")
		}
	})
	label := Type("Label", String, func() {
		MinLength(2)
	})
	arbitrary := Type("Arbitrary", Any)
	result := bytesValidationResult(blob)
	Service("bytescheck", func() {
		if rpc {
			JSONRPC(func() {
				POST("/rpc")
			})
		}
		Method("check", func() {
			Payload(bytesValidationPayload(blob, chain, choice, label, arbitrary, rpc))
			Result(result)
			if rpc {
				JSONRPC(func() {})
			} else {
				HTTP(func() {
					POST("/check")
				})
			}
		})
		if !rpc {
			Method("locations", func() {
				Payload(func() {
					Attribute("data", blob)
					Attribute("query", blob)
					Attribute("header", blob)
					Required("data")
				})
				HTTP(func() {
					POST("/locations")
					Param("query")
					Header("header:X-Blob")
				})
			})
		}
	})
}

func bytesValidationResult(blob expr.DataType) *expr.ResultTypeExpr {
	return ResultType("application/vnd.bytevalidation", func() {
		Attributes(func() {
			Attribute("data", blob)
			Required("data")
		})
		View("default", func() {
			Attribute("data")
		})
		View("tiny", func() {
			Attribute("data")
		})
	})
}

func bytesValidationPayload(blob, chain, choice, label, arbitrary expr.DataType, rpc bool) func() {
	return func() {
		if rpc {
			ID("id", String)
		}
		Attribute("data", blob)
		Attribute("optional", blob)
		Attribute("defaulted", String, func() {
			// Byte defaults exercise a separate transformer defect (#577).
			// Their validation layouts remain covered in codegen tests.
			Default("hi")
		})
		Attribute("chain", chain)
		Attribute("choice", choice)
		Attribute("items", ArrayOf(blob))
		Attribute("lookup", MapOf(String, blob))
		Attribute("nullable", blob, func() {
			Nullable()
		})
		Attribute("label", label)
		Attribute("anything", Any)
		Attribute("named_anything", arbitrary)
		Attribute("union", OneOf(blob, label))
		Required("data")
	}
}
