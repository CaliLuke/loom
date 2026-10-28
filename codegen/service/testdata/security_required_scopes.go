package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// SecurityRequiredScopesDSL exercises compound and alternative scope ownership.
func SecurityRequiredScopesDSL() {
	basic := BasicAuthSecurity("scope_basic", func() {
		Scope("basic")
		Scope("shared")
	})
	key := APIKeySecurity("key", func() {
		Scope("key")
		Scope("shared")
	})
	unscoped := APIKeySecurity("unscoped")
	jwt := JWTSecurity("scope_jwt", func() {
		Scope("read")
		Scope("write")
		Scope("shared")
	})
	oauth := OAuth2Security("oauth", func() {
		AuthorizationCodeFlow("/authorize", "/token", "/refresh")
		Scope("oauth")
		Scope("shared")
	})
	second := OAuth2Security("second", func() {
		ClientCredentialsFlow("https://example.com/token", "")
		Scope("oauth")
		Scope("shared")
	})
	Service("scopes", func() {
		Method("isolated", func() {
			Security(oauth, second, key, func() {
				ScopeFor(oauth, "shared")
				ScopeFor(second, "oauth")
				RoleFor(key, "admin")
			})
			Payload(func() {
				AccessToken("token")
				APIKey("key", "key")
				Required("token", "key")
			})
			HTTP(func() {
				GET("/isolated")
				Header("key:X-Key")
			})
		})
		Method("compound", func() {
			Security(basic, key, jwt, oauth, func() {
				Scope("shared")
				Scope("read")
				Scope("oauth")
				Scope("key")
				Scope("basic")
			})
			Payload(func() {
				Username("user")
				Password("password")
				APIKey("key", "key")
				Token("token")
				AccessToken("access")
				Required("user", "password", "key", "token", "access")
			})
			HTTP(func() {
				GET("/compound")
				// OAuth2 at a custom header cannot be represented by OpenAPI.
				Meta("openapi:generate", "false")
				Header("key:X-Key")
				Header("token:X-JWT")
				Header("access:X-OAuth")
			})
		})
		Method("alternative", func() {
			Security(jwt, unscoped, func() {
				Scope("read")
			})
			Security(jwt, unscoped, func() {
				Scope("write")
			})
			Payload(func() {
				Token("token")
				APIKey("unscoped", "key")
				Required("token", "key")
			})
			HTTP(func() {
				GET("/alternative")
				Header("key:X-Key")
			})
		})
	})
}

// SecurityOAuthScopesDSL exercises distinct OAuth2 domains on one credential.
func SecurityOAuthScopesDSL() {
	read := OAuth2Security("read", func() {
		AuthorizationCodeFlow("https://example.com/authorize", "https://example.com/token", "https://example.com/refresh")
		Scope("read")
		Scope("shared")
	})
	write := OAuth2Security("write", func() {
		AuthorizationCodeFlow("https://example.com/authorize", "https://example.com/token", "https://example.com/refresh")
		Scope("write")
		Scope("shared")
	})
	Service("oauthscopes", func() {
		Method("show", func() {
			Security(read, write, func() {
				Scope("shared")
				Scope("write")
				Scope("read")
			})
			Payload(func() {
				AccessToken("token")
				Required("token")
			})
			HTTP(func() {
				GET("/oauth")
			})
		})
	})
}
