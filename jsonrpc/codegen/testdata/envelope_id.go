package testdata

import "github.com/CaliLuke/loom/dsl"

// JSONRPCEnvelopeID exercises envelope binding with native and authored field storage.
func JSONRPCEnvelopeID() {
	dsl.API("probe", func() {
		dsl.JSONRPC(func() {})
	})
	alias := dsl.Type("Correlation", dsl.String, func() {
		dsl.Pattern("^[a-z]+$")
	})
	dsl.Service("calc", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
		})
		envelopeIDCarrierMethods(alias)
		dsl.Method("fmt", func() {
			dsl.Payload(func() {
				dsl.ID("id", dsl.String, func() {
					dsl.Format(dsl.FormatUUID)
				})
				dsl.Attribute("v", dsl.String)
				dsl.Required("id", "v")
			})
			dsl.Result(dsl.String)
			dsl.JSONRPC(func() {})
		})
		dsl.Method("plain", func() {
			dsl.Payload(func() {
				dsl.ID("id", dsl.String)
				dsl.Attribute("v", dsl.String)
			})
			dsl.Result(dsl.String)
			dsl.JSONRPC(func() {})
		})
		dsl.Method("pat", func() {
			dsl.Payload(func() {
				dsl.ID("id", dsl.String, func() {
					dsl.Pattern("^[a-z]+$")
				})
				dsl.Attribute("v", dsl.String)
				dsl.Required("v")
			})
			dsl.Result(dsl.String)
			dsl.JSONRPC(func() {})
		})
	})
	envelopeIDStreamServices()
}

func envelopeIDCarrierMethods(alias any) {
	for _, shape := range []string{"nullable", "required-nullable", "nullable-default", "defaulted", "renamed", "alias", "only", "selected", "custom"} {
		dsl.Method(shape, func() {
			dsl.Payload(func() {
				var typ any = dsl.String
				if shape == "alias" {
					typ = alias
				}
				name := "id"
				if shape == "renamed" {
					name = "id:wire"
				}
				dsl.ID(name, typ, func() {
					switch shape {
					case "nullable", "required-nullable":
						dsl.Nullable()
					case "nullable-default":
						dsl.Nullable()
						dsl.Default("fallback")
					case "custom":
						dsl.Meta("struct:field:type", "time.Time", "time")
					case "defaulted":
						dsl.Default("fallback")
					case "renamed":
						dsl.Meta("struct:field:name", "Correlation")
					}
				})
				if shape == "required-nullable" {
					dsl.Required("id")
				}
				if shape != "only" {
					dsl.Attribute("v", dsl.String)
				}
			})
			dsl.Result(dsl.String)
			dsl.JSONRPC(func() {
				if shape == "selected" {
					dsl.Body("v")
				}
			})
		})
	}
}

func envelopeIDStreamServices() {
	for _, transport := range []string{"sse", "ws"} {
		dsl.Service(transport, func() {
			dsl.JSONRPC(func() {
				if transport == "ws" {
					dsl.GET("/ws")
				} else {
					dsl.POST("/sse")
				}
			})
			dsl.Method("watch", func() {
				payload := func() {
					dsl.ID("id", dsl.String, func() {
						dsl.Pattern("^[a-z0-9]+$")
					})
					dsl.Attribute("v", dsl.String)
					dsl.Required("id", "v")
				}
				if transport == "ws" {
					dsl.StreamingPayload(payload)
					dsl.Result(dsl.String)
				} else {
					dsl.Payload(payload)
					dsl.StreamingResult(dsl.String)
				}
				dsl.JSONRPC(func() {
					if transport == "sse" {
						dsl.ServerSentEvents()
					}
				})
			})
		})
	}
}
