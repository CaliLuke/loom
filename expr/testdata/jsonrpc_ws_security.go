package testdata

import . "github.com/CaliLuke/loom/dsl"

// JSONRPCWebSocketSecurityDSL exercises effective credentials and transport
// locations independently of scope and streaming shape. It is a test fixture.
func JSONRPCWebSocketSecurityDSL(kind, shape, location, scope, transport string) func() {
	return func() {
		declare := jsonrpcWebSocketSecurity(kind)
		API("wssecurity", func() {
			if scope == "api" {
				declare()
			}
			JSONRPC(func() {})
		})
		Service("secured", func() {
			if scope != "api" && scope != "method" {
				declare()
			}
			if transport != "http-ws" {
				JSONRPC(func() {
					if transport == "post" || transport == "sse" {
						POST("/rpc")
					} else {
						GET("/rpc")
					}
				})
			}
			Method("watch", func() {
				if scope == "method" {
					declare()
				}
				if scope == "none" {
					NoSecurity()
				}
				if scope == "anonymous" {
					Security()
				}
				payload := jsonrpcWebSocketCredentials(kind, transport)
				Payload(payload)
				if shape == "bidi-retagged" {
					StreamingPayload(func() {
						Attribute("token", String)
					})
				}
				if shape == "bidi-same" {
					StreamingPayload(payload)
				}
				if shape == "bidi" || shape == "client" {
					StreamingPayload(func() {
						Attribute("value", String)
					})
				}
				if transport == "post" || shape == "client" {
					Result(String)
				} else {
					StreamingResult(String)
				}
				mapping := jsonrpcWebSocketSecurityMapping(location, transport, payload)
				if transport == "http-ws" {
					HTTP(func() {
						GET("/ws")
						mapping()
					})
				} else {
					JSONRPC(mapping)
				}
			})
		})
	}
}

func jsonrpcWebSocketSecurity(kind string) func() {
	jwt := JWTSecurity("jwt")
	key := APIKeySecurity("key")
	basic := BasicAuthSecurity("basic")
	oauth := OAuth2Security("oauth", func() {
		AuthorizationCodeFlow("https://example.com/auth", "https://example.com/token", "")
	})
	session := SessionAuth("session", func() {
		BearerTransport(jwt, "token")
		CookieTransport(key, "cookie", func() {
			CookieName("session")
		})
	})
	cookie := SessionAuth("cookie_session", func() {
		CookieTransport(key, "", func() {
			CookieName("session")
		})
	})
	return func() {
		switch kind {
		case "jwt":
			Security(jwt)
		case "optional":
			Security()
			Security(jwt)
		case "key":
			Security(key)
		case "oauth":
			Security(oauth)
		case "basic":
			Security(basic)
		case "session":
			SessionSecurity(session)
		case "cookie":
			SessionSecurity(cookie)
		case "alternative":
			Security(basic)
			Security(jwt)
		case "anonymous":
			Security()
		}
	}
}

func jsonrpcWebSocketCredentials(kind, transport string) func() {
	return func() {
		if transport != "http-ws" {
			ID("id", String)
		}
		switch kind {
		case "jwt", "alternative", "optional":
			Token("token", String)
		case "key":
			APIKey("key", "token", String)
		case "oauth":
			AccessToken("token", String)
		}
		if kind == "basic" || kind == "alternative" {
			Username("user", String)
			Password("pass", String)
		}
	}
}

func jsonrpcWebSocketSecurityMapping(location, transport string, payload func()) func() {
	return func() {
		switch location {
		case "header":
			Header("token:Authorization")
		case "cookie":
			Cookie("token:credential")
		case "query":
			Param("token")
		case "body":
			Body(payload)
		}
		if transport == "sse" {
			ServerSentEvents()
		}
	}
}
