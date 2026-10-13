package testdata

import . "github.com/CaliLuke/loom/dsl"

// DiagnosticCredentialsDSL exercises credential identities across HTTP and JSON-RPC
// previews, including body aliases and explicit duplicate Basic roles.
func DiagnosticCredentialsDSL() {
	key := APIKeySecurity("permit")
	basic := BasicAuthSecurity("basic")
	API("diagnostics", func() {
		JSONRPC(func() {
		})
	})
	Service("web", func() {
		for _, mode := range []string{"query", "header", "body", "form"} {
			Method(mode, func() {
				if mode == "body" || mode == "form" {
					Meta("openapi:generate", "false")
				}
				Security(key)
				Payload(func() {
					APIKey("permit", "credential", String, func() {
						Meta("struct:tag:json:name", "permit")
					})
					Required("credential")
				})
				HTTP(func() {
					POST("/" + mode)
					switch mode {
					case "query":
						Param("credential:ticket")
					case "header":
						Header("credential:X-Permit")
					case "body", "form":
						Body(func() {
							Attribute("credential")
						})
					}
				})
			})
		}
		Method("basic", func() {
			Security(basic)
			Payload(func() {
				Username("login", String, func() {
					Meta("struct:tag:json:name", "signin")
				})
				Password("passcode", String, func() {
					Meta("struct:tag:json:name", "proof")
				})
				Required("login", "passcode")
			})
			HTTP(func() {
				POST("/basic")
				Body(func() {
					Attribute("login")
					Attribute("passcode")
				})
			})
		})
	})
	Service("rpc", func() {
		JSONRPC(func() {
			POST("/rpc")
		})
		Method("body", func() {
			Meta("openapi:generate", "false")
			Security(key)
			Payload(func() {
				ID("id")
				APIKey("permit", "credential", String, func() {
					Meta("struct:tag:json:name", "permit")
				})
				Required("credential")
			})
			Result(func() {
				ID("id")
			})
			JSONRPC(func() {
				Body(func() {
					Attribute("credential")
				})
			})
		})
	})
}

// DiagnosticControlsDSL keeps ordinary and explicitly unauthenticated services
// free of generated credential metadata.
func DiagnosticControlsDSL() {
	key := APIKeySecurity("unused")
	API("diagnostics", func() {
	})
	for _, name := range []string{"ordinary", "disabled"} {
		Service(name, func() {
			if name == "disabled" {
				Security(key)
			}
			Method("show", func() {
				if name == "disabled" {
					NoSecurity()
				}
				Payload(func() {
					Attribute("message", String)
				})
				HTTP(func() {
					POST("/" + name)
				})
			})
		})
	}
}

// DiagnosticBodyPoliciesDSL covers whole-body omission and physical form/JSON
// names when an authored form tag makes JSON use the Go field name.
func DiagnosticBodyPoliciesDSL() {
	key := APIKeySecurity("permit")
	API("diagnostics", func() {
	})
	for _, name := range []string{"scalar", "custom"} {
		Service(name, func() {
			Security(key)
			Method("show", func() {
				Meta("openapi:generate", "false")
				Payload(func() {
					APIKey("permit", "credential", String, func() {
						if name == "custom" {
							Meta("struct:field:name", "Permit")
							Meta("struct:tag:form", "proof")
						}
					})
					Required("credential")
				})
				HTTP(func() {
					POST("/" + name)
					if name == "scalar" {
						Body("credential")
					} else {
						Body(func() {
							Attribute("credential")
						})
					}
				})
			})
		})
	}
}
