package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
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
	oauth, second := securityFlowScopeSchemes()
	Service("scopes", func() {
		securityFlowScopeMethods(oauth, second, key)
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

func securityFlowScopeSchemes() (*expr.SchemeExpr, *expr.SchemeExpr) {
	oauth := OAuth2Security("oauth", func() {
		AuthorizationCodeFlow("/authorize", "/token", "/refresh", func() {
			Scope("oauth", "Superseded description")
			Scope("oauth", "User consent")
			Scope("shared", "User grant")
		})
		ClientCredentialsFlow("https://example.com/token", "", func() {
			Scope("machine", "Machine grant")
			Scope("shared", "Machine permission")
		})
		ImplicitFlow("https://example.com/authorize", "", func() {})
	})
	second := OAuth2Security("second", func() {
		Scope("oauth", "Superseded default")
		ClientCredentialsFlow("https://example.com/token", "")
		ImplicitFlow("https://example.com/authorize", "", func() {})
		Scope("oauth")
		Scope("shared")
	})
	return oauth, second
}

func securityFlowScopeMethods(oauth, second, key *expr.SchemeExpr) {
	Method("flow", func() {
		Security(oauth, func() {
			ScopeFor(oauth, "machine")
		})
		Payload(func() {
			AccessToken("token")
			Required("token")
		})
		HTTP(func() {
			GET("/flow")
		})
	})
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
}
