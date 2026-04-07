package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// Result(Header|Body)(Type)(Required|Default)?DSL
//
// Where Type is the type of the result or result.


var ResultHeaderBoolDSL = func() {
	Service("ServiceHeaderBool", func() {
		Method("MethodHeaderBool", func() {
			Result(func() {
				Attribute("h", Boolean)
			})
			HTTP(func() {
				GET("/")
				Response(StatusOK, func() {
					Header("h")
				})
			})
		})
	})
}

var ResultHeaderIntDSL = func() {
	Service("ServiceHeaderInt", func() {
		Method("MethodHeaderInt", func() {
			Result(func() {
				Attribute("h", Int)
			})
			HTTP(func() {
				GET("/")
				Response(StatusOK, func() {
					Header("h")
				})
			})
		})
	})
}

var ResultHeaderInt32DSL = func() {
	Service("ServiceHeaderInt32", func() {
		Method("MethodHeaderInt32", func() {
			Result(func() {
				Attribute("h", Int32)
			})
			HTTP(func() {
				GET("/")
				Response(StatusOK, func() {
					Header("h")
				})
			})
		})
	})
}

var ResultHeaderInt64DSL = func() {
	Service("ServiceHeaderInt64", func() {
		Method("MethodHeaderInt64", func() {
			Result(func() {
				Attribute("h", Int64)
			})
			HTTP(func() {
				GET("/")
				Response(StatusOK, func() {
					Header("h")
				})
			})
		})
	})
}

var ResultHeaderUIntDSL = func() {
	Service("ServiceHeaderUInt", func() {
		Method("MethodHeaderUInt", func() {
			Result(func() {
				Attribute("h", UInt)
			})
			HTTP(func() {
				GET("/")
				Response(StatusOK, func() {
					Header("h")
				})
			})
		})
	})
}

var ResultHeaderUInt32DSL = func() {
	Service("ServiceHeaderUInt32", func() {
		Method("MethodHeaderUInt32", func() {
			Result(func() {
				Attribute("h", UInt32)
			})
			HTTP(func() {
				GET("/")
				Response(StatusOK, func() {
					Header("h")
				})
			})
		})
	})
}

var ResultHeaderUInt64DSL = func() {
	Service("ServiceHeaderUInt64", func() {
		Method("MethodHeaderUInt64", func() {
			Result(func() {
				Attribute("h", UInt64)
			})
			HTTP(func() {
				GET("/")
				Response(StatusOK, func() {
					Header("h")
				})
			})
		})
	})
}

var ResultHeaderFloat32DSL = func() {
	Service("ServiceHeaderFloat32", func() {
		Method("MethodHeaderFloat32", func() {
			Result(func() {
				Attribute("h", Float32)
			})
			HTTP(func() {
				GET("/")
				Response(StatusOK, func() {
					Header("h")
				})
			})
		})
	})
}

var ResultHeaderFloat64DSL = func() {
	Service("ServiceHeaderFloat64", func() {
		Method("MethodHeaderFloat64", func() {
			Result(func() {
				Attribute("h", Float64)
			})
			HTTP(func() {
				GET("/")
				Response(StatusOK, func() {
					Header("h")
				})
			})
		})
	})
}

var ResultHeaderStringDSL = func() {
	Service("ServiceHeaderString", func() {
		Method("MethodHeaderString", func() {
			Result(func() {
				Attribute("h", String)
			})
			HTTP(func() {
				GET("/")
				Response(StatusOK, func() {
					Header("h")
				})
			})
		})
	})
}

var ResultHeaderBytesDSL = func() {
	Service("ServiceHeaderBytes", func() {
		Method("MethodHeaderBytes", func() {
			Result(func() {
				Attribute("h", Bytes)
			})
			HTTP(func() {
				GET("/")
				Response(StatusOK, func() {
					Header("h")
				})
			})
		})
	})
}

var ResultHeaderAnyDSL = func() {
	Service("ServiceHeaderAny", func() {
		Method("MethodHeaderAny", func() {
			Result(func() {
				Attribute("h", Any)
			})
			HTTP(func() {
				GET("/")
				Response(StatusOK, func() {
					Header("h")
				})
			})
		})
	})
}

var ResultHeaderArrayBoolDSL = func() {
	Service("ServiceHeaderArrayBool", func() {
		Method("MethodHeaderArrayBool", func() {
			Result(func() {
				Attribute("h", ArrayOf(Boolean))
			})
			HTTP(func() {
				GET("/")
				Response(StatusOK, func() {
					Header("h")
				})
			})
		})
	})
}

var ResultHeaderArrayIntDSL = func() {
	Service("ServiceHeaderArrayInt", func() {
		Method("MethodHeaderArrayInt", func() {
			Result(func() {
				Attribute("h", ArrayOf(Int))
			})
			HTTP(func() {
				GET("/")
				Response(StatusOK, func() {
					Header("h")
				})
			})
		})
	})
}

var ResultHeaderArrayInt32DSL = func() {
	Service("ServiceHeaderArrayInt32", func() {
		Method("MethodHeaderArrayInt32", func() {
			Result(func() {
				Attribute("h", ArrayOf(Int32))
			})
			HTTP(func() {
				GET("/")
				Response(StatusOK, func() {
					Header("h")
				})
			})
		})
	})
}

var ResultHeaderArrayInt64DSL = func() {
	Service("ServiceHeaderArrayInt64", func() {
		Method("MethodHeaderArrayInt64", func() {
			Result(func() {
				Attribute("h", ArrayOf(Int64))
			})
			HTTP(func() {
				GET("/")
				Response(StatusOK, func() {
					Header("h")
				})
			})
		})
	})
}

var ResultHeaderArrayUIntDSL = func() {
	Service("ServiceHeaderArrayUInt", func() {
		Method("MethodHeaderArrayUInt", func() {
			Result(func() {
				Attribute("h", ArrayOf(UInt))
			})
			HTTP(func() {
				GET("/")
				Response(StatusOK, func() {
					Header("h")
				})
			})
		})
	})
}

var ResultHeaderArrayUInt32DSL = func() {
	Service("ServiceHeaderArrayUInt32", func() {
		Method("MethodHeaderArrayUInt32", func() {
			Result(func() {
				Attribute("h", ArrayOf(UInt32))
			})
			HTTP(func() {
				GET("/")
				Response(StatusOK, func() {
					Header("h")
				})
			})
		})
	})
}

var ResultHeaderArrayUInt64DSL = func() {
	Service("ServiceHeaderArrayUInt64", func() {
		Method("MethodHeaderArrayUInt64", func() {
			Result(func() {
				Attribute("h", ArrayOf(UInt64))
			})
			HTTP(func() {
				GET("/")
				Response(StatusOK, func() {
					Header("h")
				})
			})
		})
	})
}

var ResultHeaderArrayFloat32DSL = func() {
	Service("ServiceHeaderArrayFloat32", func() {
		Method("MethodHeaderArrayFloat32", func() {
			Result(func() {
				Attribute("h", ArrayOf(Float32))
			})
			HTTP(func() {
				GET("/")
				Response(StatusOK, func() {
					Header("h")
				})
			})
		})
	})
}

var ResultHeaderArrayFloat64DSL = func() {
	Service("ServiceHeaderArrayFloat64", func() {
		Method("MethodHeaderArrayFloat64", func() {
			Result(func() {
				Attribute("h", ArrayOf(Float64))
			})
			HTTP(func() {
				GET("/")
				Response(StatusOK, func() {
					Header("h")
				})
			})
		})
	})
}

var ResultHeaderArrayStringDSL = func() {
	Service("ServiceHeaderArrayString", func() {
		Method("MethodHeaderArrayString", func() {
			Result(func() {
				Attribute("h", ArrayOf(String))
			})
			HTTP(func() {
				GET("/")
				Response(StatusOK, func() {
					Header("h")
				})
			})
		})
	})
}

var ResultHeaderArrayBytesDSL = func() {
	Service("ServiceHeaderArrayBytes", func() {
		Method("MethodHeaderArrayBytes", func() {
			Result(func() {
				Attribute("h", ArrayOf(Bytes))
			})
			HTTP(func() {
				GET("/")
				Response(StatusOK, func() {
					Header("h")
				})
			})
		})
	})
}

var ResultHeaderArrayAnyDSL = func() {
	Service("ServiceHeaderArrayAny", func() {
		Method("MethodHeaderArrayAny", func() {
			Result(func() {
				Attribute("h", ArrayOf(Any))
			})
			HTTP(func() {
				GET("/")
				Response(StatusOK, func() {
					Header("h")
				})
			})
		})
	})
}

var ResultHeaderBoolDefaultDSL = func() {
	Service("ServiceHeaderBoolDefault", func() {
		Method("MethodHeaderBoolDefault", func() {
			Result(func() {
				Attribute("h", Boolean, func() {
					Default(true)
				})
			})
			HTTP(func() {
				GET("/")
				Response(StatusOK, func() {
					Header("h")
				})
			})
		})
	})
}

var ResultHeaderBoolRequiredDefaultDSL = func() {
	Service("ServiceHeaderBoolRequiredDefault", func() {
		Method("MethodHeaderBoolRequiredDefault", func() {
			Result(func() {
				Attribute("h", Boolean, func() {
					Default(true)
				})
				Required("h")
			})
			HTTP(func() {
				GET("/")
				Response(StatusOK, func() {
					Header("h")
				})
			})
		})
	})
}

var ResultHeaderStringDefaultDSL = func() {
	Service("ServiceHeaderStringDefault", func() {
		Method("MethodHeaderStringDefault", func() {
			Result(func() {
				Attribute("h", func() {
					Default("def")
				})
			})
			HTTP(func() {
				GET("/")
				Response(StatusOK, func() {
					Header("h")
				})
			})
		})
	})
}

var ResultHeaderStringRequiredDefaultDSL = func() {
	Service("ServiceHeaderStringRequiredDefault", func() {
		Method("MethodHeaderStringRequiredDefault", func() {
			Result(func() {
				Attribute("h", func() {
					Default("def")
				})
				Required("h")
			})
			HTTP(func() {
				GET("/")
				Response(StatusOK, func() {
					Header("h")
				})
			})
		})
	})
}

var ResultHeaderArrayBoolDefaultDSL = func() {
	Service("ServiceHeaderArrayBoolDefault", func() {
		Method("MethodHeaderArrayBoolDefault", func() {
			Result(func() {
				Attribute("h", ArrayOf(Boolean), func() {
					Default([]bool{true, false})
				})
			})
			HTTP(func() {
				GET("/")
				Response(StatusOK, func() {
					Header("h")
				})
			})
		})
	})
}

var ResultHeaderArrayBoolRequiredDefaultDSL = func() {
	Service("ServiceHeaderArrayBoolRequiredDefault", func() {
		Method("MethodHeaderArrayBoolRequiredDefault", func() {
			Result(func() {
				Attribute("h", ArrayOf(Boolean), func() {
					Default([]bool{true, false})
				})
				Required("h")
			})
			HTTP(func() {
				GET("/")
				Response(StatusOK, func() {
					Header("h")
				})
			})
		})
	})
}

var ResultHeaderArrayStringDefaultDSL = func() {
	Service("ServiceHeaderArrayStringDefault", func() {
		Method("MethodHeaderArrayStringDefault", func() {
			Result(func() {
				Attribute("h", ArrayOf(String), func() {
					Default([]string{"foo", "bar"})
				})
			})
			HTTP(func() {
				GET("/")
				Response(StatusOK, func() {
					Header("h")
				})
			})
		})
	})
}

var ResultHeaderArrayStringRequiredDefaultDSL = func() {
	Service("ServiceHeaderArrayStringRequiredDefault", func() {
		Method("MethodHeaderArrayStringRequiredDefault", func() {
			Result(func() {
				Attribute("h", ArrayOf(String), func() {
					Default([]string{"foo", "bar"})
				})
				Required("h")
			})
			HTTP(func() {
				GET("/")
				Response(StatusOK, func() {
					Header("h")
				})
			})
		})
	})
}

var ResultBodyStringDSL = func() {
	Service("ServiceBodyString", func() {
		Method("MethodBodyString", func() {
			Result(func() {
				Attribute("b", String)
			})
			HTTP(func() {
				POST("/")
			})
		})
	})
}

var ResultBodyObjectDSL = func() {
	Service("ServiceBodyObject", func() {
		Method("MethodBodyObject", func() {
			Result(func() {
				Attribute("b", String)
			})
			HTTP(func() {
				POST("/")
			})
		})
	})
}

var ResultBodyObjectHeaderDSL = func() {
	Service("ServiceBodyObjectHeader", func() {
		Method("MethodBodyObjectHeader", func() {
			Result(func() {
				Attribute("a", String)
				Attribute("b", String)
			})
			HTTP(func() {
				POST("/")
				Response(StatusOK, func() {
					Header("b:Authorization")
				})
			})
		})
	})
}

var ResultBodyUserRequiredDSL = func() {
	var Bod = Type("body", func() {
		Attribute("a")
		Required("a")
	})
	Service("ServiceBodyUserRequired", func() {
		Method("MethodBodyUserRequired", func() {
			Result(func() {
				Attribute("body", Bod)
			})
			HTTP(func() {
				GET("/")
				Response(StatusOK, func() {
					Body("body")
				})
			})
		})
	})
}

var ResultBodyUserDSL = func() {
	var ResultType = Type("ResultType", func() {
		Attribute("a", String)
	})
	Service("ServiceBodyUser", func() {
		Method("MethodBodyUser", func() {
			Result(ResultType)
			HTTP(func() {
				POST("/")
			})
		})
	})
}

var ResultBodyUnionDSL = func() {
	var Union = Type("Union", func() {
		OneOf("Vals", func() {
			Attribute("String", String)
			Attribute("Int", Int)
		})
	})
	Service("ServiceBodyUnion", func() {
		Method("MethodBodyUnion", func() {
			Result(Union)
			HTTP(func() {
				POST("/")
			})
		})
	})
}

var ResultTypeValidateDSL = func() {
	var ResultType = Type("ResultType", func() {
		Attribute("a", String, func() {
			MinLength(5)
		})
	})
	Service("ServiceResultTypeValidate", func() {
		Method("MethodResultTypeValidate", func() {
			Result(ResultType)
			HTTP(func() {
				POST("/")
			})
		})
	})
}

var ResultBodyMultipleViewsDSL = func() {
	var ResultType = ResultType("ResultTypeMultipleViews", func() {
		Attribute("a", String)
		Attribute("b", String)
		Attribute("c", String)
		View("default", func() {
			Attribute("a")
			Attribute("b")
			Attribute("c")
		})
		View("tiny", func() {
			Attribute("c")
		})
	})
	Service("ServiceBodyMultipleView", func() {
		Method("MethodBodyMultipleView", func() {
			Result(ResultType)
			HTTP(func() {
				POST("/")
				Response(StatusOK, func() {
					Header("c:Location")
				})
			})
		})
	})
}

var ResultBodyCollectionDSL = func() {
	var RT = ResultType("ResultTypeCollection", func() {
		Attributes(func() {
			Attribute("a", String)
			Attribute("b", String)
			Attribute("c", String)
		})
		View("default", func() {
			Attribute("a")
			Attribute("b")
			Attribute("c")
		})
		View("tiny", func() {
			Attribute("c")
		})
	})
	Service("ServiceBodyCollection", func() {
		Method("MethodBodyCollection", func() {
			Result(CollectionOf(RT))
			HTTP(func() {
				POST("/")
				Response(StatusOK)
			})
		})
	})
}

var ResultBodyCollectionExplicitViewDSL = func() {
	var RT = ResultType("ResultTypeCollection", func() {
		Attributes(func() {
			Attribute("a", String)
			Attribute("b", String)
			Attribute("c", String)
		})
		View("default", func() {
			Attribute("a")
			Attribute("b")
			Attribute("c")
		})
		View("tiny", func() {
			Attribute("c")
		})
	})
	Service("ServiceBodyCollectionExplicitView", func() {
		Method("MethodBodyCollectionExplicitView", func() {
			Result(CollectionOf(RT), func() {
				View("tiny")
			})
			HTTP(func() {
				POST("/")
				Response(StatusOK)
			})
		})
	})
}

var ResultWithResultCollectionDSL = func() {
	var RT = ResultType("RT", func() {
		Attributes(func() {
			Attribute("x", String, func() {
				MinLength(5)
			})
		})
	})
	var ResultType = ResultType("ResultType", func() {
		Attributes(func() {
			Attribute("x", CollectionOf(RT))
		})
	})
	Service("ServiceResultWithResultCollection", func() {
		Method("MethodResultWithResultCollection", func() {
			Result(func() {
				Attribute("a", ResultType)
			})
			HTTP(func() {
				POST("/")
				Response(StatusOK)
			})
		})
	})
}

var ResultWithCustomPkgTypeDSL = func() {
	var Foo = Type("Foo", func() {
		Meta("struct:pkg:path", "foo")
		Attribute("bar", String)
	})

	Service("ServiceResultWithCustomPkgTypeDSL", func() {
		Method("MethodResultWithCustomPkgTypeDSL", func() {
			Payload(Foo)
			Result(Foo)

			HTTP(func() {
				GET("/")
			})
		})
	})
}

var EmbeddedCustomPkgTypeDSL = func() {
	var Foo = Type("Foo", func() {
		Meta("struct:pkg:path", "foo")
		Attribute("bar", String)
	})

	var ContainedFoo = Type("ContainedFoo", func() {
		Attribute("Foo", Foo)
	})

	Service("ServiceResultWithEmbeddedCustomPkgTypeDSL", func() {
		Method("MethodResultWithEmbeddedCustomPkgTypeDSL", func() {
			Payload(ContainedFoo)
			Result(ContainedFoo)

			HTTP(func() {
				GET("/")
			})
		})
	})
}

var ArrayAliasExtendedDSL = func() {
	var Foo = Type("Foo", String)

	var Extension = Type("Extension", func() {
		Attribute("Foo", Foo)
	})

	var ResultType = Type("ResultType", func() {
		Extend(Extension)
	})

	var _ = Service("FooService", func() {
		Method("FooMethod", func() {
			Payload(ArrayOf(ResultType))
			Result(ArrayOf(ResultType))
			HTTP(func() {
				GET("/")
			})
		})
	})

}

var ExtensionWithAliasDSL = func() {
	var Bar = Type("Bar", func() {
		Attribute("Bar", UInt)
		Required("Bar")
	})

	var TypeWithAlias = Type("TypeWithAlias", func() {
		Attribute("Bar", Bar)
	})

	var Extension = Type("Extension", func() {
		Extend(TypeWithAlias)
	})

	var ResultType = Type("ResultType", func() {
		Attribute("Extension", Extension)
	})

	var _ = Service("FooService", func() {
		Method("FooMethod", func() {
			Payload(ArrayOf(ResultType))
			Result(ArrayOf(ResultType))
			HTTP(func() {
				GET("/")
			})
		})
	})
}

var EmptyErrorResponseBodyDSL = func() {
	Service("ServiceEmptyErrorResponseBody", func() {
		Method("MethodEmptyErrorResponseBody", func() {
			Error("internal_error")
			Error("not_found", String)
			HTTP(func() {
				HEAD("/")
				Response(StatusOK)
				Response("internal_error", StatusInternalServerError, func() {
					Body(Empty)
					Header("code:Error-Code")
				})
				Response("not_found", StatusNotFound, func() {
					Body(Empty)
					Header("in-header")
				})
			})
		})
	})
}

var WithErrorCustomPkgDSL = func() {
	var CustomError = Type("CustomError", func() {
		Meta("struct:pkg:path", "custom")
		ErrorName("name")
		Required("name")
	})
	Service("ServiceWithErrorCustomPkg", func() {
		Method("MethodWithErrorCustomPkg", func() {
			Error("error_name", CustomError)
			HTTP(func() {
				GET("/")
				Response("error_name", StatusBadRequest)
			})
		})
	})
}

var EmptyCustomErrorResponseBodyDSL = func() {
	var ErrorType = Type("Error", func() {
		Attribute("err", String)
	})
	Service("ServiceEmptyCustomErrorResponseBody", func() {
		Method("MethodEmptyCustomErrorResponseBody", func() {
			Error("internal_error", ErrorType)
			HTTP(func() {
				HEAD("/")
				Response(StatusOK)
				Response("internal_error", StatusInternalServerError, func() {
					Body(Empty)
				})
			})
		})
	})
}

var ResultWithResultViewDSL = func() {
	var RT = ResultType("RT", func() {
		Attributes(func() {
			Attribute("x")
		})
	})
	var ResultType = ResultType("ResultType", func() {
		Attributes(func() {
			Attribute("name")
			Attribute("rt", RT)
		})
		View("full", func() {
			Attribute("name")
			Attribute("rt")
		})
		View("default", func() {
			Attribute("name")
		})
	})
	Service("ServiceResultWithResultView", func() {
		Method("MethodResultWithResultView", func() {
			Result(ResultType, func() {
				View("full")
			})
			HTTP(func() {
				POST("/")
				Response(StatusOK)
			})
		})
	})
}


