package testdata

import (
	. "github.com/CaliLuke/loom/v3/dsl"
)

var BasicAuth = BasicAuthSecurity("basic", func() {
	Scope("api:read", "Read-only access")
	Scope("api:write", "Read and write access")
	Scope("api:admin", "Admin access")
})

var JWTAuth = JWTSecurity("jwt", func() {
	Scope("api:read", "Read-only access")
	Scope("api:write", "Read and write access")
	Scope("api:admin", "Admin access")
})

var APIKeyAuth = APIKeySecurity("api_key", func() {
	Scope("api:read", "Read-only access")
	Scope("api:write", "Read and write access")
	Scope("api:admin", "Admin access")
})

var OAuth2AuthorizationCode = OAuth2Security("authCode", func() {
	AuthorizationCodeFlow("/authorization", "/token", "/refresh")
	Scope("api:write", "Write acess")
	Scope("api:read", "Read access")
})

var EndpointWithoutRequirementDSL = func() {
	Service("EndpointWithoutRequirement", func() {
		Method("Unsecure", func() {
			HTTP(func() {
				GET("/")
			})
		})
	})
}

var EndpointNoSecurityDSL = func() {
	Service("EndpointNoSecurity", func() {
		Security(BasicAuth)
		Method("NoSecurity", func() {
			NoSecurity()
			HTTP(func() {
				GET("/")
			})
		})
	})
}

var EndpointsWithServiceRequirementsDSL = func() {
	Service("EndpointsWithServiceRequirements", func() {
		Security(BasicAuth)
		Method("SecureWithRequirements", func() {
			Payload(func() {
				Username("user", String)
				Password("pass", String)
			})
			HTTP(func() {
				GET("/")
			})
		})
		Method("AlsoSecureWithRequirements", func() {
			Payload(func() {
				Username("user", String)
				Password("pass", String)
			})
			HTTP(func() {
				POST("/")
			})
		})
	})
}

var EndpointsWithRequirementsDSL = func() {
	Service("EndpointsWithRequirements", func() {
		Method("SecureWithRequirements", func() {
			Security(BasicAuth)
			Payload(func() {
				Username("user", String)
				Password("pass", String)
			})
			HTTP(func() {
				GET("/")
			})
		})
		Method("DoublySecureWithRequirements", func() {
			Security(BasicAuth, JWTAuth)
			Payload(func() {
				Username("user", String)
				Password("pass", String)
				Token("token", String)
			})
			HTTP(func() {
				POST("/")
			})
		})
	})
}

var EndpointWithRequiredScopesDSL = func() {
	Service("EndpointWithRequiredScopes", func() {
		Method("SecureWithRequiredScopes", func() {
			Security(JWTAuth, func() {
				Scope("api:read")
				Scope("api:write")
			})
			Payload(func() {
				Token("token", String)
			})
			HTTP(func() {
				GET("/")
			})
		})
	})
}

var EndpointWithOptionalRequiredScopesDSL = func() {
	Service("EndpointWithOptionalRequiredScopes", func() {
		Method("SecureWithOptionalRequiredScopes", func() {
			Security(BasicAuth, func() {
				Scope("api:read")
				Scope("api:write")
			})
			Payload(func() {
				Username("user", String)
				Password("pass", String)
			})
			HTTP(func() {
				GET("/")
			})
		})
	})
}

var EndpointWithAPIKeyOverrideDSL = func() {
	Service("EndpointWithAPIKeyOverride", func() {
		Security(BasicAuth)
		Method("SecureWithAPIKeyOverride", func() {
			Security(APIKeyAuth)
			Payload(func() {
				APIKey("api_key", "key", String)
			})
			HTTP(func() {
				GET("/")
			})
		})
	})
}

var EndpointWithOAuth2DSL = func() {
	Service("EndpointWithOAuth2", func() {
		Method("SecureWithOAuth2", func() {
			Security(OAuth2AuthorizationCode)
			Payload(func() {
				AccessToken("token", String)
			})
			HTTP(func() {
				GET("/")
			})
		})
	})
}

var EndpointWithBasicAuthAndSkipRequestBodyEncodeDecodeDSL = func() {
	Service("EndpointWithSkipRequestBodyEncodeDecode", func() {
		Method("EndpointWithSkipRequestBodyEncodeDecode", func() {
			Security(BasicAuth)
			Payload(func() {
				Username("user", String)
				Password("pass", String)
			})
			HTTP(func() {
				SkipRequestBodyEncodeDecode()
				GET("/")
			})
		})
	})
}

var EndpointWithBearerOrCookieSecurityDSL = func() {
	Service("EndpointWithBearerOrCookieSecurity", func() {
		Method("Secure", func() {
			Security(JWTAuth)
			Security(APIKeyAuth)
			Payload(func() {
				Token("auth", String)
				APIKey("api_key", "browser_session", String)
				Attribute("message", String)
			})
		})
	})
}

var EndpointWithSessionSecurityDSL = func() {
	var AppSession = SessionAuth("app_session", func() {
		BearerTransport(JWTAuth, "auth")
		CookieTransport(APIKeyAuth, "browser_session")
	})
	Service("EndpointWithSessionSecurity", func() {
		Method("Secure", func() {
			SessionSecurity(AppSession)
			Payload(func() {
				Attribute("message", String)
			})
		})
	})
}

var EndpointWithBearerOrCookieAPISecurityDSL = func() {
	API("EndpointWithBearerOrCookieAPISecurityAPI", func() {
		Security(JWTAuth)
		Security(APIKeyAuth)
	})
	Service("EndpointWithBearerOrCookieAPISecurity", func() {
		Method("Secure", func() {
			Payload(func() {
				Token("auth", String)
				APIKey("api_key", "browser_session", String)
				Attribute("message", String)
			})
		})
	})
}

var EndpointWithAPISessionSecurityDSL = func() {
	var AppSession = SessionAuth("api_app_session", func() {
		BearerTransport(JWTAuth, "auth")
		CookieTransport(APIKeyAuth, "browser_session")
	})
	API("EndpointWithAPISessionSecurityAPI", func() {
		SessionSecurity(AppSession)
	})
	Service("EndpointWithAPISessionSecurity", func() {
		Method("Secure", func() {
			Payload(func() {
				Attribute("message", String)
			})
		})
	})
}

var EndpointWithServiceSessionSecurityNoSecurityDSL = func() {
	var AppSession = SessionAuth("service_app_session", func() {
		BearerTransport(JWTAuth, "auth")
		CookieTransport(APIKeyAuth, "browser_session")
	})
	Service("EndpointWithServiceSessionSecurityNoSecurity", func() {
		SessionSecurity(AppSession)
		Method("Secure", func() {
			NoSecurity()
			Payload(func() {
				Attribute("message", String)
			})
		})
	})
}

var SingleServiceDSL = func() {
	Service("SingleService", func() {
		Method("Method", func() {
			Security(APIKeyAuth)
			Payload(func() {
				APIKey("api_key", "key", String)
			})
			HTTP(func() {
				GET("/")
			})
		})
	})
}

var MultipleServicesDSL = func() {
	Service("ServiceWithAPIKeyAuth", func() {
		Method("Method", func() {
			Security(APIKeyAuth)
			Payload(func() {
				APIKey("api_key", "key", String)
			})
			HTTP(func() {
				GET("/")
			})
		})
	})
	Service("ServiceWithJWTAndAPIKey", func() {
		Security(APIKeyAuth, JWTAuth)
		Method("Method", func() {
			Payload(func() {
				APIKey("api_key", "key", String)
				Token("token", String)
			})
			HTTP(func() {
				GET("/")
			})
		})
	})
	Service("ServiceWithNoSecurity", func() {
		Method("Method", func() {
			Payload(func() {
				Attribute("a", String)
			})
			HTTP(func() {
				GET("/{a}")
			})
		})
	})
}
