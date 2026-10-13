package testdata

import . "github.com/CaliLuke/loom/dsl"

// JSONRPCWebSocketCredentialRolesDSL exercises role and type survival when
// the transport selects a separately authored streaming request.
func JSONRPCWebSocketCredentialRolesDSL(mode string) func() {
	return func() {
		jwt := JWTSecurity("jwt")
		key := APIKeySecurity("key")
		credential := Type("Credential", String)
		payload := Type("Request", func() {
			Token("token", credential, func() {
				if mode == "shared-lost" {
					Meta("security:apikey:key")
				}
			})
			if mode == "shared-lost" {
				return
			}
			APIKey("key", "key", String)
		})
		stream := Type("Message", func() {
			switch mode {
			case "renamed":
				Token("renamed", String)
			case "wrong-type":
				Token("token", Int)
			case "nullable":
				Token("token", String, func() {
					Nullable()
				})
			case "custom":
				Token("token", String, func() {
					Meta("struct:field:type", "string")
				})
			default:
				Token("token", credential)
			}
			if mode != "partial-and" && mode != "partial-or" {
				APIKey("key", "key", String)
			}
		})
		API("roles", func() {
			JSONRPC(func() {})
		})
		Service("secured", func() {
			JSONRPC(func() {
				GET("/rpc")
			})
			Method("watch", func() {
				if mode == "partial-or" {
					Security(jwt)
					Security(key)
				} else {
					Security(jwt, key)
				}
				Payload(payload)
				StreamingPayload(stream)
				StreamingResult(String)
				JSONRPC(func() {
					Body(func() {
						Attribute("token")
						if mode != "shared-lost" {
							Attribute("key")
						}
					})
				})
			})
		})
	}
}
