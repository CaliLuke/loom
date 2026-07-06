package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

var PayloadPathCustomTextUnmarshalerDSL = func() {
	Service("ServicePathCustomTextUnmarshaler", func() {
		Method("MethodPathCustomTextUnmarshaler", func() {
			Payload(func() {
				Attribute("id", String, func() {
					Format(FormatUUID)
					Meta("struct:field:type", "uuid.UUID", "github.com/google/uuid")
				})
			})
			HTTP(func() {
				GET("/{id}")
			})
		})
	})
}

var PayloadPathCustomTextUnmarshalerValidateDSL = func() {
	Service("ServicePathCustomTextUnmarshalerValidate", func() {
		Method("MethodPathCustomTextUnmarshalerValidate", func() {
			Payload(func() {
				Attribute("id", String, func() {
					Format(FormatUUID)
					MinLength(1)
					Meta("struct:field:type", "uuid.UUID", "github.com/google/uuid")
				})
			})
			HTTP(func() {
				GET("/{id}")
			})
		})
	})
}

var PayloadQueryCustomTextUnmarshalerDSL = func() {
	Service("ServiceQueryCustomTextUnmarshaler", func() {
		Method("MethodQueryCustomTextUnmarshaler", func() {
			Payload(func() {
				Attribute("id", String, func() {
					Format(FormatUUID)
					Meta("struct:field:type", "uuid.UUID", "github.com/google/uuid")
				})
				Required("id")
			})
			HTTP(func() {
				GET("/")
				Param("id")
			})
		})
	})
}

var PayloadQueryCustomTextUnmarshalerOptionalValidateDSL = func() {
	Service("ServiceQueryCustomTextUnmarshalerOptionalValidate", func() {
		Method("MethodQueryCustomTextUnmarshalerOptionalValidate", func() {
			Payload(func() {
				Attribute("id", String, func() {
					Format(FormatUUID)
					MinLength(1)
					Meta("struct:field:type", "uuid.UUID", "github.com/google/uuid")
				})
			})
			HTTP(func() {
				GET("/")
				Param("id")
			})
		})
	})
}

var PayloadQueryCustomTextUnmarshalerOptionalDSL = func() {
	Service("ServiceQueryCustomTextUnmarshalerOptional", func() {
		Method("MethodQueryCustomTextUnmarshalerOptional", func() {
			Payload(func() {
				Attribute("id", String, func() {
					Format(FormatUUID)
					Meta("struct:field:type", "uuid.UUID", "github.com/google/uuid")
				})
			})
			HTTP(func() {
				GET("/")
				Param("id")
			})
		})
	})
}

var PayloadHeaderCustomTextUnmarshalerDSL = func() {
	Service("ServiceHeaderCustomTextUnmarshaler", func() {
		Method("MethodHeaderCustomTextUnmarshaler", func() {
			Payload(func() {
				Attribute("id", String, func() {
					Format(FormatUUID)
					Meta("struct:field:type", "uuid.UUID", "github.com/google/uuid")
				})
				Required("id")
			})
			HTTP(func() {
				GET("/")
				Header("id")
			})
		})
	})
}

var PayloadHeaderCustomTextUnmarshalerOptionalValidateDSL = func() {
	Service("ServiceHeaderCustomTextUnmarshalerOptionalValidate", func() {
		Method("MethodHeaderCustomTextUnmarshalerOptionalValidate", func() {
			Payload(func() {
				Attribute("id", String, func() {
					Format(FormatUUID)
					MinLength(1)
					Meta("struct:field:type", "uuid.UUID", "github.com/google/uuid")
				})
			})
			HTTP(func() {
				GET("/")
				Header("id")
			})
		})
	})
}

var PayloadCookieCustomTextUnmarshalerDSL = func() {
	Service("ServiceCookieCustomTextUnmarshaler", func() {
		Method("MethodCookieCustomTextUnmarshaler", func() {
			Payload(func() {
				Attribute("id", String, func() {
					Format(FormatUUID)
					Meta("struct:field:type", "uuid.UUID", "github.com/google/uuid")
				})
				Required("id")
			})
			HTTP(func() {
				GET("/")
				Cookie("id")
			})
		})
	})
}

var PayloadCookieCustomTextUnmarshalerDefaultDSL = func() {
	Service("ServiceCookieCustomTextUnmarshalerDefault", func() {
		Method("MethodCookieCustomTextUnmarshalerDefault", func() {
			Payload(func() {
				Attribute("id", String, func() {
					Default("00000000-0000-0000-0000-000000000000")
					Format(FormatUUID)
					Meta("struct:field:type", "uuid.UUID", "github.com/google/uuid")
				})
			})
			HTTP(func() {
				GET("/")
				Cookie("id")
			})
		})
	})
}
