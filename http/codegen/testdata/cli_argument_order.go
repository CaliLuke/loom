package testdata

import (
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// CLIArgumentOrderDSL combines independently optional parser dependencies across
// one or more hosted services. The subset case reverses and filters service order.
func CLIArgumentOrderDSL(transport, shape string) func() {
	return func() {
		names := []string{"alpha", "beta"}
		if shape == "single" {
			names = names[:1]
		}
		dsl.API("order", func() {
			if transport == "jsonrpc" {
				dsl.JSONRPC(func() {
				})
			}
			dsl.Server("order", func() {
				if shape == "subset" {
					dsl.Services("beta", "alpha")
				} else {
					dsl.Services(names...)
				}
				dsl.Host("local", func() {
					dsl.URI("http://localhost:8080")
				})
			})
		})
		if shape == "subset" {
			names = append(names, "excluded")
		}
		audit := dsl.Interceptor("Audit", func() {
		})
		for _, name := range names {
			dsl.Service(name, cliArgumentService(transport, shape, name, audit))
		}
	}
}

func cliArgumentService(transport, shape, name string, audit *expr.InterceptorExpr) func() {
	return func() {
		if shape == "file-only" && name == "alpha" {
			dsl.ClientInterceptor(audit)
			dsl.Files("/static/{*filepath}", "public")
			return
		}
		if transport == "jsonrpc" {
			dsl.JSONRPC(func() {
				if shape == "rpc-stream" {
					dsl.GET("/" + name)
				} else {
					dsl.POST("/" + name)
				}
			})
		}
		dsl.Method("up", func() {
			if shape != "plain" && shape != "multipart" && (shape != "split" || name == "alpha") {
				dsl.ClientInterceptor(audit)
			}
			if shape == "rpc-stream" {
				dsl.StreamingPayload(dsl.String)
				dsl.Result(dsl.String)
			} else {
				dsl.Payload(dsl.String)
			}
			mapping := func() {
				if transport == "http" {
					dsl.POST("/" + name)
				}
				if shape != "plain" && shape != "interceptors" && shape != "rpc-stream" && (shape != "split" || name == "beta") {
					dsl.MultipartRequest()
				}
			}
			if transport == "jsonrpc" {
				dsl.JSONRPC(mapping)
			} else {
				dsl.HTTP(mapping)
			}
		})
		if shape == "stream" || shape == "rpc-stream" {
			dsl.Method("watch", func() {
				dsl.StreamingResult(dsl.String)
				mapping := func() {
					if transport == "http" {
						dsl.GET("/" + name + "/watch")
					}
				}
				if transport == "jsonrpc" {
					dsl.JSONRPC(mapping)
				} else {
					dsl.HTTP(mapping)
				}
			})
		}
	}
}
