package testdata

import . "github.com/CaliLuke/loom/dsl"

// CredentialFieldNamesDSL exercises renamed and ordinary security fields with
// required and optional presence across all authentication schemes.
var CredentialFieldNamesDSL = func() {
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
					rename := func(field string) func() {
						return func() {
							if name != "ordinary" {
								Meta("struct:field:name", field)
							}
						}
					}
					Username("user", String, rename("Login"))
					Password("pass", String, rename("Secret"))
					APIKey("key", "key", String, rename("AccessKey"))
					Token("token", String, rename("JWTValue"))
					AccessToken("access", String, rename("OAuthValue"))
					if name != "optional" {
						Required("user", "pass", "key", "token", "access")
					}
				})
				Result(String)
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
