package testdata

import . "github.com/CaliLuke/loom/dsl"

// SecurityScopeDSL exercises ordinary/session scope replacement over one transport.
// A session override replaces inherited Basic; otherwise Basic replaces a session.
func SecurityScopeDSL(transport string, sessionOverride bool) func() {
	return func() {
		basic := BasicAuthSecurity("basic")
		jwt := JWTSecurity("jwt")
		key := APIKeySecurity("key")
		session := SessionAuth("session", func() {
			BearerTransport(jwt, "auth")
			CookieTransport(key, "cookie", func() {
				CookieName("session")
			})
		})
		API("scope", func() {
			if transport == "jsonrpc" {
				JSONRPC(func() {})
			}
		})
		Service("scope", func() {
			if sessionOverride {
				Security(basic)
			} else {
				SessionSecurity(session)
			}
			if transport == "jsonrpc" {
				JSONRPC(func() {
					POST("/rpc")
				})
			}
			Method("read", func() {
				if sessionOverride {
					SessionSecurity(session)
				} else {
					Security(basic)
				}
				Payload(func() {
					Field(1, "message", String)
					if transport == "jsonrpc" {
						ID("id", String)
					}
					if !sessionOverride {
						Username("user")
						Password("password")
					}
				})
				switch transport {
				case "http":
					HTTP(func() {
						GET("/")
					})
				case "grpc":
					GRPC(func() {})
				case "jsonrpc":
					JSONRPC(func() {})
				}
			})
		})
	}
}
