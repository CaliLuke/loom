package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// BasicCLISourcesDSL isolates direct credential inputs that also feed authored
// wire mappings, alongside the established Basic representation controls.
func BasicCLISourcesDSL(transport, mapping string) func() {
	return func() {
		basic := BasicAuthSecurity("basic")
		credential := Type("Credential", String, func() {
			MinLength(2)
		})
		API("sources", func() {
			if transport == "jsonrpc" {
				JSONRPC(func() {
				})
			}
		})
		Service("sources", func() {
			if transport == "jsonrpc" {
				JSONRPC(func() {
					POST("/")
				})
			}
			Method("call", func() {
				if mapping != "ordinary-repeated" {
					Security(basic)
				}
				Payload(func() {
					username, password := Username, Password
					if mapping == "ordinary-repeated" {
						username, password = Attribute, Attribute
					}
					username("login", credential, func() {
						if mapping != "scalar-body" {
							Meta("struct:field:name", "AuthUser")
						}
						if mapping == "defaulted" {
							Default("alice")
						}
					})
					password("passcode", credential, func() {
						Meta("struct:field:name", "AuthPassword")
						if mapping == "defaulted" {
							Default("secret")
						}
					})
					if mapping != "optional" && mapping != "defaulted" {
						Required("login", "passcode")
					}
				})
				wire := func() {
					switch mapping {
					case "query-header", "optional", "defaulted":
						Param("login:signin")
						Header("passcode:X-Proof")
					case "path-cookie":
						Param("login")
						Cookie("passcode:proof")
					case "repeated", "ordinary-repeated":
						Param("login:signin")
						Header("login:X-Login")
						Cookie("passcode:proof")
					case "scalar-body":
						Body("login")
					case "object-body":
						Body(func() {
							Attribute("login")
							Attribute("passcode")
						})
					}
				}
				if transport == "jsonrpc" {
					JSONRPC(wire)
				} else {
					HTTP(func() {
						path := "/"
						if mapping == "path-cookie" {
							path = "/{login}"
						}
						POST(path)
						wire()
					})
				}
			})
		})
	}
}
