package testdata

import . "github.com/CaliLuke/loom/dsl"

// InterceptorPublicNamesDSL reproduces public interceptor declaration collisions.
func InterceptorPublicNamesDSL(mode string) {
	audit := Interceptor("Audit", func() {
		ReadPayload(func() {
			Attribute("id")
		})
		ReadResult(func() {
			Attribute("val")
		})
	})
	var info any
	if mode == "info" {
		info = Type("AuditInfo", func() {
			Attribute("label", String)
		})
	}
	Service("svc", func() {
		ServerInterceptor(audit)
		ClientInterceptor(audit)
		HTTP(func() {
			Path("/")
		})
		name := "audit"
		if mode != "method" {
			name = "ordinary"
		}
		Method(name, func() {
			Payload(func() {
				Attribute("id", String)
				if info != nil {
					Attribute("info", info)
				}
			})
			Result(func() {
				Attribute("val", Int)
			})
			HTTP(func() {
				POST("/audit")
			})
		})
	})
}

// InterceptorPublicRolesDSL exercises public accessor roles in the same package
// as authored types, including a collision between two interceptor declarations.
func InterceptorPublicRolesDSL(streaming, cross bool, side string, names []string) {
	audit := Interceptor("Audit", func() {
		ReadPayload(publicNameField)
		if streaming {
			ReadStreamingPayload(publicNameField)
			ReadStreamingResult(publicNameField)
		} else {
			ReadResult(publicNameField)
		}
	})
	var other any
	if cross {
		other = Interceptor("AuditStreaming", func() {
			ReadPayload(publicNameField)
		})
	}
	types := make([]any, len(names))
	for i, name := range names {
		types[i] = Type(name, String)
	}
	Service("roles", func() {
		Method("ordinary", func() {
			if side != "client" {
				ServerInterceptor(audit)
				if other != nil {
					ServerInterceptor(other)
				}
			}
			if side != "server" {
				ClientInterceptor(audit)
				if other != nil {
					ClientInterceptor(other)
				}
			}
			Payload(func() {
				publicNameValue()
				for i, typ := range types {
					Attribute(names[i], typ)
				}
			})
			if streaming {
				StreamingPayload(publicNameValue)
				StreamingResult(publicNameValue)
			} else {
				Result(publicNameValue)
			}
			HTTP(func() {
				if streaming {
					GET("/ordinary")
					Param("value")
					for _, name := range names {
						Param(name)
					}
				} else {
					POST("/ordinary")
				}
			})
		})
	})
}

func publicNameField() {
	Attribute("value")
}

func publicNameValue() {
	Attribute("value", String)
	Required("value")
}
