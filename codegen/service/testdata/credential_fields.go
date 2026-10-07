package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// CredentialFieldNamesDSL exercises renamed and ordinary security fields with
// required and optional presence across all authentication schemes.
var CredentialFieldNamesDSL = func() {
	credentialFieldsDSL(false, false)
}

// NamedCredentialFieldsDSL retains named string credentials at authentication
// and HTTP boundaries, with ordinary string credentials as a control.
var NamedCredentialFieldsDSL = func() {
	credentialFieldsDSL(true, false)
}

// NamedGRPCCredentialFieldsDSL exercises named strings in gRPC credentials.
var NamedGRPCCredentialFieldsDSL = func() {
	credentialFieldsDSL(true, true)
}

func credentialFieldsDSL(named, rpc bool) {
	var credential = Type("Credential", String)
	basic := BasicAuthSecurity("credential_basic")
	key := APIKeySecurity("key")
	jwt := JWTSecurity("credential_jwt", func() { Scope("read") })
	oauth := OAuth2Security("oauth", func() {
		Scope("read")
		AuthorizationCodeFlow("https://example.com/auth", "https://example.com/token", "")
	})
	Service("credentials", func() {
		// Exercise all callback boundaries in one request; OAuth outside the
		// Authorization header is supported by HTTP but not OpenAPI.
		Meta("openapi:generate", "false")
		for _, name := range []string{"required", "optional", "ordinary"} {
			Method(name, func() {
				Security(basic, key, jwt, oauth, func() { Scope("read") })
				Payload(func() {
					var fieldType expr.DataType = String
					if named && name != "ordinary" {
						fieldType = credential
					}
					rename := func(field string) func() {
						return func() {
							if name != "ordinary" {
								Meta("struct:field:name", field)
							}
						}
					}
					Username("user", fieldType, rename("Login"))
					Password("pass", fieldType, rename("Secret"))
					APIKey("key", "key", fieldType, rename("AccessKey"))
					Token("token", fieldType, rename("JWTValue"))
					AccessToken("access", fieldType, rename("OAuthValue"))
					if name != "optional" {
						Required("user", "pass", "key", "token", "access")
					}
				})
				Result(String)
				if rpc {
					GRPC(func() {
						Metadata(func() {
							Attribute("user")
							Attribute("pass")
							Attribute("key")
							Attribute("token:Authorization")
							Attribute("access")
						})
					})
				}
				HTTP(func() {
					GET("/" + name)
					Header("key:X-Key")
					Header("token:X-Token")
					Header("access:X-Access")
				})
			})
		}
	})
}

// ConstrainedCredentialsDSL exercises nested string constraints, service import
// alias allocation and the standard Authorization bearer header.
var ConstrainedCredentialsDSL = func() {
	credential := Type("Credential", Type("CredentialBase", String, func() {
		Pattern("^[a-z]+$")
		MinLength(2)
	}))
	token := Type("TokenValue", String)
	basic := BasicAuthSecurity("constrained_basic")
	jwt := JWTSecurity("constrained_jwt")
	oauth := OAuth2Security("constrained_oauth", func() {
		AuthorizationCodeFlow("https://example.com/auth", "https://example.com/token", "")
	})
	Service("user", func() {
		Meta("openapi:generate", "false")
		for _, required := range []bool{true, false} {
			name := "optional"
			if required {
				name = "required"
			}
			Method(name, func() {
				Security(basic)
				Payload(func() {
					Username("user", credential)
					Password("pass", credential)
					if required {
						Required("user", "pass")
					}
				})
				HTTP(func() { GET("/" + name) })
				GRPC(func() {
					Metadata(func() {
						Attribute("user")
						Attribute("pass")
					})
				})
			})
		}
		for _, scheme := range []*expr.SchemeExpr{jwt, oauth} {
			name := "oauth"
			if scheme == jwt {
				name = "jwt"
			}
			Method(name, func() {
				Security(scheme)
				Payload(func() {
					if scheme == jwt {
						Token("token", token)
					} else {
						AccessToken("token", token)
					}
					Required("token")
				})
				HTTP(func() { GET("/" + name) })
				GRPC(func() {})
			})
		}
	})
}
