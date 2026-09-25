package testdata

import . "github.com/CaliLuke/loom/dsl"

var NoInterceptorsDSL = func() {
	Service("Service", func() {
		Method("Method", func() {
			HTTP(func() {
				GET("/")
			})
		})
	})
}

var ValidInterceptorsDSL = func() {
	Interceptor("api", func() {})
	API("API", func() {
		ServerInterceptor("api")
	})

	Service("Service", func() {
		Interceptor("service", func() {})
		ServerInterceptor("service")

		Method("Method", func() {
			Interceptor("method", func() {})
			ServerInterceptor("method")
		})
	})
}

var DuplicateInterceptorsDSL = func() {
	Interceptor("duplicate", func() {})
	API("API", func() {
		ServerInterceptor("duplicate")
	})

	Service("Service", func() {
		ServerInterceptor("duplicate")
		Method("Method", func() {
			ServerInterceptor("duplicate")
		})
	})
}

var MixedInterceptorsDSL = func() {
	Interceptor("api", func() {})
	Interceptor("api-client", func() {})
	API("API", func() {
		ServerInterceptor("api")
		ClientInterceptor("api-client")
	})

	Service("Service", func() {
		Interceptor("service", func() {})
		Interceptor("service-client", func() {})
		ServerInterceptor("service")
		ClientInterceptor("service-client")

		Method("Method", func() {
			Interceptor("method", func() {})
			Interceptor("method-client", func() {})
			ServerInterceptor("method")
			ClientInterceptor("method-client")
		})
	})
}

var UndefinedInterceptorDSL = func() {
	Service("Service", func() {
		Method("Method", func() {
			ServerInterceptor("undefined")
		})
	})
}

var EmptyInterceptorNameDSL = func() {
	Service("Service", func() {
		Method("Method", func() {
			ServerInterceptor("")
		})
	})
}

// CaseCollidingInterceptorsDSL applies two server interceptors whose names
// differ only by case to one method.
var CaseCollidingInterceptorsDSL = func() {
	Interceptor("audit", func() {})
	Interceptor("Audit", func() {})
	Service("Service", func() {
		Method("Method", func() {
			ServerInterceptor("audit", "Audit")
		})
	})
}

// SeparatorCollidingInterceptorsDSL applies a server and a client interceptor
// whose names differ only by separators to different methods.
var SeparatorCollidingInterceptorsDSL = func() {
	Interceptor("audit_log", func() {})
	Interceptor("audit-log", func() {})
	Service("Service", func() {
		Method("A", func() {
			ServerInterceptor("audit_log")
		})
		Method("B", func() {
			ClientInterceptor("audit-log")
		})
	})
}

// APICollidingInterceptorsDSL applies an API interceptor to every service and
// a method interceptor whose name produces the same Go name.
var APICollidingInterceptorsDSL = func() {
	Interceptor("audit", func() {})
	Interceptor("Audit", func() {})
	API("API", func() {
		ClientInterceptor("audit")
	})
	Service("Service", func() {
		Method("A", func() {
			ServerInterceptor("Audit")
		})
		Method("B", func() {
			ServerInterceptor("Audit")
		})
	})
}

// ThreeCollidingInterceptorsDSL applies three interceptors that produce the
// same Go name.
var ThreeCollidingInterceptorsDSL = func() {
	Interceptor("audit_log", func() {})
	Interceptor("AuditLog", func() {})
	Interceptor("audit-log", func() {})
	Service("Service", func() {
		ServerInterceptor("audit_log", "AuditLog")
		Method("Method", func() {
			ClientInterceptor("audit-log")
		})
	})
}

// SeparateServicesInterceptorsDSL applies interceptors that produce the same
// Go name to different services, whose packages do not share names.
var SeparateServicesInterceptorsDSL = func() {
	Interceptor("audit", func() {})
	Interceptor("Audit", func() {})
	Service("A", func() {
		Method("Method", func() {
			ServerInterceptor("audit")
		})
	})
	Service("B", func() {
		Method("Method", func() {
			ClientInterceptor("Audit")
		})
	})
}

// DistinctGoNamesInterceptorsDSL applies interceptors whose names differ only
// by case but produce different Go names.
var DistinctGoNamesInterceptorsDSL = func() {
	Interceptor("audit", func() {})
	Interceptor("AUDIT", func() {})
	Service("Service", func() {
		Method("Method", func() {
			ServerInterceptor("audit", "AUDIT")
		})
	})
}
