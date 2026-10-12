package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// BasicCredentialConstraintsDSL covers constrained Basic components in shared
// request decoding, including occurrence refinements and multipart branches.
func BasicCredentialConstraintsDSL(transport string) func() {
	return func() {
		basic := BasicAuthSecurity("basic")
		credential := Type("Credential", Type("CredentialBase", String, func() {
			MinLength(2)
		}))
		API("basic", func() {
			if transport == "jsonrpc" {
				JSONRPC(func() {
				})
			}
		})
		Service("basic", func() {
			if transport == "jsonrpc" {
				JSONRPC(func() {
					POST("/")
				})
			}
			methods := []string{"user", "pass", "optional", "defaulted", "refined"}
			if transport == "http" {
				methods = append(methods, "multipart", "custom", "bound")
			}
			for _, name := range methods {
				Method(name, func() {
					Security(basic)
					basicCredentialPayload(name, credential)
					if transport == "jsonrpc" {
						JSONRPC(func() {
						})
					} else {
						HTTP(func() {
							POST("/" + name)
							if name == "bound" {
								Param("count")
							}
							if name == "multipart" || name == "custom" {
								MultipartRequest()
							}
						})
					}
				})
			}
		})
	}
}

func basicCredentialPayload(name string, credential expr.DataType) {
	Payload(func() {
		Username("login", credential, func() {
			Meta("struct:field:name", "AuthUser")
			if name == "defaulted" {
				Default("alice")
			}
			if name == "refined" {
				Enum("alice", "bob")
				MaxLength(5)
			}
		})
		Password("secret", credential, func() {
			Meta("struct:field:name", "AuthPassword")
			if name == "defaulted" {
				Default("secret")
			}
			if name == "refined" {
				Format(FormatEmail)
			}
		})
		if name == "bound" {
			Attribute("count", Int, func() {
				Minimum(1)
			})
		}
		if name == "user" || name == "bound" {
			Required("login")
		}
		if name == "pass" {
			Required("secret")
		}
		if name == "multipart" {
			Attribute("file", Bytes)
		}
		if name == "custom" {
			Attribute("files", ArrayOf(Bytes))
		}
	})
}
