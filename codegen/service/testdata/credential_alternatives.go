package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// CredentialAlternativesDSL exercises credential extraction without weakening
// payload-wide requiredness or malformed bearer framing.
func CredentialAlternativesDSL(transport string) func() {
	return func() {
		basic := BasicAuthSecurity("basic")
		jwt := JWTSecurity("jwt")
		key := APIKeySecurity("key")
		credential := Type("Credential", String)
		API("alternatives", func() {
			if transport == "jsonrpc" {
				JSONRPC(func() {})
			}
		})
		Service("alternatives", func() {
			if transport == "jsonrpc" {
				JSONRPC(func() {
					POST("/")
				})
			}
			for _, name := range []string{"either", "reverse", "key", "and", "single", "required", "required_token", "anonymous"} {
				Method(name, func() {
					credentialAlternativeSecurity(name, basic, jwt, key)
					credentialAlternativePayload(name, credential)
					Result(String)
					credentialAlternativeTransport(transport, name)
				})
			}
		})
	}
}

func credentialAlternativeSecurity(name string, basic, jwt, key *expr.SchemeExpr) {
	switch name {
	case "either", "required":
		Security(basic)
		Security(jwt)
	case "reverse":
		Security(jwt)
		Security(basic)
	case "key", "required_token":
		Security(jwt)
		Security(key)
	case "and":
		Security(jwt, key)
	case "single":
		Security(jwt)
	case "anonymous":
		Security(jwt)
		Security()
	}
}

func credentialAlternativePayload(name string, credential expr.DataType) {
	Payload(func() {
		Token("token", credential, func() {
			Pattern("^[a-z]+$")
		})
		if name == "either" || name == "reverse" || name == "required" {
			Username("user", credential, func() {
				MinLength(2)
			})
			Password("pass", credential, func() {
				MinLength(2)
			})
		}
		if name == "key" || name == "and" || name == "required_token" {
			APIKey("key", "key", credential)
		}
		if name == "required" {
			Required("user")
		}
		if name == "required_token" {
			Required("token")
		}
	})
}

func credentialAlternativeTransport(transport, name string) {
	switch transport {
	case "http":
		HTTP(func() {
			GET("/" + name)
			if name == "key" || name == "and" || name == "required_token" {
				Header("key:X-Key")
			}
		})
	case "jsonrpc":
		JSONRPC(func() {
			if name == "key" || name == "and" || name == "required_token" {
				Header("key:X-Key")
			}
		})
	case "grpc":
		GRPC(func() {
			Metadata(func() {
				Attribute("token:authorization")
				if name == "either" || name == "reverse" || name == "required" {
					Attribute("user")
					Attribute("pass")
				}
				if name == "key" || name == "and" || name == "required_token" {
					Attribute("key:x-key")
				}
			})
		})
	}
}
