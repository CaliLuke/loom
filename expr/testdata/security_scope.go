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
						AuthErrorResponses()
					})
				case "grpc":
					GRPC(func() {
						if sessionOverride {
							Metadata(func() {
								Attribute("cookie:x-session")
							})
						}
					})
				case "jsonrpc":
					JSONRPC(func() {})
				}
			})
		})
	}
}

// SecurityScopeMatrixDSL exercises inherited, replaced, alternative, and
// explicitly disabled requirements through one transport.
func SecurityScopeMatrixDSL(transport string) func() {
	return func() {
		basic := BasicAuthSecurity("basic")
		jwt := JWTSecurity("jwt")
		key := APIKeySecurity("key")
		API("scope", func() {
			Security(basic)
			if transport == "jsonrpc" {
				JSONRPC(func() {})
			}
		})
		Service("api_scope", func() {
			matrixTransportService(transport, "api_scope")
			Method("read", func() {
				matrixMethod(transport, "read", nil, "basic")
			})
		})
		Service("scope", func() {
			Security(jwt)
			matrixTransportService(transport, "scope")
			Method("inherited", func() {
				matrixMethod(transport, "inherited", nil, "jwt")
			})
			Method("override", func() {
				matrixMethod(transport, "override", func() {
					Security(key)
				}, "key")
			})
			Method("alternatives", func() {
				matrixMethod(transport, "alternatives", func() {
					Security(basic)
					Security(jwt)
				}, "basic", "jwt")
			})
			Method("open", func() {
				matrixMethod(transport, "open", func() {
					NoSecurity()
				})
			})
		})
	}
}

func matrixTransportService(transport, path string) {
	if transport == "jsonrpc" {
		JSONRPC(func() {
			POST("/" + path)
		})
	}
}

func matrixMethod(transport, path string, security func(), credentials ...string) {
	if security != nil {
		security()
	}
	Payload(func() {
		Field(1, "message", String)
		for _, credential := range credentials {
			switch credential {
			case "basic":
				Username("user", String)
				Password("password", String)
			case "jwt":
				Token("token", String)
			case "key":
				APIKey("key", "key", String)
			}
		}
		if transport == "jsonrpc" {
			ID("id", String)
		}
	})
	switch transport {
	case "http":
		HTTP(func() {
			GET("/" + path)
			if len(credentials) > 0 {
				AuthErrorResponses()
			}
		})
	case "grpc":
		GRPC(func() {})
	case "jsonrpc":
		JSONRPC(func() {})
	}
}

// SecurityPerSchemeDSL checks separate requirements on schemes with overlapping
// scope inventories and separate alternatives on one physical credential.
func SecurityPerSchemeDSL(transport string) func() {
	return func() {
		left := OAuth2Security("left", func() {
			ClientCredentialsFlow("https://example.com/token", "")
			Scope("read")
			Scope("write")
		})
		right := OAuth2Security("right", func() {
			ClientCredentialsFlow("https://example.com/token", "")
			Scope("read")
			Scope("write")
		})
		API("per_scheme", func() {
			if transport == "jsonrpc" {
				JSONRPC(func() {})
			}
		})
		Service("per_scheme", func() {
			matrixTransportService(transport, "per_scheme")
			Method("read", func() {
				Security(left, right, func() {
					ScopeFor(left, "read")
					ScopeFor(right, "write")
				})
				Security(right, func() {
					ScopeFor(right, "read")
				})
				Security()
				Payload(func() {
					AccessToken("token")
					if transport == "jsonrpc" {
						ID("id", String)
					}
				})
				switch transport {
				case "http":
					HTTP(func() {
						GET("/per_scheme")
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
