package testdata

import . "goa.design/goa/v3/dsl"

var BasicAuth = BasicAuthSecurity("basic")

var JWTAuth = JWTSecurity("jwt", func() {
	Scope("api:read", "Read-only access")
	Scope("api:write", "Read and write access")
	Scope("api:admin", "Admin access")
})

var APIKeyAuth = APIKeySecurity("api_key")

var OAuth2 = OAuth2Security("authCode", func() {
	AuthorizationCodeFlow("http://^authorization", "^example:/token<>", "http://refresh^") // invalid URLs
	Scope("api:write", "Write acess")
	Scope("api:read", "Read access")
})

var ValidSecuritySchemesExtendDSL = func() {
	var CommonAttr = Type("Common", func() {
		Attribute("version", String)
	})
	var SecurityAttr = Type("Security", func() {
		Username("user", String)
		Password("pass", String)
	})
	Service("ValidSecuritySchemesExtendService", func() {
		Method("SecureMethod", func() {
			Security(BasicAuth)
			Payload(func() {
				Extend(CommonAttr)
				Extend(SecurityAttr)
			})
		})
	})
}

var InvalidSecuritySchemesDSL = func() {
	Service("InvalidSecuritySchemesService", func() {
		Security(OAuth2, APIKeyAuth, func() {
			Scope("not:found") // invalid security scope
		})
		Method("SecureMethod", func() {
			Security(BasicAuth, JWTAuth, func() {
				Scope("not:found") // invalid security scope
			})
			Payload(func() {
				Attribute("a", String)
				// invalid: missing security attribute definitions
			})
		})
		Method("InheritedSecureMethod", func() {
			Payload(func() {
				Attribute("b", String)
				// invalid: missing security attribute definitions
			})
		})
	})
	Service("AnotherInvalidSecuritySchemesService", func() {
		Method("Method", func() {
			Payload(func() {
				Username("user", String)
				Password("pass", String)
				APIKey("key_key", "key", String)
				Token("token", String)
				AccessToken("access_token", String)
			})
			// invalid: missing security scheme
		})
	})
}

var UnionBranchSecurityAttributeDSL = func() {
	TokenPayload := Type("UnionBranchSecurityTokenPayload", func() {
		Token("token", String)
		Attribute("message", String)
	})
	JSONPayload := Type("UnionBranchSecurityJSONPayload", func() {
		Attribute("message", String)
	})
	Service("UnionBranchSecurityService", func() {
		Method("SecureMethod", func() {
			Security(JWTAuth)
			Payload(OneOf(TokenPayload, JSONPayload))
		})
	})
}

var ConstructorUnionResultViewDSL = func() {
	TinyA := ResultType("application/vnd.a", func() {
		TypeName("TinyA")
		Attributes(func() {
			Attribute("a", String)
		})
		View("default", func() {
			Attribute("a")
		})
		View("tiny", func() {
			Attribute("a")
		})
	})
	TinyB := ResultType("application/vnd.b", func() {
		TypeName("TinyB")
		Attributes(func() {
			Attribute("b", String)
		})
		View("default", func() {
			Attribute("b")
		})
	})
	Service("ConstructorUnionResultViewService", func() {
		Method("Show", func() {
			Result(OneOf(TinyA, TinyB), func() {
				View("tiny")
			})
		})
	})
}

var ValidSessionSecurityDSL = func() {
	var AppSession = SessionAuth("app_session", func() {
		Description("Application session")
		BearerTransport(JWTAuth, "auth")
		CookieTransport(APIKeyAuth, "browser_session")
	})
	Service("ValidSessionSecurityService", func() {
		Method("SecureMethod", func() {
			SessionSecurity(AppSession)
			Payload(func() {
				Token("token", String)
				APIKey("api_key", "browser_session", String)
			})
		})
	})
}

var InvalidSessionSecurityTransportDSL = func() {
	var InvalidSession = SessionAuth("invalid_session", func() {
		CookieTransport(JWTAuth, "browser_session")
	})
	Service("InvalidSessionSecurityTransportService", func() {
		Method("SecureMethod", func() {
			SessionSecurity(InvalidSession)
			Payload(func() {
				Token("token", String)
			})
		})
	})
}

var InvalidSessionSecurityDuplicateTransportDSL = func() {
	var OAuth2JWT = OAuth2Security("oauth2_jwt", func() {
		AuthorizationCodeFlow("https://example.com/auth", "https://example.com/token", "https://example.com/refresh")
	})
	var InvalidSession = SessionAuth("duplicate_transport_session", func() {
		BearerTransport(JWTAuth, "auth")
		BearerTransport(OAuth2JWT, "access_token")
	})
	Service("InvalidSessionSecurityDuplicateTransportService", func() {
		Method("SecureMethod", func() {
			SessionSecurity(InvalidSession)
			Payload(func() {
				Token("token", String)
				AccessToken("oauth_token", String)
			})
		})
	})
}

var ValidSessionSecurityAutoPayloadDSL = func() {
	var AppSession = SessionAuth("auto_payload_session", func() {
		BearerTransport(JWTAuth, "auth")
		CookieTransport(APIKeyAuth, "browser_session")
	})
	Service("ValidSessionSecurityAutoPayloadService", func() {
		Method("SecureMethod", func() {
			SessionSecurity(AppSession)
			Payload(func() {
				Attribute("message", String)
			})
		})
	})
}

var ValidAPISessionSecurityDSL = func() {
	var JWT = JWTSecurity("jwt")
	var APIKey = APIKeySecurity("api_key")
	var AppSession = SessionAuth("api_session", func() {
		BearerTransport(JWT, "auth")
		CookieTransport(APIKey, "browser_session")
	})

	API("ValidAPISessionSecurityAPI", func() {
		SessionSecurity(AppSession)
	})

	Service("ValidAPISessionSecurityService", func() {
		Method("SecureMethod", func() {
			Payload(func() {
				Attribute("payload", String)
			})
			Result(String)
		})
	})
}

var ValidServiceSessionSecurityNoSecurityDSL = func() {
	var JWT = JWTSecurity("jwt")
	var APIKey = APIKeySecurity("api_key")
	var AppSession = SessionAuth("service_no_security_session", func() {
		BearerTransport(JWT, "auth")
		CookieTransport(APIKey, "browser_session")
	})

	Service("ValidServiceSessionSecurityNoSecurityService", func() {
		SessionSecurity(AppSession)
		Method("SecureMethod", func() {
			NoSecurity()
			Payload(func() {
				Attribute("payload", String)
			})
			Result(String)
		})
	})
}

var InvalidSessionSecurityPayloadConflictDSL = func() {
	var AppSession = SessionAuth("conflict_session", func() {
		BearerTransport(JWTAuth, "auth")
		CookieTransport(APIKeyAuth, "browser_session")
	})
	Service("InvalidSessionSecurityPayloadConflictService", func() {
		Method("SecureMethod", func() {
			SessionSecurity(AppSession)
			Payload(func() {
				Attribute("auth", Int)
			})
		})
	})
}

var ValidMethodAuthErrorResponsesDSL = func() {
	Service("ValidMethodAuthErrorResponsesService", func() {
		Method("SecureMethod", func() {
			Security(JWTAuth)
			Payload(func() {
				Token("auth", String)
			})
			HTTP(func() {
				GET("/secure")
				AuthErrorResponses()
			})
		})
	})
}

var ValidServiceAuthErrorResponsesDSL = func() {
	Service("ValidServiceAuthErrorResponsesService", func() {
		Security(JWTAuth)
		HTTP(func() {
			Path("/secure")
			AuthErrorResponses()
		})
		Method("SecureMethod", func() {
			Payload(func() {
				Token("auth", String)
			})
			HTTP(func() {
				GET("/")
			})
		})
	})
}

var ValidAPIAuthErrorResponsesDSL = func() {
	API("auth-errors", func() {
		Security(JWTAuth)
		HTTP(func() {
			Path("/api")
			AuthErrorResponses()
		})
	})
	Service("ValidAPIAuthErrorResponsesService", func() {
		Method("SecureMethod", func() {
			Payload(func() {
				Token("auth", String)
			})
			HTTP(func() {
				GET("/secure")
			})
		})
	})
}

var ValidMethodAuthErrorResponsesReuseDSL = func() {
	var Unauthorized = Type("MethodAuthErrorUnauthorized", func() {
		Attribute("reason", String)
		Required("reason")
	})
	Service("ValidMethodAuthErrorResponsesReuseService", func() {
		Method("SecureMethod", func() {
			Security(JWTAuth)
			Error("unauthorized", Unauthorized)
			Payload(func() {
				Token("auth", String)
			})
			HTTP(func() {
				GET("/secure")
				AuthErrorResponses()
			})
		})
	})
}

var InvalidAuthErrorResponsesPlacementDSL = func() {
	Service("InvalidAuthErrorResponsesPlacementService", func() {
		Method("SecureMethod", func() {
			AuthErrorResponses()
		})
	})
}

var ValidMethodAuthErrorResponsesRepeatedDSL = func() {
	Service("ValidMethodAuthErrorResponsesRepeatedService", func() {
		Method("SecureMethod", func() {
			Security(JWTAuth)
			Payload(func() {
				Token("auth", String)
			})
			HTTP(func() {
				GET("/secure")
				AuthErrorResponses()
				AuthErrorResponses()
			})
		})
	})
}

var ValidMethodAuthErrorResponsesCustomMappingDSL = func() {
	Service("ValidMethodAuthErrorResponsesCustomMappingService", func() {
		Method("SecureMethod", func() {
			Security(JWTAuth)
			Payload(func() {
				Token("auth", String)
			})
			Error("unauthorized")
			HTTP(func() {
				GET("/secure")
				Response("unauthorized", StatusPaymentRequired, func() {
					Description("Custom auth challenge")
				})
				AuthErrorResponses()
			})
		})
	})
}
