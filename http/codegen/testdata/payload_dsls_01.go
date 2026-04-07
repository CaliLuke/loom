package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadQueryBoolDSL = func() {
	Service("ServiceQueryBool", func() {
		Method("MethodQueryBool", func() {
			Payload(func() {
				Attribute("q", Boolean)
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryBoolValidateDSL = func() {
	Service("ServiceQueryBoolValidate", func() {
		Method("MethodQueryBoolValidate", func() {
			Payload(func() {
				Attribute("q", Boolean, func() {
					Enum(true)
				})
				Required("q")
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryIntDSL = func() {
	Service("ServiceQueryInt", func() {
		Method("MethodQueryInt", func() {
			Payload(func() {
				Attribute("q", Int)
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryIntValidateDSL = func() {
	Service("ServiceQueryIntValidate", func() {
		Method("MethodQueryIntValidate", func() {
			Payload(func() {
				Attribute("q", Int, func() {
					Minimum(1)
				})
				Required("q")
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryInt32DSL = func() {
	Service("ServiceQueryInt32", func() {
		Method("MethodQueryInt32", func() {
			Payload(func() {
				Attribute("q", Int32)
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryInt32ValidateDSL = func() {
	Service("ServiceQueryInt32Validate", func() {
		Method("MethodQueryInt32Validate", func() {
			Payload(func() {
				Attribute("q", Int32, func() {
					Minimum(1)
				})
				Required("q")
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryInt64DSL = func() {
	Service("ServiceQueryInt64", func() {
		Method("MethodQueryInt64", func() {
			Payload(func() {
				Attribute("q", Int64)
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryInt64ValidateDSL = func() {
	Service("ServiceQueryInt64Validate", func() {
		Method("MethodQueryInt64Validate", func() {
			Payload(func() {
				Attribute("q", Int64, func() {
					Minimum(1)
				})
				Required("q")
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryUIntDSL = func() {
	Service("ServiceQueryUInt", func() {
		Method("MethodQueryUInt", func() {
			Payload(func() {
				Attribute("q", UInt)
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryUIntValidateDSL = func() {
	Service("ServiceQueryUIntValidate", func() {
		Method("MethodQueryUIntValidate", func() {
			Payload(func() {
				Attribute("q", UInt, func() {
					Minimum(1)
				})
				Required("q")
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryUInt32DSL = func() {
	Service("ServiceQueryUInt32", func() {
		Method("MethodQueryUInt32", func() {
			Payload(func() {
				Attribute("q", UInt32)
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryUInt32ValidateDSL = func() {
	Service("ServiceQueryUInt32Validate", func() {
		Method("MethodQueryUInt32Validate", func() {
			Payload(func() {
				Attribute("q", UInt32, func() {
					Minimum(1)
				})
				Required("q")
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryUInt64DSL = func() {
	Service("ServiceQueryUInt64", func() {
		Method("MethodQueryUInt64", func() {
			Payload(func() {
				Attribute("q", UInt64)
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryUInt64ValidateDSL = func() {
	Service("ServiceQueryUInt64Validate", func() {
		Method("MethodQueryUInt64Validate", func() {
			Payload(func() {
				Attribute("q", UInt64, func() {
					Minimum(1)
				})
				Required("q")
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryFloat32DSL = func() {
	Service("ServiceQueryFloat32", func() {
		Method("MethodQueryFloat32", func() {
			Payload(func() {
				Attribute("q", Float32)
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryFloat32ValidateDSL = func() {
	Service("ServiceQueryFloat32Validate", func() {
		Method("MethodQueryFloat32Validate", func() {
			Payload(func() {
				Attribute("q", Float32, func() {
					Minimum(1)
				})
				Required("q")
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryFloat64DSL = func() {
	Service("ServiceQueryFloat64", func() {
		Method("MethodQueryFloat64", func() {
			Payload(func() {
				Attribute("q", Float64)
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryFloat64ValidateDSL = func() {
	Service("ServiceQueryFloat64Validate", func() {
		Method("MethodQueryFloat64Validate", func() {
			Payload(func() {
				Attribute("q", Float64, func() {
					Minimum(1)
				})
				Required("q")
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryStringDSL = func() {
	Service("ServiceQueryString", func() {
		Method("MethodQueryString", func() {
			Payload(func() {
				Attribute("q", String)
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryStringValidateDSL = func() {
	Service("ServiceQueryStringValidate", func() {
		Method("MethodQueryStringValidate", func() {
			Payload(func() {
				Attribute("q", String, func() {
					Enum("val")
				})
				Required("q")
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryStringNotRequiredValidateDSL = func() {
	Service("ServiceQueryStringNotRequiredValidate", func() {
		Method("MethodQueryStringNotRequiredValidate", func() {
			Payload(func() {
				Attribute("q", String, func() {
					Enum("val")
				})
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryBytesDSL = func() {
	Service("ServiceQueryBytes", func() {
		Method("MethodQueryBytes", func() {
			Payload(func() {
				Attribute("q", Bytes)
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryBytesValidateDSL = func() {
	Service("ServiceQueryBytesValidate", func() {
		Method("MethodQueryBytesValidate", func() {
			Payload(func() {
				Attribute("q", Bytes, func() {
					MinLength(1)
				})
				Required("q")
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryAnyDSL = func() {
	Service("ServiceQueryAny", func() {
		Method("MethodQueryAny", func() {
			Payload(func() {
				Attribute("q", Any)
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryAnyValidateDSL = func() {
	Service("ServiceQueryAnyValidate", func() {
		Method("MethodQueryAnyValidate", func() {
			Payload(func() {
				Attribute("q", Any, func() {
					Enum("val", 1)
				})
				Required("q")
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryArrayBoolDSL = func() {
	Service("ServiceQueryArrayBool", func() {
		Method("MethodQueryArrayBool", func() {
			Payload(func() {
				Attribute("q", ArrayOf(Boolean))
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryArrayBoolValidateDSL = func() {
	Service("ServiceQueryArrayBoolValidate", func() {
		Method("MethodQueryArrayBoolValidate", func() {
			Payload(func() {
				Attribute("q", ArrayOf(Boolean), func() {
					MinLength(1)
					Elem(func() {
						Enum(true)
					})
				})
				Required("q")
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryArrayIntDSL = func() {
	Service("ServiceQueryArrayInt", func() {
		Method("MethodQueryArrayInt", func() {
			Payload(func() {
				Attribute("q", ArrayOf(Int))
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryArrayIntValidateDSL = func() {
	Service("ServiceQueryArrayIntValidate", func() {
		Method("MethodQueryArrayIntValidate", func() {
			Payload(func() {
				Attribute("q", ArrayOf(Int), func() {
					MinLength(1)
					Elem(func() {
						Minimum(1)
					})
				})
				Required("q")
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryArrayInt32DSL = func() {
	Service("ServiceQueryArrayInt32", func() {
		Method("MethodQueryArrayInt32", func() {
			Payload(func() {
				Attribute("q", ArrayOf(Int32))
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryArrayInt32ValidateDSL = func() {
	Service("ServiceQueryArrayInt32Validate", func() {
		Method("MethodQueryArrayInt32Validate", func() {
			Payload(func() {
				Attribute("q", ArrayOf(Int32), func() {
					MinLength(1)
					Elem(func() {
						Minimum(1)
					})
				})
				Required("q")
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryArrayInt64DSL = func() {
	Service("ServiceQueryArrayInt64", func() {
		Method("MethodQueryArrayInt64", func() {
			Payload(func() {
				Attribute("q", ArrayOf(Int64))
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryArrayInt64ValidateDSL = func() {
	Service("ServiceQueryArrayInt64Validate", func() {
		Method("MethodQueryArrayInt64Validate", func() {
			Payload(func() {
				Attribute("q", ArrayOf(Int64), func() {
					MinLength(1)
					Elem(func() {
						Minimum(1)
					})
				})
				Required("q")
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryArrayUIntDSL = func() {
	Service("ServiceQueryArrayUInt", func() {
		Method("MethodQueryArrayUInt", func() {
			Payload(func() {
				Attribute("q", ArrayOf(UInt))
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryArrayUIntValidateDSL = func() {
	Service("ServiceQueryArrayUIntValidate", func() {
		Method("MethodQueryArrayUIntValidate", func() {
			Payload(func() {
				Attribute("q", ArrayOf(UInt), func() {
					MinLength(1)
					Elem(func() {
						Minimum(1)
					})
				})
				Required("q")
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryArrayUInt32DSL = func() {
	Service("ServiceQueryArrayUInt32", func() {
		Method("MethodQueryArrayUInt32", func() {
			Payload(func() {
				Attribute("q", ArrayOf(UInt32))
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryArrayUInt32ValidateDSL = func() {
	Service("ServiceQueryArrayUInt32Validate", func() {
		Method("MethodQueryArrayUInt32Validate", func() {
			Payload(func() {
				Attribute("q", ArrayOf(UInt32), func() {
					MinLength(1)
					Elem(func() {
						Minimum(1)
					})
				})
				Required("q")
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryArrayUInt64DSL = func() {
	Service("ServiceQueryArrayUInt64", func() {
		Method("MethodQueryArrayUInt64", func() {
			Payload(func() {
				Attribute("q", ArrayOf(UInt64))
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryArrayUInt64ValidateDSL = func() {
	Service("ServiceQueryArrayUInt64Validate", func() {
		Method("MethodQueryArrayUInt64Validate", func() {
			Payload(func() {
				Attribute("q", ArrayOf(UInt64), func() {
					MinLength(1)
					Elem(func() {
						Minimum(1)
					})
				})
				Required("q")
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryArrayFloat32DSL = func() {
	Service("ServiceQueryArrayFloat32", func() {
		Method("MethodQueryArrayFloat32", func() {
			Payload(func() {
				Attribute("q", ArrayOf(Float32))
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryArrayFloat32ValidateDSL = func() {
	Service("ServiceQueryArrayFloat32Validate", func() {
		Method("MethodQueryArrayFloat32Validate", func() {
			Payload(func() {
				Attribute("q", ArrayOf(Float32), func() {
					MinLength(1)
					Elem(func() {
						Minimum(1)
					})
				})
				Required("q")
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryArrayFloat64DSL = func() {
	Service("ServiceQueryArrayFloat64", func() {
		Method("MethodQueryArrayFloat64", func() {
			Payload(func() {
				Attribute("q", ArrayOf(Float64))
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryArrayFloat64ValidateDSL = func() {
	Service("ServiceQueryArrayFloat64Validate", func() {
		Method("MethodQueryArrayFloat64Validate", func() {
			Payload(func() {
				Attribute("q", ArrayOf(Float64), func() {
					MinLength(1)
					Elem(func() {
						Minimum(1)
					})
				})
				Required("q")
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryArrayStringDSL = func() {
	Service("ServiceQueryArrayString", func() {
		Method("MethodQueryArrayString", func() {
			Payload(func() {
				Attribute("q", ArrayOf(String))
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryArrayStringValidateDSL = func() {
	Service("ServiceQueryArrayStringValidate", func() {
		Method("MethodQueryArrayStringValidate", func() {
			Payload(func() {
				Attribute("q", ArrayOf(String), func() {
					MinLength(1)
					Elem(func() {
						Enum("val")
					})
				})
				Required("q")
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryArrayBytesDSL = func() {
	Service("ServiceQueryArrayBytes", func() {
		Method("MethodQueryArrayBytes", func() {
			Payload(func() {
				Attribute("q", ArrayOf(Bytes))
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryArrayBytesValidateDSL = func() {
	Service("ServiceQueryArrayBytesValidate", func() {
		Method("MethodQueryArrayBytesValidate", func() {
			Payload(func() {
				Attribute("q", ArrayOf(Bytes), func() {
					MinLength(1)
					Elem(func() {
						MinLength(2)
					})
				})
				Required("q")
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryArrayAnyDSL = func() {
	Service("ServiceQueryArrayAny", func() {
		Method("MethodQueryArrayAny", func() {
			Payload(func() {
				Attribute("q", ArrayOf(Any))
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryArrayAnyValidateDSL = func() {
	Service("ServiceQueryArrayAnyValidate", func() {
		Method("MethodQueryArrayAnyValidate", func() {
			Payload(func() {
				Attribute("q", ArrayOf(Any), func() {
					MinLength(1)
					Elem(func() {
						Enum("val", 1)
					})
				})
				Required("q")
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryArrayAliasDSL = func() {
	var Alias = Type("Alias", String)
	Service("ServiceQueryArrayAlias", func() {
		Method("MethodQueryArrayAlias", func() {
			Payload(func() {
				Attribute("q", ArrayOf(Alias))
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryMapStringStringDSL = func() {
	Service("ServiceQueryMapStringString", func() {
		Method("MethodQueryMapStringString", func() {
			Payload(func() {
				Attribute("q", MapOf(String, String))
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryMapStringStringValidateDSL = func() {
	Service("ServiceQueryMapStringStringValidate", func() {
		Method("MethodQueryMapStringStringValidate", func() {
			Payload(func() {
				Attribute("q", MapOf(String, String), func() {
					MinLength(1)
					Key(func() {
						Enum("key")
					})
					Elem(func() {
						Enum("val")
					})
				})
				Required("q")
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryMapStringBoolDSL = func() {
	Service("ServiceQueryMapStringBool", func() {
		Method("MethodQueryMapStringBool", func() {
			Payload(func() {
				Attribute("q", MapOf(String, Boolean))
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryMapStringBoolValidateDSL = func() {
	Service("ServiceQueryMapStringBoolValidate", func() {
		Method("MethodQueryMapStringBoolValidate", func() {
			Payload(func() {
				Attribute("q", MapOf(String, Boolean), func() {
					MinLength(1)
					Key(func() {
						Enum("key")
					})
					Elem(func() {
						Enum(true)
					})
				})
				Required("q")
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryMapBoolStringDSL = func() {
	Service("ServiceQueryMapBoolString", func() {
		Method("MethodQueryMapBoolString", func() {
			Payload(func() {
				Attribute("q", MapOf(Boolean, String))
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}

var PayloadQueryMapBoolStringValidateDSL = func() {
	Service("ServiceQueryMapBoolStringValidate", func() {
		Method("MethodQueryMapBoolStringValidate", func() {
			Payload(func() {
				Attribute("q", MapOf(Boolean, String), func() {
					MinLength(1)
					Key(func() {
						Enum(true)
					})
					Elem(func() {
						Enum("val")
					})
				})
				Required("q")
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}


