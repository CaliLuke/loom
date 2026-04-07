package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadFormBodyUnionDSL = func() {
	var Grant = Type("FormGrant", func() {
		OneOf("Values", func() {
			Attribute("AuthorizationCode", func() {
				Attribute("code", String)
				Required("code")
			})
			Attribute("RefreshToken", func() {
				Attribute("refresh_token", String)
				Required("refresh_token")
			})
		})
	})
	Service("ServiceFormBodyUnion", func() {
		Method("MethodFormBodyUnion", func() {
			Payload(Grant)
			HTTP(func() {
				POST("/")
				FormRequest()
			})
		})
	})
}


