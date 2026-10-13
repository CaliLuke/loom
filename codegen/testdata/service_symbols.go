package testdata

import (
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// ServiceReservedSymbolsDSL uses fixed generated package names as domain types.
// Their wire names remain valid while generated Go declarations must be distinct.
func ServiceReservedSymbolsDSL() {
	serviceReservedSymbolsDSL(false)
}

// ServiceExternalSymbolsDSL keeps fixed-name domain types in an unrelated package.
func ServiceExternalSymbolsDSL() {
	serviceReservedSymbolsDSL(true)
}

func serviceReservedSymbolsDSL(external bool) {
	dsl.API("symbols", func() {
	})
	types := make([]expr.UserType, 0, 9)
	for _, name := range []string{"Client", "Endpoints", "Service", "APIName", "APIVersion", "ServiceName", "MethodNames", "NewClient", "NewEndpoints", "Client2"} {
		types = append(types, dsl.Type(name, func() {
			if external {
				dsl.Meta("struct:pkg:path", "domain")
			}
			dsl.Field(1, "value", dsl.String)
		}))
	}
	dsl.Service("symbols", func() {
		dsl.Method("exchange", func() {
			dsl.Payload(func() {
				for i, typ := range types {
					dsl.Field(i+1, typ.Name(), typ)
				}
			})
			dsl.Result(types[0])
			dsl.HTTP(func() {
				dsl.POST("/symbols")
			})
			dsl.GRPC(func() {})
		})
	})
}

// ServiceConditionalSymbolsDSL exercises conditional framework declarations and
// ordinary domain names in the same service-package namespace.
func ServiceConditionalSymbolsDSL(mode string) {
	dsl.API("symbols", func() {
		if mode == "sse" || mode == "websocket" {
			dsl.JSONRPC(func() {
			})
		}
	})
	types := make(map[string]expr.UserType)
	names := []string{"Authorizer", "Stream", "Event", "ServerInterceptors", "ClientInterceptors"}
	for _, name := range names {
		types[name] = dsl.Type(name, func() {
			dsl.Field(1, "value", dsl.String)
		})
	}
	var key *expr.SchemeExpr
	if mode == "secure" {
		key = dsl.APIKeySecurity("key")
	}
	audit := dsl.Interceptor("Audit", func() {
	})
	dsl.Service("symbols", func() {
		if mode == "sse" || mode == "websocket" {
			dsl.JSONRPC(func() {
				if mode == "sse" {
					dsl.POST("/rpc")
				} else {
					dsl.GET("/rpc")
				}
			})
		}
		dsl.Method("exchange", func() {
			if mode == "secure" {
				dsl.Security(key)
			}
			if mode == "interceptors" {
				dsl.ServerInterceptor(audit)
				dsl.ClientInterceptor(audit)
			}
			dsl.Payload(func() {
				for i, name := range names {
					dsl.Field(i+1, name, types[name])
				}
				if mode == "secure" {
					dsl.APIKey("key", "credential", dsl.String)
					dsl.Required("credential")
				}
			})
			if mode == "sse" || mode == "websocket" {
				dsl.StreamingResult(types["Event"])
				dsl.JSONRPC(func() {
					if mode == "sse" {
						dsl.ServerSentEvents()
					}
				})
			} else {
				dsl.Result(types["Event"])
				dsl.HTTP(func() {
					dsl.POST("/symbols")
				})
			}
		})
	})
}

// ServiceFixedSpellingMethodsDSL keeps legal receiver method names independent
// of declarations in the surrounding package.
func ServiceFixedSpellingMethodsDSL() {
	dsl.Service("symbols", func() {
		for _, name := range []string{"Client", "Endpoints", "APIName", "APIVersion", "ServiceName", "NewClient", "NewEndpoints"} {
			dsl.Method(name, func() {
				dsl.HTTP(func() {
					dsl.GET("/" + name)
				})
			})
		}
	})
}

// ServiceMethodTypeCouplingDSL places nominal types alongside similarly named
// service methods and the package helpers derived from those methods.
func ServiceMethodTypeCouplingDSL() {
	names := []string{"Foo", "FooServerStream", "FooClientStream", "FooEndpointInput", "NewFooEndpoint"}
	types := make([]expr.UserType, len(names))
	for i, name := range names {
		types[i] = dsl.Type(name, func() {
			dsl.Field(1, "value", dsl.String)
		})
	}
	dsl.Service("symbols", func() {
		dsl.Method("seed", func() {
			dsl.Payload(func() {
				for i, name := range names {
					dsl.Field(i+1, name, types[i])
				}
			})
			dsl.HTTP(func() {
				dsl.POST("/seed")
			})
			dsl.GRPC(func() {})
		})
		dsl.Method("foo", func() {
			dsl.StreamingResult(types[0])
			dsl.HTTP(func() {
				dsl.GET("/foo")
			})
			dsl.GRPC(func() {})
		})
	})
}

// ServiceMemberFieldsDSL checks actual receiver-member collisions independently
// of the surrounding package's type declarations.
func ServiceMemberFieldsDSL() {
	dsl.Service("symbols", func() {
		for _, name := range []string{"foo", "fooEndpoint", "Use", "HandleStream"} {
			dsl.Method(name, func() {
				dsl.HTTP(func() {
					dsl.GET("/" + name)
				})
			})
		}
	})
}

// ServiceCarrierSymbolsDSL exercises emitted HTTP carriers and method-derived
// interceptor wrappers alongside domain types with the same package spelling.
func ServiceCarrierSymbolsDSL() {
	names := []string{"UploadRequestData", "DownloadResponseData", "ServeFileResponseData", "ServeResponseData", "WrapFooEndpoint", "WrapFooClientEndpoint", "Foo"}
	types := make([]expr.UserType, len(names))
	for i, name := range names {
		types[i] = dsl.Type(name, func() {
			dsl.Field(1, "value", dsl.String)
		})
	}
	audit := dsl.Interceptor("Audit", func() {
		dsl.ReadStreamingResult(func() {
			dsl.Attribute("value")
		})
	})
	dsl.Service("symbols", func() {
		dsl.Method("seed", func() {
			dsl.Payload(func() {
				for i, name := range names {
					dsl.Field(i+1, name, types[i])
				}
			})
			dsl.HTTP(func() {
				dsl.POST("/seed")
			})
		})
		dsl.Method("upload", func() {
			dsl.HTTP(func() {
				dsl.POST("/upload")
				dsl.SkipRequestBodyEncodeDecode()
				dsl.OpenAPIRequestBody(dsl.Bytes, "application/octet-stream", true)
			})
		})
		dsl.Method("download", func() {
			dsl.HTTP(func() {
				dsl.GET("/download")
				dsl.SkipResponseBodyEncodeDecode()
			})
		})
		dsl.Method("serve", func() {
			dsl.HTTP(func() {
				dsl.GET("/serve")
				dsl.FileResponse()
			})
		})
		for _, name := range []string{"foo", "fooClient"} {
			dsl.Method(name, func() {
				dsl.ServerInterceptor(audit)
				dsl.ClientInterceptor(audit)
				dsl.StreamingResult(types[len(types)-1])
				dsl.HTTP(func() {
					dsl.GET("/" + name)
				})
			})
		}
	})
}

// ServiceSSEEventSymbolsDSL makes a domain result and a method event declaration
// compete in the package scope without changing their wire names.
func ServiceSSEEventSymbolsDSL() {
	dsl.API("symbols", func() {
		dsl.JSONRPC(func() {})
	})
	event := dsl.Type("WatchEvent", func() {
		dsl.Field(1, "value", dsl.String)
	})
	dsl.Service("symbols", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
		})
		dsl.Method("watch", func() {
			dsl.StreamingResult(event)
			dsl.JSONRPC(func() {
				dsl.ServerSentEvents()
			})
		})
	})
}
