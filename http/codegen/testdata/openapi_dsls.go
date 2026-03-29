package testdata

import . "github.com/CaliLuke/loom/dsl"

var SimpleDSL = func() {
	var PayloadT = Type("Payload", func() {
		Attribute("string", String, func() {
			Example("")
		})
	})
	var ResultT = Type("Result", func() {
		Attribute("string", String, func() {
			Example("")
		})
	})
	var _ = API("test", func() {
		Server("test", func() {
			Host("localhost", func() {
				URI("https://loom.design")
			})
		})
	})
	Service("testService", func() {
		Method("testEndpoint", func() {
			Payload(PayloadT)
			Result(ResultT)
			HTTP(func() {
				GET("/")
			})
		})
	})
}

var MultipleServicesDSL = func() {
	var PayloadT = Type("Payload", func() {
		Attribute("string", String, func() {
			Example("")
		})
	})
	var ResultT = Type("Result", func() {
		Attribute("string", String, func() {
			Example("")
		})
	})
	var _ = API("test", func() {
		Server("test", func() {
			Services("testService", "anotherTestService")
			Host("localhost", func() {
				URI("https://loom.design")
			})
		})
	})
	Service("testService", func() {
		Method("testEndpoint", func() {
			Payload(PayloadT)
			Result(ResultT)
			HTTP(func() {
				GET("/")
			})
		})
	})
	Service("anotherTestService", func() {
		Method("testEndpoint", func() {
			Payload(PayloadT)
			Result(ResultT)
			HTTP(func() {
				POST("/")
			})
		})
	})
}

var MultipleViewsDSL = func() {
	var ResultT = ResultType("application/json", func() {
		ContentType("application/vnd.custom+json")
		TypeName("Result")
		Attributes(func() {
			Attribute("string", String, func() {
				Example("")
			})
			Attribute("int", Int, func() {
				Example(1)
			})
		})
		View("default", func() {
			Attribute("string")
			Attribute("int")
		})
		View("tiny", func() {
			Attribute("string")
		})
	})
	Service("testService", func() {
		Method("testEndpointDefault", func() {
			Result(ResultT)
			HTTP(func() {
				GET("/")
				Response(StatusOK, func() {
					ContentType("application/custom+json")
				})
			})
		})
		Method("testEndpointTiny", func() {
			Result(ResultT, func() {
				View("tiny")
			})
			HTTP(func() {
				GET("/tiny")
			})
		})
	})
}

var ExplicitViewDSL = func() {
	var ResultT = ResultType("application/json", func() {
		TypeName("Result")
		Attributes(func() {
			Attribute("string", String, func() {
				Example("")
			})
			Attribute("int", Int, func() {
				Example(1)
			})
		})
		View("tiny", func() {
			Attribute("string")
		})
	})
	Service("testService", func() {
		Method("testEndpointDefault", func() {
			Result(ResultT, func() {
				View("default")
			})
			HTTP(func() {
				GET("/")
			})
		})
		Method("testEndpointTiny", func() {
			Result(ResultT, func() {
				View("tiny")
			})
			HTTP(func() {
				GET("/tiny")
			})
		})
	})
}

var InvalidDSL = func() {
	var _ = API("test", func() {
		Server("test", func() {
			Host("localhost", func() {
				URI("http://[::1]:namedport") // invalid URL
			})
		})
	})
	Service("httpService", func() {
		Method("httpEndpoint", func() {
			HTTP(func() { GET("/") })
		})
	})
}

var EmptyDSL = func() {
	var _ = API("test", func() {
		Server("test", func() {
			Host("localhost", func() {
				URI("https://loom.design")
			})
		})
	})
}

var FileServiceDSL = func() {
	var _ = Service("service-name", func() {
		Files("path1", "filename")
		Files("path2", "filename", func() {
			Meta("openapi:tag:user-tag")
		})
	})
}

// FileServiceWildcardDSL defines a service with a file server using a wildcard path.
var FileServiceWildcardDSL = func() {
	var _ = API("test", func() {
		Server("test", func() {
			Host("localhost", func() {
				URI("http://localhost:80")
			})
		})
	})
	var _ = Service("front", func() {
		Files("/ui/{*filepath}", "ui/dist", func() {
			Meta("openapi:summary", "Download ui/dist")
			Meta("openapi:tag:front")
		})
	})
}

var StringValidationDSL = func() {
	var _ = API("test", func() {
		Server("test", func() {
			Host("localhost", func() {
				URI("https://loom.design")
			})
		})
	})
	Service("testService", func() {
		Method("testEndpoint", func() {
			Payload(String, func() {
				MinLength(0)
				MaxLength(42)
				Example("")
			})
			Result(String, func() {
				MinLength(0)
				MaxLength(42)
				Example("")
			})
			HTTP(func() {
				POST("/")
				Response(StatusOK)
			})
		})
	})
}

var IntValidationDSL = func() {
	var _ = API("test", func() {
		Server("test", func() {
			Host("localhost", func() {
				URI("https://loom.design")
			})
		})
	})
	Service("testService", func() {
		Method("testEndpoint", func() {
			Payload(Int, func() {
				Minimum(0)
				Maximum(42)
				Example(1)
			})
			Result(Int, func() {
				Minimum(0)
				Maximum(42)
				Example(1)
			})
			HTTP(func() {
				POST("/")
				Response(StatusOK)
			})
		})
	})
}

var ArrayValidationDSL = func() {
	var Bar = Type("bar", func() {
		Attribute("string", String, func() {
			MinLength(0)
			MaxLength(42)
			Example("")
		})
	})
	var FooBar = Type("foobar", func() {
		Attribute("foo", ArrayOf(String), func() {
			MinLength(0)
			MaxLength(42)
		})
		Attribute("bar", ArrayOf(Bar), func() {
			MinLength(0)
			MaxLength(42)
		})
	})
	var _ = API("test", func() {
		Server("test", func() {
			Host("localhost", func() {
				URI("https://loom.design")
			})
		})
	})
	Service("testService", func() {
		Method("testEndpoint", func() {
			Payload(ArrayOf(FooBar))
			Result(String, func() {
				MinLength(0)
				MaxLength(42)
				Example("")
			})
			HTTP(func() {
				POST("/")
				Response(StatusOK)
			})
		})
	})
}

var ExtensionDSL = func() {
	var PayloadT = Type("Payload", func() {
		Attribute("string", String, func() {
			Example("")
			Meta("openapi:extension:x-test-schema", "Payload")
		})
	})
	var ResultT = Type("Result", func() {
		Attribute("string", String, func() {
			Example("")
			Meta("openapi:extension:x-test-schema", "Result")
		})
	})
	var _ = API("test", func() {
		Server("test", func() {
			Host("localhost", func() {
				URI("https://loom.design")
			})
		})
		Meta("openapi:extension:x-test-api", "API")
		Meta("openapi:tag:Backend")
		Meta("openapi:tag:Backend:desc", "Description of Backend")
		Meta("openapi:tag:Backend:url", "http://example.com")
		Meta("openapi:tag:Backend:url:desc", "See more docs here")
		Meta("openapi:tag:Backend:extension:x-data", `{"foo":"bar"}`)
	})
	Service("testService", func() {
		Method("testEndpoint", func() {
			Payload(PayloadT)
			Result(ResultT)
			HTTP(func() {
				POST("/")
				Meta("openapi:extension:x-test-foo", "bar")
			})
			Meta("openapi:extension:x-test-operation", "Operation")
		})
	})
}

var SecurityDSL = func() {
	var JWTAuth = JWTSecurity("jwt", func() {
		Description(`Secures endpoint by requiring a valid JWT token retrieved via the signin endpoint. Supports scopes "api:read" and "api:write".`)
		Scope("api:read", "Read-only access")
		Scope("api:write", "Read and write access")
	})

	var APIKeyAuth = APIKeySecurity("api_key", func() {
		Description("Secures endpoint by requiring an API key.")
	})

	var BasicAuth = BasicAuthSecurity("basic", func() {
		Description("Basic authentication used to authenticate security principal during signin")
	})

	var OAuth2Auth = OAuth2Security("oauth2", func() {
		AuthorizationCodeFlow("http://loom.design/authorization", "http://loom.design/token", "http://loom.design/refresh")
		Description(`Secures endpoint by requiring a valid OAuth2 token retrieved via the signin endpoint. Supports scopes "api:read" and "api:write".`)
		Scope("api:read", "Read-only access")
		Scope("api:write", "Read and write access")
	})

	Service("testService", func() {
		Method("testEndpointA", func() {
			Security(BasicAuth, OAuth2Auth, JWTAuth, APIKeyAuth, func() {
				Scope("api:read")
			})
			Payload(func() {
				Username("username", String)
				Password("password", String)
				APIKey("api_key", "key", String)
				Token("token", String)
				AccessToken("oauth_token", String)
				Required("username", "password", "key", "token", "oauth_token")
			})
			HTTP(func() {
				GET("/")
				Header("oauth_token:Token")
				Param("key:k")
				Header("token:X-Authorization")
			})
		})
		Method("testEndpointB", func() {
			Security(APIKeyAuth)
			Security(OAuth2Auth, func() {
				Scope("api:read")
				Scope("api:write")
			})
			Payload(func() {
				APIKey("api_key", "key", String)
				AccessToken("oauth_token", String)
				Required("key", "oauth_token")
			})
			HTTP(func() {
				POST("/")
				Param("oauth_token:auth")
				Header("key:Authorization")
			})
		})
	})
}

var ServerHostWithVariablesDSL = func() {
	var _ = API("test", func() {
		Server("test", func() {
			Host("localhost", func() {
				URI("https://{version}.loom.design")
				Variable("version", String, "API Version", func() {
					Default("v1")
				})
			})
		})
	})
	Service("testService", func() {
		Method("testEndpoint", func() {
			Payload(Empty)
			Result(Empty)
			HTTP(func() {
				POST("/")
			})
		})
	})
}

var WithSpacesDSL = func() {
	var Bar = Type("bar", func() {
		Attribute("string", String, func() {
			Example("")
		})
	})
	var FooBar = ResultType("application/vnd.loom.foobar", func() {
		TypeName("Foo Bar")
		Attribute("foo", String, func() {
			Example("")
		})
		Attribute("bar", ArrayOf(Bar))
	})
	Service("test service", func() {
		Method("test endpoint", func() {
			Payload(Bar)
			Result(FooBar)
			HTTP(func() {
				POST("/")
				Response(StatusOK)
				Response(StatusNotFound)
			})
		})
	})
}

var WithMapDSL = func() {
	var Bar = Type("bar", func() {
		Attribute("string", String, func() {
			Example("")
		})
	})
	var FooBar = ResultType("application/vnd.loom.foobar", func() {
		TypeName("Foo Bar")
		Attribute("foo", String, func() {
			Example("")
		})
		Attribute("bar", ArrayOf(Bar))
	})
	Service("test service", func() {
		Method("test endpoint", func() {
			Payload(func() {
				Attribute("int_map", MapOf(String, Int, func() {
					Key(func() { Example("") })
					Elem(func() { Example(1) })
				}))
				Attribute("uint_map", MapOf(String, UInt, func() {
					Key(func() { Example("") })
					Elem(func() { Example(uint(1)) })
				}))
				Attribute("type_map", MapOf(String, Bar), func() {
					Key(func() { Example("") })
				})
			})
			Result(func() {
				Attribute("uint32_map", MapOf(String, UInt32, func() {
					Key(func() { Example("") })
					Elem(func() { Example(uint32(1)) })
				}))
				Attribute("uint64_map", MapOf(String, UInt64, func() {
					Key(func() { Example("") })
					Elem(func() { Example(uint64(1)) })
				}))
				Attribute("resulttype_map", MapOf(String, FooBar, func() {
					Key(func() { Example("") })
				}))
			})
			HTTP(func() {
				POST("/")
			})
		})
	})
}

var WithAnyDSL = func() {
	Service("testService", func() {
		Method("testEndpoint", func() {
			Payload(func() {
				Attribute("any", Any, func() {
					Example("")
				})
				Attribute("any_array", ArrayOf(Any, func() {
					Example("")
				}))
				Attribute("any_map", MapOf(String, Any), func() {
					Key(func() { Example("") })
					Elem(func() { Example("") })
				})
			})
			Result(func() {
				Attribute("any", Any, func() {
					Example("")
				})
				Attribute("any_array", ArrayOf(Any, func() {
					Example("")
				}))
				Attribute("any_map", MapOf(String, Any), func() {
					Key(func() { Example("") })
					Elem(func() { Example("") })
				})
			})
			HTTP(func() {
				POST("/")
			})
		})
	})
}

var PathWithWildcardDSL = func() {
	Service("test service", func() {
		Method("test endpoint", func() {
			Payload(func() {
				Attribute("int_map", Int)
			})
			HTTP(func() {
				POST("/{*int_map}")
			})
		})
	})
}

var PathWithMultipleWildcardDSL = func() {
	Service("test service", func() {
		Method("test endpoint", func() {
			Payload(func() {
				Attribute("foo", Int)
				Attribute("bar", Int)
			})
			HTTP(func() {
				POST("/{bar}")
			})
		})
		HTTP(func() {
			Path("/{foo}")
		})
	})
}

var PathWithMultipleExplicitWildcardDSL = func() {
	Service("test service", func() {
		Method("test endpoint", func() {
			Payload(func() {
				Attribute("foo", Int)
				Attribute("bar", Int)
			})
			HTTP(func() {
				POST("/{bar}")
				Param("bar")
			})
		})
		HTTP(func() {
			Path("/{foo}")
			Param("foo")
		})
	})
}

var HeadersDSL = func() {
	Service("test service", func() {
		Method("test endpoint", func() {
			Payload(func() {
				Attribute("foo", Int)
				Attribute("bar", Int)
			})
			HTTP(func() {
				POST("/")
				Header("bar")
			})
		})
		HTTP(func() {
			Header("foo")
		})
	})
}

var WithTagsDSL = func() {
	Service("test service", func() {
		HTTP(func() {
			Meta("openapi:tag:SomeTag:desc", "Endpoint description")
			Meta("openapi:tag:SomeTag:url", "Endpoint URL")
			Meta("openapi:tag:AnotherTag:desc", "Endpoint description")
			Meta("openapi:tag:AnotherTag:url", "Endpoint URL")
		})
		Method("test endpoint", func() {
			Payload(func() {
				Attribute("int_map", Int)
			})
			HTTP(func() {
				Meta("openapi:tag:SomeTag")
				POST("/{*int_map}")
			})
		})
		Method("another test endpoint", func() {
			Payload(func() {
				Attribute("int_map", Int)
			})
			HTTP(func() {
				Meta("openapi:generate", "false")
				Meta("openapi:tag:AnotherTag")
				POST("/{*int_map}")
			})
		})
	})
	Service("another test service", func() {
		Meta("openapi:generate", "false")
		HTTP(func() {
			Meta("openapi:tag:AnotherService:desc", "Another service description")
		})
		Method("another test endpoint", func() {
			Payload(func() {
				Attribute("int_map", Int)
			})
			HTTP(func() {
				Meta("openapi:tag:AnotherService")
				POST("/{*int_map}")
			})
		})
	})
}

var TypenameDSL = func() {
	var _ = API("test", func() {
		Server("test", func() {
			Host("localhost", func() {
				URI("https://loom.design")
			})
		})
	})

	var Foo = Type("Foo", func() {
		Meta("openapi:typename", "FooPayload")
		Attribute("value", String, func() {
			Example("")
		})
	})

	var Bar = ResultType("application/vnd.loom.example.bar", func() {
		TypeName("Bar")
		Meta("openapi:typename", "BarResult")
		Attribute("value", String, func() {
			Example("")
		})
	})

	var _ = Service("testService", func() {
		Method("foo", func() {
			Payload(Foo)
			Result(Bar, func() {
				Meta("openapi:typename", "FooResult")
			})
			HTTP(func() {
				POST("/foo")
			})
		})
		Method("bar", func() {
			Payload(Foo, func() {
				Meta("openapi:typename", "BarPayload")
			})
			Result(Bar)
			HTTP(func() {
				POST("/bar")
			})
		})
		Method("baz", func() {
			Payload(func() {
				Meta("openapi:typename", "BazPayload")
				Attribute("value", String, func() {
					Example("")
				})
			})
			Result(func() {
				Meta("openapi:typename", "BazResult")
				Attribute("value", String, func() {
					Example("")
				})
			})
			HTTP(func() {
				POST("/baz")
			})
		})
	})
}

var OpenAPISchemaDedupDSL = func() {
	var Alpha = Type("AlphaSchemaDedup", func() {
		Attribute("alpha", String)
		Required("alpha")
	})

	var Beta = Type("BetaSchemaDedup", func() {
		Attribute("beta", String)
		Required("beta")
	})

	Service("dedupService", func() {
		Method("first", func() {
			Payload(func() {
				Attribute("name", String)
				Required("name")
			})
			HTTP(func() {
				POST("/first")
			})
		})

		Method("second", func() {
			Payload(func() {
				Attribute("name", String)
				Required("name")
			})
			HTTP(func() {
				POST("/second")
			})
		})

		Method("union_first", func() {
			Payload(OneOf(Alpha, Beta))
			HTTP(func() {
				POST("/union/first")
			})
		})

		Method("union_second", func() {
			Payload(OneOf(Alpha, Beta))
			HTTP(func() {
				POST("/union/second")
			})
		})
	})
}

var OpenAPIParameterComponentsDSL = func() {
	Service("componentService", func() {
		Method("listWidgets", func() {
			Payload(func() {
				Attribute("widgetID", String, func() {
					Description("Widget identifier.")
					Example("widget-123")
				})
				Attribute("limit", Int, func() {
					Description("Page size.")
					Example(25)
					Minimum(1)
				})
				Required("widgetID")
			})
			Result(String)
			HTTP(func() {
				GET("/widgets/{widgetID}")
				Param("widgetID")
				Param("limit")
				Response(StatusOK)
			})
		})

		Method("listGadgets", func() {
			Payload(func() {
				Attribute("widgetID", String, func() {
					Description("Widget identifier.")
					Example("widget-123")
				})
				Attribute("limit", Int, func() {
					Description("Page size.")
					Example(25)
					Minimum(1)
				})
				Required("widgetID")
			})
			Result(String)
			HTTP(func() {
				GET("/gadgets/{widgetID}")
				Param("widgetID")
				Param("limit")
				Response(StatusOK)
			})
		})
	})
}

var OpenAPIReusableComponentsDSL = func() {
	var Credentials = Type("Credentials", func() {
		Attribute("username", String, func() {
			Example("alice")
		})
		Attribute("password", String, func() {
			Meta("openapi:writeOnly", "true")
			Example("hunter2")
		})
		Attribute("legacyToken", String, func() {
			Meta("openapi:deprecated", "true")
			Example("legacy-token")
		})
		Attribute("profile", String, func() {
			Meta("openapi:contentEncoding", "base64")
			Meta("openapi:contentMediaType", "application/json")
			Example("eyJyb2xlIjoiYWRtaW4ifQ==")
		})
		Required("username", "password")
		Example("primary", Val{
			"username":    "alice",
			"password":    "hunter2",
			"legacyToken": "legacy-token",
			"profile":     "eyJyb2xlIjoiYWRtaW4ifQ==",
		})
		Example("backup", Val{
			"username":    "bob",
			"password":    "swordfish",
			"legacyToken": "legacy-token-2",
			"profile":     "eyJyb2xlIjoiZWRpdG9yIn0=",
		})
	})

	var Session = Type("Session", func() {
		Attribute("token", String, func() {
			Meta("openapi:readOnly", "true")
			Example("token-123")
		})
		Attribute("traceID", String, func() {
			Example("trace_123")
		})
		Attribute("expiresAt", String, func() {
			Format(FormatDateTime)
			Example("2026-03-20T12:00:00Z")
		})
		Required("token", "traceID", "expiresAt")
		Example("current", Val{
			"token":     "token-123",
			"traceID":   "trace_123",
			"expiresAt": "2026-03-20T12:00:00Z",
		})
		Example("next", Val{
			"token":     "token-456",
			"traceID":   "trace_456",
			"expiresAt": "2026-03-21T12:00:00Z",
		})
	})

	Service("auth", func() {
		HTTP(func() {
			Meta("openapi:tag:Auth")
			Meta("openapi:tag:Auth:desc", "Authentication operations.")
		})

		Method("signin", func() {
			Payload(Credentials)
			Result(Session)
			HTTP(func() {
				POST("/auth/signin")
				Response(StatusOK, func() {
					Header("traceID:X-Trace-ID", String, func() {
						Example("trace-a", "trace_123")
						Example("trace-b", "trace_456")
					})
				})
			})
		})

		Method("refresh", func() {
			Payload(Credentials)
			Result(Session)
			HTTP(func() {
				POST("/auth/refresh")
				Response(StatusOK, func() {
					Header("traceID:X-Trace-ID", String, func() {
						Example("trace-a", "trace_123")
						Example("trace-b", "trace_456")
					})
				})
			})
		})

		Method("revoke", func() {
			Payload(Empty)
			Result(Empty)
			HTTP(func() {
				POST("/auth/revoke")
				Response(StatusNoContent)
			})
		})

		Method("signout", func() {
			Payload(Empty)
			Result(Empty)
			HTTP(func() {
				POST("/auth/signout")
				Response(StatusNoContent)
			})
		})
	})
}

var OpenAPIExplicitReusableComponentNamesDSL = func() {
	var SearchFilters = Type("SearchFilters", func() {
		Meta("openapi:component:requestBody", "SearchFiltersRequest")
		Attribute("query", String, func() {
			Example("soup")
		})
		Required("query")
	})

	Service("componentNames", func() {
		Method("searchRecipes", func() {
			Payload(func() {
				Attribute("widgetID", String, func() {
					Meta("openapi:component:parameter", "WidgetIDParam")
					Example("widget-123")
				})
				Attribute("body", SearchFilters)
				Required("widgetID", "body")
			})
			Result(String)
			HTTP(func() {
				POST("/recipes/{widgetID}/search")
				Param("widgetID")
				Body("body")
				Response(StatusOK)
			})
		})

		Method("searchGadgets", func() {
			Payload(func() {
				Attribute("widgetID", String, func() {
					Meta("openapi:component:parameter", "WidgetIDParam")
					Example("widget-123")
				})
				Attribute("body", SearchFilters)
				Required("widgetID", "body")
			})
			Result(String)
			HTTP(func() {
				POST("/gadgets/{widgetID}/search")
				Param("widgetID")
				Body("body")
				Response(StatusOK)
			})
		})
	})
}

var OpenAPIRequestResponseSplitDSL = func() {
	var Account = Type("Account", func() {
		Attribute("id", String, func() {
			Meta("openapi:readOnly", "true")
			Example("acct_123")
		})
		Attribute("email", String, func() {
			Format(FormatEmail)
			Example("user@example.test")
		})
		Attribute("password", String, func() {
			Meta("openapi:writeOnly", "true")
			Example("hunter2")
		})
		Required("id", "email", "password")
	})

	Service("accounts", func() {
		Method("create", func() {
			Payload(Account)
			Result(Account)
			HTTP(func() {
				POST("/accounts")
				Response(StatusCreated)
			})
		})
	})
}

var OpenAPIProblemLinksAsyncDSL = func() {
	var ThreadSummary = Type("OpenAPIThreadSummary", func() {
		Attribute("thread_id", String, func() {
			Example("thr_42")
			Pattern("^thr_[0-9]+$")
		})
		Attribute("title", String, func() {
			Example("Release freeze follow-up")
		})
		Required("thread_id", "title")
	})
	var ThreadAccepted = Type("OpenAPIThreadAccepted", func() {
		Attribute("thread_id", String, func() {
			Example("thr_42")
			Pattern("^thr_[0-9]+$")
		})
		Attribute("watch_url", String, func() {
			Example("https://api.contract-surfaces.example.test/threads/thr_42/events")
		})
		Required("thread_id", "watch_url")
	})
	var ThreadEvent = Type("OpenAPIThreadEvent", func() {
		Attribute("id", String, func() {
			Example("evt_8")
		})
		Attribute("event", String, func() {
			Example("thread.message_posted")
		})
		Attribute("data", func() {
			Attribute("author", String, func() {
				Example("alice")
			})
			Attribute("preview", String, func() {
				Example("Shipping the OpenAPI cleanup next.")
			})
			Required("author", "preview")
		})
		Required("id", "event", "data")
	})

	var _ = API("contract-surfaces", func() {
		Title("Contract Surfaces API")
		Description("Exercises problem documents, workflow links, and async streaming contracts.")
		Meta("openapi:closed-objects", "true")
		Server("contract-surfaces", func() {
			Host("api", func() {
				URI("https://api.contract-surfaces.example.test")
			})
		})
	})

	Service("threadOps", func() {
		Description("Thread creation and retrieval operations.")
		Error("not_found", ProblemResult)
		Error("conflict", ProblemResult)
		Error("busy", ProblemResult)
		HTTP(func() {
			Response("not_found", StatusNotFound)
			Response("conflict", StatusConflict)
			Response("busy", StatusTooManyRequests)
		})

		Method("getThread", func() {
			NoSecurity()
			Payload(func() {
				Attribute("thread_id", String, func() {
					Example("thr_42")
				})
				Required("thread_id")
			})
			Result(ThreadSummary)
			HTTP(func() {
				GET("/threads/{thread_id}")
				Param("thread_id")
				Response(StatusOK)
			})
		})

		Method("createThread", func() {
			NoSecurity()
			Payload(func() {
				Attribute("title", String, func() {
					Example("Release freeze follow-up")
				})
				Required("title")
			})
			Result(ThreadAccepted)
			HTTP(func() {
				POST("/threads")
				Response(StatusAccepted, func() {
					Link("thread", func() {
						LinkOperation("getThread")
						LinkParam("thread_id", "$response.body#/thread_id")
					})
					Link("watch", func() {
						LinkOperation("watchThread")
						LinkParam("thread_id", "$response.body#/thread_id")
					})
				})
			})
		})

		Method("reopenThread", func() {
			NoSecurity()
			Payload(func() {
				Attribute("thread_id", String, func() {
					Example("thr_42")
				})
				Required("thread_id")
			})
			Result(ThreadAccepted)
			HTTP(func() {
				POST("/threads/{thread_id}/reopen")
				Param("thread_id")
				Response(StatusAccepted, func() {
					Link("thread", func() {
						LinkOperation("getThread")
						LinkParam("thread_id", "$response.body#/thread_id")
					})
					Link("watch", func() {
						LinkOperation("watchThread")
						LinkParam("thread_id", "$response.body#/thread_id")
					})
				})
			})
		})

		Method("archiveThread", func() {
			NoSecurity()
			Payload(func() {
				Attribute("thread_id", String, func() {
					Example("thr_42")
				})
				Required("thread_id")
			})
			Result(Empty)
			HTTP(func() {
				POST("/threads/{thread_id}/archive")
				Param("thread_id")
				Response(StatusNoContent)
			})
		})

		Method("watchThread", func() {
			NoSecurity()
			Payload(func() {
				Attribute("thread_id", String, func() {
					Example("thr_42")
				})
				Attribute("last_event_id", String, func() {
					Example("evt_7")
				})
				Required("thread_id")
			})
			StreamingResult(ThreadEvent)
			HTTP(func() {
				GET("/threads/{thread_id}/events")
				Param("thread_id")
				Param("last_event_id")
				ServerSentEvents(func() {
					SSERequestID("last_event_id")
					SSEEventID("id")
				})
			})
		})
	})

	Service("opsSocket", func() {
		Description("Bidirectional operator control channel.")

		Method("streamCommands", func() {
			NoSecurity()
			Payload(func() {
				Attribute("channel", String, func() {
					Example("deployments")
				})
				Required("channel")
			})
			StreamingPayload(func() {
				Attribute("op", String, func() {
					Example("pause")
				})
				Attribute("target", String, func() {
					Example("worker-eu-1")
				})
				Required("op", "target")
			})
			StreamingResult(func() {
				Attribute("event", String, func() {
					Example("command.accepted")
				})
				Attribute("ok", Boolean, func() {
					Example(true)
				})
				Required("event", "ok")
			})
			HTTP(func() {
				GET("/ws/ops/{channel}")
				Param("channel")
			})
		})
	})
}

var SkipResponseBodyEncodeDecodeDSL = func() {
	Service("testService", func() {
		Method("empty", func() {
			Payload(Empty)
			Result(Empty)
			HTTP(func() {
				GET("/empty")
			})
		})
		Method("empty_ok", func() {
			Payload(Empty)
			Result(Empty)
			HTTP(func() {
				GET("/empty/ok")
				Response(StatusOK)
			})
		})
		Method("binary", func() {
			Payload(Empty)
			Result(Empty)
			HTTP(func() {
				GET("/binary")
				SkipResponseBodyEncodeDecode()
				Response(StatusOK, func() {
					ContentType("image/png")
				})
			})
		})
		Method("html", func() {
			Payload(Empty)
			Result(Empty)
			HTTP(func() {
				GET("/html")
				SkipResponseBodyEncodeDecode()
				Response(StatusOK, func() {
					ContentType("text/html")
					OpenAPIBody(String)
				})
			})
		})
	})
}

var NotGenerateServerDSL = func() {
	var _ = API("test", func() {
		Server("test", func() {
			Host("localhost", func() {
				URI("https://loom.design")
			})
			Meta("openapi:generate", "false")
		})
	})
	Service("testService", func() {
		Method("testEndpoint", func() {
			Result(String)
			HTTP(func() {
				GET("/")
			})
		})
	})
}

var NotGenerateHostDSL = func() {
	var _ = API("test", func() {
		Server("test", func() {
			Host("localhost", func() {
				URI("https://loom.design")
				Meta("openapi:generate", "false")
			})
		})
	})
	Service("testService", func() {
		Method("testEndpoint", func() {
			Result(String)
			HTTP(func() {
				GET("/")
			})
		})
	})
}

var NotGenerateAttributeDSL = func() {
	var _ = API("test", func() {
		Server("test", func() {
			Host("localhost", func() {
				URI("https://loom.design")
			})
		})
	})
	var PayloadT = Type("Payload", func() {
		Attribute("int", Int, func() {
			Meta("openapi:generate", "false")
		})
		Attribute("string", String, func() {
			Example("")
		})
		Attribute("required_int", Int, func() {
			Meta("openapi:generate", "false")
		})
		Attribute("required_string", String, func() {
			Example("")
		})
		Required("required_int", "required_string")
	})
	var ResultT = Type("Result", func() {
		Attribute("int", Int, func() {
			Example(0)
		})
		Attribute("string", String, func() {
			Meta("openapi:generate", "false")
		})
		Attribute("required_int", Int, func() {
			Example(0)
		})
		Attribute("required_string", String, func() {
			Meta("openapi:generate", "false")
		})
		Required("required_int", "required_string")
	})
	Service("testService", func() {
		Method("testEndpoint", func() {
			Payload(PayloadT)
			Result(ResultT)
			HTTP(func() {
				GET("/")
			})
		})
	})
}

var JSONPrefixDSL = func() {
	var _ = API("test", func() {
		Server("test", func() {
			Host("localhost", func() {
				URI("https://loom.design")
			})
		})
		Meta("openapi:json:prefix", "  ")
	})
	var _ = Service("service-name", func() {
		Files("path1", "filename")
		Files("path2", "filename", func() {
			Meta("openapi:tag:user-tag")
		})
	})
}

var JSONIndentDSL = func() {
	var _ = API("test", func() {
		Server("test", func() {
			Host("localhost", func() {
				URI("https://loom.design")
			})
		})
		Meta("openapi:json:indent", "  ")
	})
	var _ = Service("service-name", func() {
		Files("path1", "filename")
		Files("path2", "filename", func() {
			Meta("openapi:tag:user-tag")
		})
	})
}

var JSONPrefixIndentDSL = func() {
	var _ = API("test", func() {
		Server("test", func() {
			Host("localhost", func() {
				URI("https://loom.design")
			})
		})
		Meta("openapi:json:prefix", " ")
		Meta("openapi:json:indent", "  ")
	})
	var _ = Service("service-name", func() {
		Files("path1", "filename")
		Files("path2", "filename", func() {
			Meta("openapi:tag:user-tag")
		})
	})
}

var AdditionalPropertiesTypeDSL = func() {
	var PayloadT = Type("Payload", func() {
		Attribute("string", String, func() {
			Example("")
		})
		Meta("openapi:additionalProperties", "false")
	})
	var ResultT = Type("Result", func() {
		Attribute("string", String, func() {
			Example("")
		})
		Meta("openapi:additionalProperties", "false")
	})
	var _ = API("test", func() {
		Server("test", func() {
			Host("localhost", func() {
				URI("https://loom.design")
			})
		})
	})
	Service("testService", func() {
		Method("testEndpoint", func() {
			Payload(PayloadT)
			Result(ResultT)
			HTTP(func() {
				GET("/")
			})
		})
	})
}

var AdditionalPropertiesPayloadResultDSL = func() {
	var PayloadT = Type("Payload", func() {
		Attribute("string", String, func() {
			Example("")
		})
	})
	var ResultT = Type("Result", func() {
		Attribute("string", String, func() {
			Example("")
		})
	})
	var _ = API("test", func() {
		Server("test", func() {
			Host("localhost", func() {
				URI("https://loom.design")
			})
		})
	})
	Service("testService", func() {
		Method("testEndpoint", func() {
			Payload(PayloadT, func() {
				Meta("openapi:additionalProperties", "false")
			})
			Result(ResultT, func() {
				Meta("openapi:additionalProperties", "false")
			})
			HTTP(func() {
				GET("/")
			})
		})
	})
}

var AdditionalPropertiesEmbeddedPayloadResultDSL = func() {
	var _ = API("test", func() {
		Server("test", func() {
			Host("localhost", func() {
				URI("https://loom.design")
			})
		})
	})
	Service("testService", func() {
		Method("testEndpoint", func() {
			Payload(func() {
				Attribute("string", String, func() {
					Example("")
				})
				Meta("openapi:additionalProperties", "false")
			})
			Result(func() {
				Attribute("string", String, func() {
					Example("")
				})
				Meta("openapi:additionalProperties", "false")
			})
			HTTP(func() {
				GET("/")
			})
		})
	})
}

var OpenAPIClosedObjectsDSL = func() {
	var Nested = Type("ClosedObjectsNested", func() {
		Attribute("street", String, func() {
			Example("Main")
		})
		Required("street")
	})

	var UnionAlpha = Type("ClosedObjectsUnionAlpha", func() {
		Attribute("alpha", String, func() {
			Example("a")
		})
		Required("alpha")
	})

	var UnionBeta = Type("ClosedObjectsUnionBeta", func() {
		Attribute("beta", String, func() {
			Example("b")
		})
		Required("beta")
	})

	var _ = API("test", func() {
		Meta("openapi:closed-objects", "true")
		Server("test", func() {
			Host("localhost", func() {
				URI("https://loom.design")
			})
		})
	})

	Service("closedObjectsService", func() {
		Method("object", func() {
			Payload(func() {
				Attribute("name", String, func() {
					Example("")
				})
				Attribute("address", Nested)
				Required("name", "address")
			})
			HTTP(func() {
				POST("/object")
			})
		})

		Method("map_object", func() {
			Payload(func() {
				Attribute("labels", MapOf(String, String))
				Required("labels")
			})
			HTTP(func() {
				POST("/map")
			})
		})

		Method("union_object", func() {
			Payload(OneOf(UnionAlpha, UnionBeta))
			HTTP(func() {
				POST("/union")
			})
		})
	})
}

var MealPlannerDSL = func() {
	var accessToken = JWTSecurity("access_token", func() {
		Description("Bearer token used by web and mobile clients.")
	})
	var browserSession = APIKeySecurity("browser_session", func() {
		Description("Browser session cookie used by the meal planner web app.")
	})
	var appSession = SessionAuth("meal_planner_session", func() {
		BearerTransport(accessToken, "auth")
		CookieTransport(browserSession, "browser_session", func() {
			CookieName("__Host-mealplanner_session")
		})
	})

	var Recipe = ResultType("application/vnd.mealplanner.recipe", func() {
		TypeName("Recipe")
		Description("A saved recipe with planning metadata.")
		Attributes(func() {
			Attribute("id", String, func() {
				Example("recipe_42")
			})
			Attribute("title", String, func() {
				Example("Sheet Pan Gnocchi")
			})
			Attribute("minutes", Int, func() {
				Minimum(5)
				Example(30)
			})
			Attribute("difficulty", String, func() {
				Enum("easy", "medium", "hard")
				Example("easy")
			})
			Attribute("tags", ArrayOf(String), func() {
				Example([]string{"weeknight", "vegetarian"})
			})
			Attribute("ingredients", ArrayOf(String), func() {
				Example([]string{"gnocchi", "tomatoes", "spinach"})
			})
			Required("id", "title", "minutes", "difficulty")
		})
		View("default", func() {
			Attribute("id")
			Attribute("title")
			Attribute("minutes")
			Attribute("difficulty")
			Attribute("tags")
			Attribute("ingredients")
		})
		View("summary", func() {
			Attribute("id")
			Attribute("title")
			Attribute("minutes")
			Attribute("difficulty")
		})
	})

	var MealPlanEntry = Type("MealPlanEntry", func() {
		Attribute("day", String, func() {
			Enum("monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday")
			Example("monday")
		})
		Attribute("slot", String, func() {
			Enum("breakfast", "lunch", "dinner")
			Example("dinner")
		})
		Attribute("recipe_id", String, func() {
			Example("recipe_42")
		})
		Required("day", "slot", "recipe_id")
	})

	var MealPlan = Type("MealPlan", func() {
		Description("A meal plan covering one planning week.")
		Attribute("week_start", String, func() {
			Format(FormatDate)
			Example("2026-03-16")
		})
		Attribute("meals", ArrayOf(MealPlanEntry))
		Required("week_start")
	})

	var MealPlanPatch = Type("MealPlanPatch", func() {
		Description("A partial meal plan update.")
		Attribute("meals", ArrayOf(MealPlanEntry))
	})

	var PantryImportReceipt = Type("PantryImportReceipt", func() {
		Attribute("import_id", String, func() {
			Example("import_7")
		})
		Attribute("accepted_items", Int, func() {
			Example(12)
		})
		Required("import_id", "accepted_items")
	})

	var SingleRecipeSelection = Type("SingleRecipeSelection", func() {
		Attribute("recipe_id", String, func() {
			Example("recipe_42")
		})
		Required("recipe_id")
	})

	var RecipePairSelection = Type("RecipePairSelection", func() {
		Attribute("primary_recipe_id", String, func() {
			Example("recipe_42")
		})
		Attribute("backup_recipe_id", String, func() {
			Example("recipe_84")
		})
		Required("primary_recipe_id", "backup_recipe_id")
	})

	var RecipeSelection = Type("RecipeSelection", func() {
		OneOf("selection", func() {
			Meta("oneof:type:field", "mode")
			Meta("oneof:value:field", "value")
			Attribute("single", SingleRecipeSelection)
			Attribute("pair", RecipePairSelection)
		})
	})

	API("mealplanner", func() {
		Meta("openapi:closed-objects", "true")
		Title("Meal Planner API")
		Description("Plan weekly meals, manage recipes, and coordinate pantry imports.")
		Version("2026-03-19")
		TermsOfService("https://mealplanner.example.test/terms")
		Contact(func() {
			Name("Meal Planner Support")
			Email("support@mealplanner.example.test")
			URL("https://mealplanner.example.test/support")
		})
		License(func() {
			Name("Apache-2.0")
			URL("https://www.apache.org/licenses/LICENSE-2.0.html")
		})
		Server("mealplanner", func() {
			Host("production", func() {
				URI("https://api.mealplanner.example.test")
			})
		})
	})

	Service("mealPlanner", func() {
		Description("Plan meals, browse recipes, and import pantry inventory.")
		SessionSecurity(appSession)

		Method("health", func() {
			NoSecurity()
			Error("rate_limited")
			Result(func() {
				Attribute("status", String, func() {
					Enum("ok")
					Example("ok")
				})
				Attribute("checked_at", String, func() {
					Format(FormatDateTime)
					Example("2026-03-19T12:00:00Z")
				})
				Required("status", "checked_at")
			})
			HTTP(func() {
				GET("/healthz")
				Response(StatusOK)
				Response("rate_limited", StatusTooManyRequests)
			})
		})

		Method("listRecipes", func() {
			Error("bad_request")
			Result(CollectionOf(Recipe), func() {
				View("summary")
			})
			Payload(func() {
				Attribute("tag", String, func() {
					Example("vegetarian")
				})
				Attribute("cursor", String, func() {
					Example("cursor_2")
				})
				Attribute("limit", Int, func() {
					Minimum(1)
					Maximum(100)
					Example(20)
				})
			})
			HTTP(func() {
				GET("/recipes")
				Param("tag")
				Param("cursor")
				Param("limit")
				AuthErrorResponses()
				Response(StatusOK)
				Response("bad_request", StatusBadRequest)
			})
		})

		Method("showRecipe", func() {
			Error("not_found")
			Result(Recipe)
			Payload(func() {
				Attribute("recipe_id", String, func() {
					Pattern("^recipe_[0-9]+$")
					Example("recipe_42")
				})
				Required("recipe_id")
			})
			HTTP(func() {
				GET("/recipes/{recipe_id}")
				Param("recipe_id")
				AuthErrorResponses()
				Response(StatusOK)
				Response("not_found", StatusNotFound)
			})
		})

		Method("upsertMealPlan", func() {
			Error("bad_request")
			Result(MealPlan)
			Payload(func() {
				Attribute("week_start", String, func() {
					Format(FormatDate)
					Example("2026-03-16")
				})
				Attribute("body", MealPlanPatch)
				Required("week_start")
			})
			HTTP(func() {
				PATCH("/meal-plans/{week_start}")
				Param("week_start")
				Body("body")
				OptionalRequestBody()
				AuthErrorResponses()
				Response(StatusOK)
				Response("bad_request", StatusBadRequest)
			})
		})

		Method("shareMealPlan", func() {
			Error("bad_request")
			Result(Empty)
			Payload(func() {
				Attribute("week_start", String, func() {
					Format(FormatDate)
					Example("2026-03-16")
				})
				Attribute("email", String, func() {
					Format(FormatEmail)
					Example("friend@example.test")
				})
				Attribute("note", String, func() {
					MaxLength(280)
					Example("Dinner plan for next week.")
				})
				Attribute("include_shopping_list", Boolean, func() {
					Example(true)
				})
				Required("week_start", "email")
			})
			HTTP(func() {
				POST("/meal-plans/{week_start}/share")
				Param("week_start")
				FormRequest()
				AuthErrorResponses()
				Response(StatusNoContent)
				Response("bad_request", StatusBadRequest)
			})
		})

		Method("previewSelection", func() {
			Error("bad_request")
			Result(Empty)
			Payload(func() {
				Attribute("selection", RecipeSelection, func() {
					Example(map[string]any{
						"mode": "single",
						"value": map[string]any{
							"recipe_id": "recipe_42",
						},
					})
				})
				Required("selection")
			})
			HTTP(func() {
				POST("/plans/preview")
				AuthErrorResponses()
				Response(StatusNoContent)
				Response("bad_request", StatusBadRequest)
			})
		})

		Method("previewSelectionSuppressed", func() {
			Error("bad_request")
			Result(Empty)
			Payload(func() {
				Attribute("selection", RecipeSelection, func() {
					Meta("openapi:example", "false")
				})
				Required("selection")
			})
			HTTP(func() {
				POST("/plans/preview-suppressed")
				AuthErrorResponses()
				Response(StatusNoContent)
				Response("bad_request", StatusBadRequest)
			})
		})

		Method("importPantry", func() {
			Error("bad_request")
			Result(PantryImportReceipt)
			Payload(func() {
				Attribute("pantry_id", String, func() {
					Example("pantry_12")
				})
				Attribute("file", Bytes)
				Attribute("filename", String)
				Attribute("content_type", String)
				Attribute("label", String, func() {
					Pattern("^scan-[a-z]+$")
					Example("scan-weekly")
				})
				Required("pantry_id", "file", "label")
			})
			HTTP(func() {
				POST("/pantries/{pantry_id}/imports")
				Param("pantry_id")
				MultipartRequest()
				AuthErrorResponses()
				Response(StatusAccepted)
				Response("bad_request", StatusBadRequest)
			})
		})

		Method("watchPlannerEvents", func() {
			NoSecurity()
			Error("rate_limited")
			StreamingResult(func() {
				Attribute("event", String, func() {
					Example("plan.updated")
				})
				Attribute("data", func() {
					Attribute("meal_plan_id", String, func() {
						Example("plan_12")
					})
					Required("meal_plan_id")
				})
				Required("event", "data")
			})
			HTTP(func() {
				GET("/events")
				ServerSentEvents()
				Response("rate_limited", StatusTooManyRequests)
			})
		})
	})
}

var CollabStreamsDSL = func() {
	var ThreadSummary = Type("ThreadSummary", func() {
		Attribute("thread_id", String, func() {
			Example("thr_42")
			Pattern("^thr_[0-9]+$")
		})
		Attribute("title", String, func() {
			Example("Release freeze follow-up")
		})
		Attribute("participants", ArrayOf(String), func() {
			Example([]string{"alice", "bob"})
		})
		Required("thread_id", "title")
	})
	var _ = API("collab-streams", func() {
		Title("Collaboration Streams API")
		Description("Collaborative thread operations exposed through HTTP and SSE.")
		Meta("openapi:closed-objects", "true")
		License(func() {
			Name("Apache-2.0")
			URL("https://www.apache.org/licenses/LICENSE-2.0.html")
		})
		Server("collab-streams", func() {
			Host("api", func() {
				URI("https://api.collab-streams.example.test")
			})
		})
	})
	Service("collabStreams", func() {
		Description("Thread collaboration entry points.")

		Method("getThread", func() {
			NoSecurity()
			Payload(func() {
				Attribute("thread_id", String, func() {
					Example("thr_42")
					Pattern("^thr_[0-9]+$")
				})
				Required("thread_id")
			})
			Result(ThreadSummary)
			Error("not_found")
			HTTP(func() {
				GET("/threads/{thread_id}")
				Param("thread_id")
				Response(StatusOK)
				Response("not_found", StatusNotFound)
			})
		})

		Method("watchThread", func() {
			NoSecurity()
			Payload(func() {
				Attribute("thread_id", String, func() {
					Example("thr_42")
					Pattern("^thr_[0-9]+$")
				})
				Attribute("last_event_id", String, func() {
					Example("evt_7")
				})
				Required("thread_id")
			})
			StreamingResult(func() {
				Attribute("id", String, func() {
					Example("evt_8")
				})
				Attribute("event", String, func() {
					Example("thread.message_posted")
				})
				Attribute("data", func() {
					Attribute("author", String, func() {
						Example("alice")
					})
					Attribute("preview", String, func() {
						Example("Shipping the OpenAPI cleanup next.")
					})
					Required("author", "preview")
				})
				Required("id", "event", "data")
			})
			Error("busy")
			HTTP(func() {
				GET("/threads/{thread_id}/events")
				Param("thread_id")
				Param("last_event_id")
				ServerSentEvents(func() {
					SSERequestID("last_event_id")
					SSEEventID("id")
				})
				Response("busy", StatusTooManyRequests)
			})
		})
	})
}

var OpsSocketDSL = func() {
	var _ = API("ops-socket", func() {
		Title("Ops Socket API")
		Description("WebSocket control surface for operator consoles.")
		Meta("openapi:closed-objects", "true")
		License(func() {
			Name("Apache-2.0")
			URL("https://www.apache.org/licenses/LICENSE-2.0.html")
		})
		Server("ops-socket", func() {
			Host("api", func() {
				URI("https://api.ops-socket.example.test")
			})
		})
	})
	Service("opsSocket", func() {
		Description("Bidirectional operator control channel.")

		Method("streamCommands", func() {
			NoSecurity()
			Payload(func() {
				Attribute("channel", String, func() {
					Example("deployments")
				})
				Required("channel")
			})
			StreamingPayload(func() {
				Attribute("op", String, func() {
					Example("pause")
				})
				Attribute("target", String, func() {
					Example("worker-eu-1")
				})
				Required("op", "target")
			})
			StreamingResult(func() {
				Attribute("event", String, func() {
					Example("command.accepted")
				})
				Attribute("ok", Boolean, func() {
					Example(true)
				})
				Required("event", "ok")
			})
			HTTP(func() {
				GET("/ws/ops/{channel}")
				Param("channel")
			})
		})
	})
}

var ActivityFeedDSL = func() {
	var CommentAdded = Type("CommentAddedActivity", func() {
		Attribute("comment_id", String, func() {
			Example("cmt_17")
		})
		Attribute("thread_id", String, func() {
			Example("thr_42")
		})
		Attribute("author", String, func() {
			Example("alice")
		})
		Required("comment_id", "thread_id", "author")
	})
	var TaskClosed = Type("TaskClosedActivity", func() {
		Attribute("task_id", String, func() {
			Example("TASK-9")
		})
		Attribute("closed_by", String, func() {
			Example("bob")
		})
		Required("task_id", "closed_by")
	})
	var ActivityEnvelope = Type("ActivityEnvelope", func() {
		OneOf("entry", func() {
			Attribute("comment_added", CommentAdded, func() {
				Meta("oneof:type:tag", "comment_added")
			})
			Attribute("task_closed", TaskClosed, func() {
				Meta("oneof:type:tag", "task_closed")
			})
		})
		Meta("oneof:type:field", "kind")
		Meta("oneof:value:field", "payload")
	})
	var ValidationError = Type("ActivityValidationError", func() {
		Attribute("message", String, func() {
			Example("cursor must be a base64 token")
		})
		Attribute("field", String, func() {
			Example("cursor")
		})
		Required("message", "field")
	})
	var _ = API("activity-feed", func() {
		Title("Activity Feed API")
		Description("Union-heavy audit and activity retrieval endpoints.")
		Meta("openapi:closed-objects", "true")
		License(func() {
			Name("Apache-2.0")
			URL("https://www.apache.org/licenses/LICENSE-2.0.html")
		})
		Server("activity-feed", func() {
			Host("api", func() {
				URI("https://api.activity-feed.example.test")
			})
		})
	})
	Service("activityFeed", func() {
		Description("Read activity entries across project workstreams.")

		Method("listActivities", func() {
			NoSecurity()
			Payload(func() {
				Attribute("project_id", String, func() {
					Example("proj_9")
				})
				Attribute("cursor", String, func() {
					Example("cursor_2")
				})
				Required("project_id")
			})
			Result(ArrayOf(ActivityEnvelope))
			Error("bad_request", ValidationError)
			HTTP(func() {
				GET("/projects/{project_id}/activities")
				Param("project_id")
				Param("cursor")
				Response(StatusOK)
				Response("bad_request", StatusBadRequest)
			})
		})

		Method("showActivity", func() {
			NoSecurity()
			Payload(func() {
				Attribute("project_id", String, func() {
					Example("proj_9")
				})
				Attribute("activity_id", String, func() {
					Example("act_4")
				})
				Required("project_id", "activity_id")
			})
			Result(ActivityEnvelope)
			Error("bad_request", ValidationError)
			HTTP(func() {
				GET("/projects/{project_id}/activities/{activity_id}")
				Param("project_id")
				Param("activity_id")
				Response(StatusOK)
				Response("bad_request", StatusBadRequest)
			})
		})
	})
}

var StreamingPartialExamplesDSL = func() {
	var RealtimeSSEEvent = Type("RealtimeSSEEvent", func() {
		Attribute("event", String, func() {
			Example("abc123")
		})
		Attribute("data", func() {
			Attribute("message", String)
			Required("message")
		})
		Required("event", "data")
	})
	var RealtimeEnvelope = Type("RealtimeEnvelope", func() {
		Attribute("ts", Int64, func() {
			Example(1)
		})
		Attribute("event", String)
		Required("ts", "event")
	})
	var _ = API("streaming-partial-examples", func() {
		Title("Streaming Partial Examples API")
		Description("Exercises suppression of invalid synthetic examples for streaming responses.")
		License(func() {
			Name("Apache-2.0")
			URL("https://www.apache.org/licenses/LICENSE-2.0.html")
		})
		Server("streaming-partial-examples", func() {
			Host("api", func() {
				URI("https://api.streaming-partial-examples.example.test")
			})
		})
	})
	Service("streamingPartialExamples", func() {
		Method("events", func() {
			NoSecurity()
			StreamingResult(RealtimeSSEEvent)
			HTTP(func() {
				GET("/events")
				ServerSentEvents()
			})
		})
		Method("projectSocket", func() {
			NoSecurity()
			Payload(func() {
				Attribute("projectID", String, func() {
					Example("proj_1")
				})
				Required("projectID")
			})
			StreamingResult(RealtimeEnvelope)
			HTTP(func() {
				GET("/ws/projects/{projectID}")
				Param("projectID")
			})
		})
	})
}

var AsyncSessionSecurityDSL = func() {
	var browserSession = APIKeySecurity("browser_session_cookie", func() {
		Description("Browser session cookie used by first-party async clients.")
	})
	var appSession = SessionAuth("async_session", func() {
		CookieTransport(browserSession, "browser_session", func() {
			CookieName("__Host-ak_session")
		})
	})
	var RealtimeEnvelope = Type("AsyncSessionRealtimeEnvelope", func() {
		Attribute("event", String, func() {
			Example("project.updated")
		})
		Attribute("project_id", String, func() {
			Example("proj_42")
		})
		Required("event", "project_id")
	})
	var RealtimeSSEEvent = Type("AsyncSessionRealtimeSSEEvent", func() {
		Attribute("id", String, func() {
			Example("evt_9")
		})
		Attribute("event", String, func() {
			Example("project.updated")
		})
		Attribute("project_id", String, func() {
			Example("proj_42")
		})
		Required("id", "event", "project_id")
	})
	var _ = API("async-session-security", func() {
		Title("Async Session Security API")
		Description("Exercises async streaming contracts secured by cookie-backed session auth.")
		Meta("openapi:closed-objects", "true")
		Server("async-session-security", func() {
			Host("api", func() {
				URI("https://api.async-session-security.example.test")
			})
		})
	})
	Service("asyncSessionSecurity", func() {
		Description("Async routes protected by the browser session cookie.")
		SessionSecurity(appSession)

		Method("projectSocket", func() {
			Payload(func() {
				Attribute("project_id", String, func() {
					Example("proj_42")
				})
				Required("project_id")
			})
			StreamingPayload(String)
			StreamingResult(RealtimeEnvelope)
			HTTP(func() {
				GET("/ws/projects/{project_id}")
				Param("project_id")
				Response(StatusOK)
			})
		})

		Method("events", func() {
			Payload(func() {
				Attribute("project_id", String, func() {
					Example("proj_42")
				})
				Attribute("last_event_id", String, func() {
					Example("evt_8")
				})
				Required("project_id")
			})
			StreamingResult(RealtimeSSEEvent)
			HTTP(func() {
				GET("/events/{project_id}")
				Param("project_id")
				Param("last_event_id")
				ServerSentEvents(func() {
					SSERequestID("last_event_id")
					SSEEventID("id")
				})
				Response(StatusOK)
			})
		})
	})
}
