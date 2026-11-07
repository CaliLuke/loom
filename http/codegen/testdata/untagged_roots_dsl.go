package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// UntaggedRootsDSL exercises the root grammar independently of text bindings.
var UntaggedRootsDSL = func() {
	untaggedRootsDSL(false)
}

// UntaggedRootsJSONRPCDSL uses the same root contracts through JSON-RPC.
var UntaggedRootsJSONRPCDSL = func() {
	untaggedRootsDSL(true)
}

func untaggedRootsDSL(rpc bool) {
	API("rootunion", func() {
		if rpc {
			JSONRPC(func() {
			})
		}
	})
	record, scalar, number, anything, alias := untaggedScalarRootTypes()
	mapping, nested, nullable, widths, typed := untaggedCompositeRootTypes(scalar, record)
	open, envelope, taggedEnvelope, nullableObject, nullableMap := untaggedClosureRootTypes()
	var profileMethods func()
	if !rpc {
		profileMethods = untaggedJSONProfileMethods()
	}
	Service("roots", func() {
		if rpc {
			JSONRPC(func() {
				POST("/rpc")
			})
		}
		for _, entry := range []struct {
			name string
			typ  expr.DataType
		}{
			{"scalar", scalar}, {"number", number}, {"any", anything},
			{"alias", alias}, {"mapping", mapping}, {"nested", nested},
			{"nullable", nullable}, {"widths", widths}, {"typed", typed},
			{"closure", envelope},
			{"taggedclosure", taggedEnvelope},
			{"nullableobject", nullableObject}, {"nullablemap", nullableMap},
		} {
			Method(entry.name, func() {
				Payload(entry.typ)
				Result(entry.typ)
				if rpc {
					JSONRPC(func() {
					})
				} else {
					HTTP(func() {
						POST("/" + entry.name)
					})
				}
			})
		}
		untaggedSelectedRootMethods(open, rpc)
		if !rpc {
			profileMethods()
		}
	})
}

func untaggedScalarRootTypes() (expr.UserType, expr.UserType, expr.UserType, expr.UserType, expr.UserType) {
	code := Type("Code", String, func() {
		Pattern("^[a-z]+$")
	})
	record := Type("Record", func() {
		Attribute("name", String)
		AdditionalAttributes(Int32)
		Required("name", "count")
	})
	scalar := Type("ScalarChoice", OneOf(String, Int), Untagged)
	number := Type("NumberChoice", OneOf(Int, Float64), Untagged)
	anything := Type("AnyChoice", OneOf(Any, String), Untagged)
	alias := Type("AliasChoice", OneOf(code, Int), Untagged)
	return record, scalar, number, anything, alias
}

func untaggedCompositeRootTypes(scalar, record expr.UserType) (expr.UserType, expr.UserType, expr.UserType, expr.UserType, expr.UserType) {
	mapping := Type("MapChoice", OneOf(func() {
		Untagged()
		Attribute("Map", MapOf(String, Int))
		Attribute("Text", String)
	}))
	nested := Type("NestedChoice", OneOf(scalar, Boolean), Untagged)
	nullable := Type("NullableChoice", OneOf(func() {
		Untagged()
		Attribute("Text", String, Nullable)
		Attribute("Number", Int)
	}))
	widths := Type("WidthChoice", OneOf(Int, Int32), Untagged)
	typed := Type("RecordChoice", OneOf(record, Boolean), Untagged)
	return mapping, nested, nullable, widths, typed
}

func untaggedClosureRootTypes() (expr.UserType, expr.UserType, expr.UserType, expr.UserType, expr.UserType) {
	plain := Type("PlainRecord", func() {
		Attribute("x", Int)
		Required("x")
	})
	nullableObject := Type("NullableObjectChoice", OneOf(func() {
		Untagged()
		Attribute("Record", plain, Nullable)
		Attribute("Number", Int)
	}))
	nullableMap := Type("NullableMapChoice", OneOf(func() {
		Untagged()
		Attribute("Map", MapOf(String, Int), Nullable)
		Attribute("Boolean", Boolean)
	}))
	open := Type("OpenChoice", OneOf(plain, String), Untagged)
	envelope := Type("ClosureEnvelope", func() {
		Attribute("open", open)
		Attribute("closed", open, func() {
			Meta("openapi:unevaluatedProperties", "false")
		})
		Required("open", "closed")
	})
	tagged := Type("TaggedChoice", OneOf(func() {
		Attribute("PlainRecord", plain)
		Attribute("String", String)
		Meta("oneof:type:field", "kind")
		Meta("oneof:value:field", "content")
	}))
	taggedEnvelope := Type("TaggedEnvelope", func() {
		Attribute("open", tagged)
		Attribute("closed", tagged, func() {
			Meta("openapi:unevaluatedProperties", "false")
		})
		Required("open", "closed")
	})
	return open, envelope, taggedEnvelope, nullableObject, nullableMap
}

func untaggedSelectedRootMethods(open expr.UserType, rpc bool) {
	for _, name := range []string{"openbody", "closedbody"} {
		Method(name, func() {
			Payload(func() {
				Attribute("value", open, func() {
					if name == "closedbody" {
						Meta("openapi:unevaluatedProperties", "false")
					}
				})
				Required("value")
			})
			if rpc {
				JSONRPC(func() {
				})
			} else {
				HTTP(func() {
					POST("/" + name)
				})
			}
		})
	}
}
