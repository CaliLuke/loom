package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestHTTPWireBindingCollision(t *testing.T) {
	cases := []struct {
		name     string
		response bool
		mapping  func()
	}{
		{name: "request header", mapping: func() {
			Header("a:X-Same")
			Header("b:X-Same")
		}},
		{name: "request header case", mapping: func() {
			Header("a:X-Same")
			Header("b:x-same")
		}},
		{name: "query", mapping: func() {
			Param("a:q")
			Param("b:q")
		}},
		{name: "request cookie", mapping: func() {
			Cookie("a:session")
			Cookie("b:session")
		}},
		{name: "response header", response: true, mapping: func() {
			Header("a:X-Same")
			Header("b:X-Same")
		}},
		{name: "response cookie existing rejection", response: true, mapping: func() {
			Cookie("a:session")
			Cookie("b:session")
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, wireBindingDSL(tc.response, tc.mapping))
			if tc.name == "response cookie existing rejection" {
				require.ErrorContains(t, err, "duplicate cookie")
			} else {
				require.ErrorContains(t, err, "conflicting bindings")
				require.ErrorContains(t, err, `attributes "a"`)
				require.ErrorContains(t, err, `attributes "b"`)
			}
		})
	}
}

func TestHTTPWireBindingDistinct(t *testing.T) {
	cases := []struct {
		name    string
		mapping func()
	}{
		{name: "header distinct", mapping: func() {
			Header("a:X-A")
			Header("b:X-B")
		}},
		{name: "query case sensitive", mapping: func() {
			Param("a:q")
			Param("b:Q")
		}},
		{name: "cookie case sensitive", mapping: func() {
			Cookie("a:session")
			Cookie("b:Session")
		}},
		{name: "different locations", mapping: func() {
			Header("a:same")
			Param("b:same")
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expr.RunDSL(t, wireBindingDSL(false, tc.mapping))
		})
	}
}

func wireBindingDSL(response bool, mapping func()) func() {
	return func() {
		shape := Type("Pair", func() {
			Attribute("a", String)
			Attribute("b", String)
		})
		Service("pairs", func() {
			Method("send", func() {
				Payload(shape)
				Result(shape)
				HTTP(func() {
					POST("/pairs")
					if response {
						Response(StatusOK, mapping)
					} else {
						mapping()
						Response(StatusOK)
					}
				})
			})
		})
	}
}

func TestHTTPWireBindingImplicitCredentials(t *testing.T) {
	t.Run("AND credentials conflict", func(t *testing.T) {
		requireWireBindingCollision(t, func() {
			jwt := JWTSecurity("jwt")
			key := APIKeySecurity("key")
			Service("credentials", func() {
				Method("both", func() {
					Security(jwt, key)
					Payload(func() {
						Token("token", String)
						APIKey("key", "key", String)
						Required("token", "key")
					})
					HTTP(func() {
						POST("/both")
					})
				})
			})
		})
	})
	t.Run("Basic or Bearer remains supported", func(t *testing.T) {
		expr.RunDSL(t, func() {
			basic := BasicAuthSecurity("basic")
			jwt := JWTSecurity("jwt")
			Service("credentials", func() {
				Method("either", func() {
					Security(basic)
					Security(jwt)
					Payload(func() {
						Username("username", String)
						Password("password", String)
						Token("token", String)
					})
					HTTP(func() {
						POST("/either")
					})
				})
			})
		})
	})
}

func TestHTTPWireBindingInherited(t *testing.T) {
	requireWireBindingCollision(t, func() {
		Service("inherited", func() {
			HTTP(func() {
				Header("a:X-Shared")
			})
			Method("send", func() {
				Payload(func() {
					Attribute("a", String)
					Attribute("b", String)
				})
				HTTP(func() {
					POST("/send")
					Header("b:x-shared")
				})
			})
		})
	})
}

func TestHTTPWireBindingSession(t *testing.T) {
	for _, owned := range []bool{false, true} {
		name := "transport owned"
		field := ""
		if owned {
			name, field = "payload owned", "credential"
		}
		t.Run(name, func(t *testing.T) {
			requireWireBindingCollision(t, func() {
				key := APIKeySecurity("session_key")
				session := SessionAuth("session", func() {
					CookieTransport(key, field, func() {
						CookieName("session")
					})
				})
				Service("sessions", func() {
					SessionSecurity(session)
					Method("read", func() {
						Payload(func() {
							Attribute("other", String)
						})
						HTTP(func() {
							GET("/")
							Cookie("other:session")
						})
					})
				})
			})
		})
	}
}

func TestHTTPWireBindingError(t *testing.T) {
	for _, implicit := range []bool{false, true} {
		name := "explicit"
		if implicit {
			name = "implicit error header"
		}
		t.Run(name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, func() {
				Service("errors", func() {
					Method("read", func() {
						if implicit {
							Error("oops")
						} else {
							Error("oops", func() {
								Attribute("a", String)
								Attribute("b", String)
							})
						}
						HTTP(func() {
							GET("/")
							Response("oops", StatusBadRequest, func() {
								if implicit {
									Body(Empty)
									Header("title:loom-attribute-detail")
								} else {
									Header("a:X-Same")
									Header("b:x-same")
								}
							})
						})
					})
				})
			})
			require.ErrorContains(t, err, "conflicting bindings")
		})
	}
}

func TestHTTPWireBindingCredentialAlternatives(t *testing.T) {
	for _, shared := range []bool{false, true} {
		name := "distinct credential fields"
		if shared {
			name = "same source different codec"
		}
		t.Run(name, func(t *testing.T) {
			requireWireBindingCollision(t, func() {
				jwt := JWTSecurity("jwt")
				key := APIKeySecurity("key")
				Service("credentials", func() {
					Method("either", func() {
						Security(jwt)
						Security(key)
						Payload(func() {
							Token("token", String, func() {
								if shared {
									Meta("security:apikey:key", "key")
								}
							})
							if !shared {
								APIKey("key", "key", String)
							}
						})
						HTTP(func() {
							GET("/")
						})
					})
				})
			})
		})
	}
}

func TestHTTPWireBindingExplicitBearerAlternative(t *testing.T) {
	expr.RunDSL(t, func() {
		basic := BasicAuthSecurity("basic")
		jwt := JWTSecurity("jwt")
		Service("explicit", func() {
			Method("read", func() {
				Security(basic)
				Security(jwt)
				Payload(func() {
					Username("user", String)
					Password("password", String)
					Token("token", String)
				})
				HTTP(func() {
					GET("/")
					Header("token:Authorization")
				})
			})
		})
	})
}

func TestHTTPWireBindingRemapControl(t *testing.T) {
	root := expr.RunDSL(t, func() {
		Service("remap", func() {
			HTTP(func() {
				Header("a:X-Old")
			})
			Method("read", func() {
				Payload(func() {
					Attribute("a", String)
					Attribute("b", String)
				})
				HTTP(func() {
					GET("/")
					Header("a:X-New")
					Header("b:X-Old")
				})
			})
		})
	})
	endpoint := root.API.HTTP.Service("remap").Endpoint("read")
	require.Equal(t, "X-New", endpoint.Headers.ElemName("a"))
	require.Equal(t, "X-Old", endpoint.Headers.ElemName("b"))
}

func TestHTTPWireBindingPath(t *testing.T) {
	requireWireBindingCollision(t, func() {
		Service("paths", func() {
			Method("read", func() {
				Payload(func() {
					Attribute("a", String)
					Attribute("b", String)
				})
				HTTP(func() {
					GET("/{id}")
					Param("a:id")
					Param("b:id")
				})
			})
		})
	})
}

func TestHTTPWireBindingSharedSource(t *testing.T) {
	t.Run("incompatible Basic and Bearer", func(t *testing.T) {
		requireWireBindingCollision(t, func() {
			basic := BasicAuthSecurity("basic")
			jwt := JWTSecurity("jwt")
			Service("shared", func() {
				Method("read", func() {
					Security(basic)
					Security(jwt)
					Payload(func() {
						Token("token", String, func() {
							Meta("security:username")
						})
						Password("password", String)
					})
					HTTP(func() {
						GET("/")
					})
				})
			})
		})
	})
	t.Run("same field and same Bearer codec", func(t *testing.T) {
		expr.RunDSL(t, func() {
			jwt := JWTSecurity("jwt")
			oauth := OAuth2Security("oauth", func() {
				AuthorizationCodeFlow("https://example.com/auth", "https://example.com/token", "")
			})
			Service("shared", func() {
				Method("read", func() {
					Security(jwt, oauth)
					Payload(func() {
						Token("token", String, func() {
							Meta("security:accesstoken")
						})
					})
					HTTP(func() {
						GET("/")
					})
				})
			})
		})
	})
}

func TestHTTPWireBindingSelectedSecurity(t *testing.T) {
	root := expr.RunDSL(t, func() {
		parent := APIKeySecurity("parent")
		local := APIKeySecurity("local")
		Service("scope", func() {
			Security(parent)
			Method("read", func() {
				Security(local)
				Payload(func() {
					APIKey("local", "active", String)
					APIKey("parent", "other", String)
				})
				HTTP(func() {
					POST("/")
				})
			})
		})
	})
	endpoint := root.API.HTTP.Service("scope").Endpoint("read")
	require.Len(t, endpoint.Requirements, 1)
	require.Equal(t, "local", endpoint.Requirements[0].Schemes[0].SchemeName)
	require.NotNil(t, endpoint.Body.Find("other"), "overridden auth must not remove an ordinary body field")
}

func TestHTTPWireBindingOrdinaryAuthorization(t *testing.T) {
	for _, name := range []string{"Authorization", "authorization"} {
		t.Run(name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, func() {
				basic := BasicAuthSecurity("basic")
				Service("ordinary", func() {
					Method("read", func() {
						Security(basic)
						Payload(func() {
							Username("user", String)
							Password("pass", String)
							Attribute("other", String)
						})
						HTTP(func() {
							GET("/")
							Header("other:" + name)
						})
					})
				})
			})
			require.ErrorContains(t, err, "conflicting bindings")
			require.ErrorContains(t, err, "basic encoding")
			require.ErrorContains(t, err, `attributes "other"`)
		})
	}
}

func TestHTTPWireBindingMissingCredential(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		jwt := JWTSecurity("jwt")
		Service("missing", func() {
			Method("read", func() {
				Security(jwt)
				HTTP(func() {
					GET("/")
				})
			})
		})
	})
	require.ErrorContains(t, err, "JWT")
}

func requireWireBindingCollision(t *testing.T, dsl func()) {
	t.Helper()
	err := expr.RunInvalidDSL(t, dsl)
	require.ErrorContains(t, err, "conflicting bindings")
}

func TestHTTPWireBindingJSONRPCStreamingSource(t *testing.T) {
	root := expr.RunDSL(t, func() {
		jwt := JWTSecurity("jwt")
		Service("stream", func() {
			JSONRPC(func() {
				GET("/rpc")
			})
			Method("read", func() {
				Security(jwt)
				Payload(func() {
					Token("token", String)
					Attribute("filter", String)
				})
				StreamingResult(String)
				JSONRPC(func() {
					Body(func() {
						Attribute("token")
						Attribute("filter")
					})
				})
			})
		})
	})
	endpoint := root.API.JSONRPC.Service("stream").Endpoint("read")
	require.Equal(t, "token", expr.TaggedAttribute(endpoint.MethodExpr.Payload, "security:token"))
	require.Equal(t, "body", endpoint.Requirements[0].Schemes[0].In)
	require.Equal(t, "token", endpoint.Requirements[0].Schemes[0].Name)
	require.True(t, endpoint.Headers.IsEmpty())
}
