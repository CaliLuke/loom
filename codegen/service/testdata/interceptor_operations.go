package testdata

import "github.com/CaliLuke/loom/dsl"

// InterceptorContractStreamDSL covers distinct server operation and endpoint shapes.
func InterceptorContractStreamDSL(transport string) func() {
	return func() {
		fields := func() {
			dsl.Field(1, "value", dsl.String)
		}
		access := func() {
			dsl.Attribute("value")
		}
		audit := dsl.Interceptor("Audit", func() {
			if transport == "rpc-request-reply" || transport == "rpc-bidi" {
				dsl.ReadStreamingPayload(access)
				dsl.WriteStreamingPayload(access)
			}
			if transport != "rpc-request-reply" {
				dsl.ReadStreamingResult(access)
				dsl.WriteStreamingResult(access)
			}
			if transport == "rpc-request-reply" {
				dsl.ReadResult(access)
				dsl.WriteResult(access)
			}
		})
		observe := dsl.Interceptor("Observe", func() {
		})
		if transport != "grpc" {
			dsl.API("contracts", func() {
				dsl.JSONRPC(func() {
				})
			})
		}
		dsl.Service("contracts", func() {
			if transport != "grpc" {
				dsl.JSONRPC(func() {
					if transport == "rpc-sse" {
						dsl.POST("/rpc")
					} else {
						dsl.GET("/ws")
					}
				})
			}
			dsl.Method("watch", func() {
				dsl.ServerInterceptor(observe, audit)
				dsl.ClientInterceptor(observe, audit)
				if transport == "rpc-request-reply" || transport == "rpc-bidi" {
					dsl.StreamingPayload(fields)
				} else {
					dsl.Payload(func() {
						dsl.Field(1, "id", dsl.String)
					})
				}
				if transport == "rpc-request-reply" {
					dsl.Result(fields)
				} else {
					dsl.StreamingResult(fields)
				}
				if transport == "grpc" {
					dsl.GRPC(func() {
					})
				} else {
					dsl.JSONRPC(func() {
						if transport == "rpc-sse" {
							dsl.ServerSentEvents(func() {
								dsl.SSENotificationMethod("watch")
							})
						}
					})
				}
			})
		})
	}
}
