package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadJWTAuthorizationCustomHeaderDSL = func() {
	var JWT = JWTSecurity("jwt", func() {
		Scope("api:read")
	})
	Service("ServiceHeaderPrimitiveStringDefault", func() {
		Method("MethodHeaderPrimitiveStringDefault", func() {
			Security(JWT)
			Payload(func() {
				Token("token", String)
				Required("token")
			})
			HTTP(func() {
				GET("")
				Header("token:X-Auth")
			})
		})
	})
}


