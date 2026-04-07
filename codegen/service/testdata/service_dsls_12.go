package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

var MixedAndMultipleAPIKeySecurityDSL = func() {
	var Signature = JWTSecurity("jwt", func() {
		Scope("api:read", "Read access to API")
	})
	var APIKeyAuth = APIKeySecurity("api_key")
	var TenantKeyAuth = APIKeySecurity("tenant")

	Service("MixedAndMultipleAPIKeySecurity", func() {
		Security(Signature, func() {
			Scope("api:read")
		})
		Security(APIKeyAuth, TenantKeyAuth)

		Method("A", func() {
			Payload(func() {
				Token("jwt", String)
				APIKey("api_key", "api_key", String)
				APIKey("tenant", "tenant_id", String)
			})

			HTTP(func() {
				POST("/")
				Header("api_key:X-API-Key")
				Header("tenant_id:X-Tenant")
				Header("jwt:Authorization")
			})
		})
	})
}

// RawObjectPayloadTypeNameCollisionDSL exercises the naming of synthetic user
// types created when wrapping raw object payloads.
//
// The service generator wraps raw object payloads in a synthetic user type
// named after the method (e.g. Foo -> FooPayload). That name must be computed
// using NameScope uniqueness rules even when a would-be suffix (FooPayload2) is
// already taken, otherwise codegen can emit invalid references.
