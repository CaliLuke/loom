package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

var MultipleAPIKeySecurityDSL = func() {
	var APIKeyAuth = APIKeySecurity("api_key")
	var TenantKeyAuth = APIKeySecurity("tenant")

	Service("MultipleAPIKeySecurity", func() {
		Security(APIKeyAuth, TenantKeyAuth)
		Method("A", func() {
			Payload(func() {
				APIKey("api_key", "api_key", String)
				APIKey("tenant", "tenant_id", String)
				Required("api_key", "tenant_id")
			})

			HTTP(func() {
				POST("/")
				Header("api_key:X-API-Key")
				Header("tenant_id:X-Tenant")
			})
		})
	})
}
