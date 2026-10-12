package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// BasicCLIDefaultsDSL adds plain, inherited and independently defaulted Basic
// components to the constrained named-credential fixture.
func BasicCLIDefaultsDSL(transport string) func() {
	return func() {
		BasicCredentialConstraintsDSL(transport)()
		basic := BasicAuthSecurity("defaults")
		inherited := Type("DefaultCredential", String, func() {
			Default("inherited")
		})
		Service("defaults", func() {
			if transport == "jsonrpc" {
				JSONRPC(func() {
					POST("/defaults")
				})
			}
			for _, name := range []string{"plain", "both", "user", "pass", "required", "empty", "inherited", "override"} {
				Method(name, func() {
					Security(basic)
					Payload(func() {
						var kind expr.DataType = String
						if name == "inherited" || name == "override" {
							kind = inherited
						}
						Username("login", kind, func() {
							if name == "both" || name == "user" || name == "required" || name == "override" {
								Default("alice")
							}
							if name == "empty" {
								Default("")
							}
						})
						Password("secret", kind, func() {
							if name == "both" || name == "pass" || name == "required" {
								Default("secret")
							}
						})
						if name == "required" {
							Required("login", "secret")
						}
					})
					if transport == "jsonrpc" {
						JSONRPC(func() {
						})
					} else {
						HTTP(func() {
							POST("/defaults/" + name)
						})
					}
				})
			}
		})
	}
}
