package expr_test

import (
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/require"
)

func TestHTTPErrorResponseValidation(t *testing.T) {
	cases := []struct {
		Name  string
		DSL   func()
		Error string
	}{
		{"header string error", stringErrorResponseWithHeadersDSL, ""},
		{"header object result", objectErrorResponseWithHeadersDSL, ""},
		{"header array result", arrayErrorResponseWithHeadersDSL, ""},
		{"header map result", mapErrorResponseWithHeadersDSL, `HTTP response of service "MapErrorResponseWithHeaders" HTTP endpoint "Method": attribute "foo" used in HTTP headers must be a primitive type or an array of primitive types.`},
		{"implicit object in header", implicitObjectErrorResponseWithHeadersDSL, `HTTP response of service "ArrayObjectErrorResponseWithHeaders" HTTP endpoint "Method": attribute "foo" used in HTTP headers must be a primitive type or an array of primitive types.`},
		{"array of object in header", arrayObjectErrorResponseWithHeadersDSL, `HTTP response of service "ArrayObjectErrorResponseWithHeaders" HTTP endpoint "Method": Array error type is mapped to an HTTP header but is not an array of primitive types.`},
		{"map in header", mapErrorTypeResponseWithHeadersDSL, `HTTP response of service "MapErrorTypeResponseWithHeaders" HTTP endpoint "Method": error type must be a primitive type or an array of primitive types.`},
		{"missing header result attribute", missingHeaderErrorAttributeDSL, `HTTP response of service "MissingHeaderErrorAttribute" HTTP endpoint "Method": header "bar" has no equivalent attribute in error type, use notation 'attribute_name:header_name' to identify corresponding error type attribute.`},
		{"valid cookie mapping", validErrorResponseCookieDSL, ""},
		{"missing cookie error attribute", missingCookieErrorAttributeDSL, `HTTP response of service "MissingCookieErrorAttribute" HTTP endpoint "Method": cookie "sid" has no equivalent attribute in error type, use notation 'attribute_name:cookie_name' to identify corresponding error type attribute.
HTTP response of service "MissingCookieErrorAttribute" HTTP endpoint "Method": attribute "missing" used in HTTP cookies must be a primitive type.`},
		{"duplicate cookie name", duplicateErrorResponseCookieNameDSL, `HTTP response of service "DuplicateErrorResponseCookieName" HTTP endpoint "Method": response defines duplicate cookie "sid"`},
		{"duplicate cookie attribute", duplicateErrorResponseCookieAttributeDSL, `HTTP response of service "DuplicateErrorResponseCookieAttribute" HTTP endpoint "Method": response defines duplicate cookie mapping for attribute "session"`},
		{"non-primitive cookie error attribute", nonPrimitiveCookieErrorAttributeDSL, `HTTP response of service "NonPrimitiveCookieErrorAttribute" HTTP endpoint "Method": attribute "session" used in HTTP cookies must be a primitive type.`},
		{"empty error cookies", emptyErrorResponseWithCookiesDSL, `attribute: error type "boom" must not be Empty. Omit the type and map Body(Empty) on a bodyless HTTP response
HTTP response of service "EmptyErrorResponseWithCookies" HTTP endpoint "Method": response defines cookies but error type is empty`},
		{"array error cookies", arrayErrorResponseWithCookiesDSL, `HTTP response of service "ArrayErrorResponseWithCookies" HTTP endpoint "Method": Array error type is mapped to an HTTP cookie.`},
		{"insecure host cookie", insecureErrorResponseCookieDSL("InsecureHostErrorCookie", "__Host-session", false), `HTTP response of service "InsecureHostErrorCookie" HTTP endpoint "Method": cookie "__Host-session" requires CookieSecure because its name uses the "__Host-" prefix`},
		{"insecure secure-prefix cookie", insecureErrorResponseCookieDSL("InsecureSecurePrefixErrorCookie", "__Secure-session", false), `HTTP response of service "InsecureSecurePrefixErrorCookie" HTTP endpoint "Method": cookie "__Secure-session" requires CookieSecure because its name uses the "__Secure-" prefix`},
		{"insecure same-site-none cookie", insecureErrorResponseCookieDSL("InsecureSameSiteNoneErrorCookie", "session", true), `HTTP response of service "InsecureSameSiteNoneErrorCookie" HTTP endpoint "Method": cookie "session" requires CookieSecure when SameSite is None`},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			if c.Error == "" {
				expr.RunDSL(t, c.DSL)
			} else {
				err := expr.RunInvalidDSL(t, c.DSL)
				if err.Error() != c.Error {
					t.Errorf("\ngot error %q\nexpected %q", err.Error(), c.Error)
				}
			}
		})
	}
}

func TestUndeclaredHTTPErrorResponseValidation(t *testing.T) {
	tests := []struct {
		name string
		dsl  func()
		want []string
	}{
		{
			name: "one method response",
			dsl:  undeclaredMethodHTTPErrorResponses("example", "unauthenticated"),
			want: []string{
				`HTTP response of service "example" HTTP endpoint "list"`,
				`Error "unauthenticated" does not match an error defined in the method`,
			},
		},
		{
			name: "multiple method responses",
			dsl: undeclaredMethodHTTPErrorResponses(
				"example",
				"unauthenticated",
				"forbidden",
				"internal_error",
			),
			want: []string{
				`Error "unauthenticated" does not match an error defined in the method`,
				`Error "forbidden" does not match an error defined in the method`,
				`Error "internal_error" does not match an error defined in the method`,
			},
		},
		{
			name: "service helper responses",
			dsl: func() {
				Service("example-helper", func() {
					HTTP(func() {
						addUndeclaredHTTPErrorResponses("unauthenticated", "forbidden")
					})
					Method("list", func() {
						Result(ArrayOf(String))
						HTTP(func() {
							GET("/examples")
							Response(StatusOK)
						})
					})
				})
			},
			want: []string{
				`HTTP response of service "example-helper"`,
				`Error "unauthenticated" does not match an error defined in the service`,
				`Error "forbidden" does not match an error defined in the service`,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, test.dsl)
			diagnostic := err.Error()
			require.NotContains(t, diagnostic, "PANIC")
			for _, want := range test.want {
				require.Contains(t, diagnostic, want)
			}
		})
	}
}

func undeclaredMethodHTTPErrorResponses(service string, names ...string) func() {
	return func() {
		Service(service, func() {
			Method("list", func() {
				Result(ArrayOf(String))
				HTTP(func() {
					GET("/examples")
					Response(StatusOK)
					addUndeclaredHTTPErrorResponses(names...)
				})
			})
		})
	}
}

func addUndeclaredHTTPErrorResponses(names ...string) {
	for _, name := range names {
		Response(name, StatusBadRequest)
	}
}

var validErrorResponseCookieDSL = func() {
	Service("ValidErrorResponseCookie", func() {
		Method("Method", func() {
			Error("boom", func() {
				Attribute("session", String)
			})
			HTTP(func() {
				POST("/")
				Response("boom", StatusConflict, func() {
					Cookie("session:sid")
				})
			})
		})
	})
}

var missingCookieErrorAttributeDSL = func() {
	Service("MissingCookieErrorAttribute", func() {
		Method("Method", func() {
			Error("boom", func() {
				Attribute("session", String)
			})
			HTTP(func() {
				POST("/")
				Response("boom", StatusConflict, func() {
					Cookie("missing:sid")
				})
			})
		})
	})
}

var duplicateErrorResponseCookieNameDSL = func() {
	Service("DuplicateErrorResponseCookieName", func() {
		Method("Method", func() {
			Error("boom", func() {
				Attribute("session", String)
				Attribute("refresh", String)
			})
			HTTP(func() {
				POST("/")
				Response("boom", StatusConflict, func() {
					Cookie("session:sid")
					Cookie("refresh:sid")
				})
			})
		})
	})
}

var duplicateErrorResponseCookieAttributeDSL = func() {
	Service("DuplicateErrorResponseCookieAttribute", func() {
		Method("Method", func() {
			Error("boom", func() {
				Attribute("session", String)
			})
			HTTP(func() {
				POST("/")
				Response("boom", StatusConflict, func() {
					Cookie("session:sid")
					Cookie("session:other")
				})
			})
		})
	})
}

var nonPrimitiveCookieErrorAttributeDSL = func() {
	Service("NonPrimitiveCookieErrorAttribute", func() {
		Method("Method", func() {
			Error("boom", func() {
				Attribute("session", MapOf(String, String))
			})
			HTTP(func() {
				POST("/")
				Response("boom", StatusConflict, func() {
					Cookie("session")
				})
			})
		})
	})
}

var emptyErrorResponseWithCookiesDSL = func() {
	Service("EmptyErrorResponseWithCookies", func() {
		Method("Method", func() {
			Error("boom", Empty)
			HTTP(func() {
				POST("/")
				Response("boom", StatusConflict, func() {
					Cookie("session")
				})
			})
		})
	})
}

var arrayErrorResponseWithCookiesDSL = func() {
	Service("ArrayErrorResponseWithCookies", func() {
		Method("Method", func() {
			Error("boom", ArrayOf(String))
			HTTP(func() {
				POST("/")
				Response("boom", StatusConflict, func() {
					Cookie("session")
				})
			})
		})
	})
}

func insecureErrorResponseCookieDSL(serviceName, cookieName string, sameSiteNone bool) func() {
	return func() {
		Service(serviceName, func() {
			Method("Method", func() {
				Error("boom", func() {
					Attribute("session", String)
				})
				HTTP(func() {
					POST("/")
					Response("boom", StatusConflict, func() {
						SessionCookie("session:" + cookieName)
						CookieInsecure()
						if sameSiteNone {
							CookieSameSite(CookieSameSiteNone)
						}
					})
				})
			})
		})
	}
}

var stringErrorResponseWithHeadersDSL = func() {
	Service("StringErrorResponseWithHeaders", func() {
		Method("Method", func() {
			Error("error", String)
			HTTP(func() {
				POST("/")
				Response("error", func() {
					Header("Location")
				})
			})
		})
	})
}

var objectErrorResponseWithHeadersDSL = func() {
	Service("ObjectErrorResponseWithHeaders", func() {
		Method("Method", func() {
			Error("error", func() {
				Attribute("foo", String)
			})
			HTTP(func() {
				POST("/")
				Response("error", func() {
					Header("foo:Location")
				})
			})
		})
	})
}

var implicitObjectErrorResponseWithHeadersDSL = func() {
	Service("ArrayObjectErrorResponseWithHeaders", func() {
		Method("Method", func() {
			Error("error", func() {
				Attribute("foo", func() {
					Attribute("bar", String)
					Attribute("baz", String)
				})
			})
			HTTP(func() {
				POST("/")
				Response("error", func() {
					Header("foo:Location")
				})
			})
		})
	})
}

var arrayObjectErrorResponseWithHeadersDSL = func() {
	var Obj = Type("Obj", func() {
		Attribute("foo", String)
	})
	Service("ArrayObjectErrorResponseWithHeaders", func() {
		Method("Method", func() {
			Error("error", ArrayOf(Obj))
			HTTP(func() {
				POST("/")
				Response("error", func() {
					Header("foo:Location")
				})
			})
		})
	})
}

var mapErrorTypeResponseWithHeadersDSL = func() {
	Service("MapErrorTypeResponseWithHeaders", func() {
		Method("Method", func() {
			Error("error", MapOf(String, Int))
			HTTP(func() {
				POST("/")
				Response("error", func() {
					Header("Location")
				})
			})
		})
	})
}

var arrayErrorResponseWithHeadersDSL = func() {
	Service("ArrayErrorResponseWithHeaders", func() {
		Method("Method", func() {
			Error("error", func() {
				Attribute("foo", ArrayOf(String))
			})
			HTTP(func() {
				POST("/")
				Response("error", func() {
					Header("foo:Location")
				})
			})
		})
	})
}

var mapErrorResponseWithHeadersDSL = func() {
	Service("MapErrorResponseWithHeaders", func() {
		Method("Method", func() {
			Error("error", func() {
				Attribute("foo", MapOf(String, String))
			})
			HTTP(func() {
				POST("/")
				Response("error", func() {
					Header("foo:Location")
				})
			})
		})
	})
}

var missingHeaderErrorAttributeDSL = func() {
	Service("MissingHeaderErrorAttribute", func() {
		Method("Method", func() {
			Error("error")
			HTTP(func() {
				POST("/")
				Response("error", func() {
					Header("bar")
				})
			})
		})
	})
}
