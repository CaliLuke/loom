package testdata

import . "github.com/CaliLuke/loom/dsl"

// EffectiveWireBindingsDSL exercises unambiguous request, response and security
// mappings through the shared HTTP expression owner.
func EffectiveWireBindingsDSL(transport string) func() {
	return effectiveWireBindingsDSL(transport, "authorization")
}

// ExplicitAuthorizationWireBindingsDSL uses an explicit Authorization mapping
// for a Bearer alternative beside Basic authentication.
func ExplicitAuthorizationWireBindingsDSL(transport string) func() {
	return effectiveWireBindingsDSL(transport, "Authorization")
}

func effectiveWireBindingsDSL(transport, authorization string) func() {
	return func() {
		basic := BasicAuthSecurity("basic")
		jwt := JWTSecurity("jwt")
		key := APIKeySecurity("key")
		oauth := OAuth2Security("oauth", func() {
			AuthorizationCodeFlow("https://example.com/auth", "https://example.com/token", "")
		})
		endpoint := func(path string, mapping func()) {
			if transport == "jsonrpc" {
				JSONRPC(mapping)
			} else {
				HTTP(func() {
					POST(path)
					mapping()
				})
			}
		}
		Service("binding", func() {
			if transport == "jsonrpc" {
				JSONRPC(func() {
					POST("/rpc")
				})
			}
			wireBindingOrdinaryMethods(endpoint)
			for _, name := range []string{"either", "explicit", "and-key", "or-key", "shared-frame"} {
				Method(name, func() {
					switch name {
					case "and-key":
						Security(jwt, key)
					case "or-key":
						Security(jwt)
						Security(key)
					case "shared-frame":
						Security(jwt, oauth)
					default:
						Security(basic)
						Security(jwt)
					}
					Payload(func() {
						Token("token", String, func() {
							if name == "shared-frame" {
								Meta("security:accesstoken")
							}
						})
						if name == "and-key" || name == "or-key" {
							APIKey("key", "key", String)
						} else if name != "shared-frame" {
							Username("username", String)
							Password("password", String)
						}
						if name == "and-key" {
							Required("token", "key")
						}
					})
					endpoint("/"+name, func() {
						if name == "explicit" {
							Header("token:" + authorization)
						}
						if name == "and-key" || name == "or-key" {
							Header("key:X-Api-Key")
						}
					})
				})
			}
		})
	}
}

// EffectiveWireBindingScopeDSL retains an ordinary body field whose inherited
// credential scheme is replaced by method-level security.
func EffectiveWireBindingScopeDSL(transport string) func() {
	return func() {
		parent := APIKeySecurity("parent")
		local := APIKeySecurity("local")
		Service("scope", func() {
			Security(parent)
			if transport == "jsonrpc" {
				JSONRPC(func() {
					POST("/rpc")
				})
			}
			Method("read", func() {
				Security(local)
				Payload(func() {
					APIKey("local", "active", String)
					APIKey("parent", "other", String)
					Required("active", "other")
				})
				if transport == "jsonrpc" {
					JSONRPC(func() {
					})
				} else {
					HTTP(func() {
						POST("/")
					})
				}
			})
		})
	}
}

func wireBindingOrdinaryMethods(endpoint func(string, func())) {
	Method("response", func() {
		Result(func() {
			Attribute("a", String)
			Attribute("b", String)
			Required("a", "b")
		})
		endpoint("/response", func() {
			Response(StatusOK, func() {
				Header("a:X-A")
				Header("b:X-B")
			})
		})
	})
	for _, name := range []string{"header", "query", "cookie"} {
		Method(name, func() {
			Payload(func() {
				Attribute("a", String)
				Attribute("b", String)
				Required("a", "b")
			})
			endpoint("/"+name, func() {
				switch name {
				case "header":
					Header("a:X-A")
					Header("b:X-B")
				case "query":
					Param("a:q")
					Param("b:Q")
				case "cookie":
					Cookie("a:session")
					Cookie("b:Session")
				}
			})
		})
	}
}

// EffectiveWireBindingErrorsDSL exercises inferred problem headers beside
// explicit headers and full or empty error response bodies.
func EffectiveWireBindingErrorsDSL() {
	Service("problems", func() {
		for _, name := range []string{"full", "empty", "mapped"} {
			Method(name, func() {
				Error("oops")
				HTTP(func() {
					GET("/" + name)
					Response("oops", StatusBadRequest, func() {
						switch name {
						case "empty":
							Body(Empty)
						case "mapped":
							Body(Empty)
							Header("title:X-Title")
						}
					})
				})
			})
		}
	})
}
