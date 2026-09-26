package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// MappedNamesDSL declares payload and result attributes with a transport
// element name suffix, such as "n:m", in a type, a nested type, unions and an
// inline payload: optional, required and defaulted primitives, a validated
// string, an object, an array, a map, a constructor OneOf and a OneOf block
// with suffixed branches, and an untagged union whose branch fields are
// suffixed. The HTTP bodies, the OpenAPI schemas and examples and the CLI
// body examples use the suffix as the JSON name of the field.
var MappedNamesDSL = func() {
	var Leaf = Type("Leaf", func() {
		Attribute("leaf:l", String)
		Attribute("count:c", Int)
		Required("count:c")
	})
	var Envelope = Type("Envelope", func() {
		Attribute("n:m", String, func() {
			MinLength(2)
		})
		Attribute("req:r", Int)
		Attribute("def:d", Int, func() {
			Default(3)
		})
		Attribute("pick:p", OneOf(String, Int))
		Attribute("obj:o", Leaf)
		Attribute("list:ls", ArrayOf(String))
		Attribute("index:ix", MapOf(String, Leaf))
		OneOf("choice:ch", func() {
			Attribute("text:t", String)
			Attribute("leaf_branch:lb", Leaf)
		})
		Required("req:r", "pick:p", "obj:o")
	})
	var DataResult = Type("DataResult", func() {
		Attribute("data:dt", String)
		Attribute("size:sz", Int)
		Required("data:dt")
	})
	var FailResult = Type("FailResult", func() {
		Attribute("reason:rs", String)
		Required("reason:rs")
	})
	Service("mappednames", func() {
		Method("echo", func() {
			NoSecurity()
			Payload(Envelope)
			Result(Envelope)
			HTTP(func() {
				POST("/echo")
			})
		})
		Method("inline", func() {
			NoSecurity()
			Payload(func() {
				Attribute("id:i", String)
				Attribute("count:c", Int)
				Required("id:i")
			})
			Result(func() {
				Attribute("id:i", String)
				Attribute("count:c", Int)
				Required("id:i")
			})
			HTTP(func() {
				PUT("/inline")
			})
		})
		Method("lookup", func() {
			NoSecurity()
			Payload(OneOf(DataResult, FailResult), func() {
				Untagged()
			})
			Result(OneOf(DataResult, FailResult), func() {
				Untagged()
			})
			HTTP(func() {
				POST("/lookup")
			})
		})
	})
}

// MappedParamsDSL declares path and query parameters with a transport element
// name suffix: Param("key:k") with the route wildcard "{k}" is the path
// parameter of the key attribute, an implicit route parameter keeps its name,
// and suffixed params absent from the route are query parameters named after
// their element names.
var MappedParamsDSL = func() {
	Service("mappedparams", func() {
		Method("show", func() {
			NoSecurity()
			Payload(func() {
				Attribute("key", String, func() {
					MinLength(2)
				})
				Attribute("id", Int)
				Attribute("q", String)
				Attribute("flag", Boolean)
				Required("key", "id")
			})
			Result(func() {
				Attribute("key", String)
				Attribute("id", Int)
				Attribute("q", String)
				Attribute("flag", Boolean)
				Required("key", "id")
			})
			HTTP(func() {
				GET("/items/{k}/{id}")
				Param("key:k")
				Param("q:query")
				Param("flag:f")
			})
		})
	})
}

// MappedExplicitBodyDSL declares explicit request and response bodies whose
// attributes carry a transport element name suffix, such as
// Attribute("name:n") in Body. The body attributes inherit the type,
// description, validations and requiredness of the payload and result
// attributes of the same attribute names, and the bodies use the suffixes as
// the JSON names of the fields. The create method maps a user type payload
// and an inline result. The pair method lists body attributes of an inline
// payload that are not strings and requires one of them in the body.
var MappedExplicitBodyDSL = func() {
	var Account = Type("Account", func() {
		Attribute("id", Int, "Account ID")
		Attribute("name", String, "Account name", func() {
			MinLength(2)
		})
		Attribute("age", Int, "Account age")
		Required("id", "name")
	})
	Service("mappedbody", func() {
		Method("create", func() {
			NoSecurity()
			Payload(Account)
			Result(func() {
				Attribute("id", Int, "Account ID")
				Attribute("name", String, "Account name", func() {
					MinLength(2)
				})
				Attribute("age", Int, "Account age")
				Required("name")
			})
			HTTP(func() {
				POST("/accounts/{id}")
				Body(func() {
					Attribute("name:n")
					Attribute("age:ag")
				})
				Response(StatusOK, func() {
					Body(func() {
						Attribute("name:n")
						Attribute("age:ag")
					})
				})
			})
		})
		Method("pair", func() {
			NoSecurity()
			Payload(func() {
				Attribute("a", Int)
				Attribute("b", Int)
				Required("a")
			})
			HTTP(func() {
				POST("/pair")
				Body(func() {
					Attribute("a:x")
					Attribute("b:y")
					Required("a:x")
				})
			})
		})
	})
}

// MappedPayloadReferencesDSL maps payload, result and error attributes
// declared with a transport element name suffix, such as "tok:t", to
// headers, params, cookies, route wildcards, MapParams and bodies by their
// attribute names. The mapping names the transport element: Header("tok")
// reads the "tok" header, and the suffix names only the field of a body.
var MappedPayloadReferencesDSL = func() {
	var Fault = Type("Fault", func() {
		Attribute("code:c", String)
		Attribute("detail:dt", String)
		Required("code:c")
	})
	Service("mappedrefs", func() {
		Method("send", func() {
			NoSecurity()
			Payload(func() {
				Attribute("id:i", Int)
				Attribute("tok:t", String, func() {
					MinLength(2)
				})
				Attribute("ver:v", String)
				Attribute("q:qq", String)
				Attribute("sid:s", String)
				Attribute("name:nm", String)
				Required("id:i", "tok:t")
			})
			Result(func() {
				Attribute("loc:l", String)
				Attribute("sid:s", String)
				Attribute("name:nm", String)
				Required("loc:l")
			})
			Error("bad", Fault)
			HTTP(func() {
				POST("/items/{id}")
				Header("tok")
				Header("ver:X-Version")
				Param("q")
				Cookie("sid")
				Response(StatusOK, func() {
					Header("loc:Location")
					Cookie("sid")
				})
				Response("bad", StatusBadRequest, func() {
					Header("code:X-Code")
				})
			})
		})
		Method("put", func() {
			NoSecurity()
			Payload(func() {
				Attribute("id:i", Int)
				Attribute("data:d", MapOf(String, Int))
				Required("data:d")
			})
			Result(func() {
				Attribute("data:d", ArrayOf(String))
				Required("data:d")
			})
			HTTP(func() {
				PUT("/data/{id}")
				Body("data")
				Response(StatusOK, func() {
					Body("data")
				})
			})
		})
		Method("rename", func() {
			NoSecurity()
			Payload(func() {
				Attribute("name:nm", String, func() {
					MinLength(2)
				})
				Required("name:nm")
			})
			HTTP(func() {
				POST("/rename")
				Body(func() {
					Attribute("name")
					Required("name")
				})
			})
		})
		Method("filter", func() {
			NoSecurity()
			Payload(func() {
				Attribute("filters:f", MapOf(String, String))
			})
			HTTP(func() {
				GET("/filter")
				MapParams("filters")
			})
		})
	})
}
