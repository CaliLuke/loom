package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// FinalResultInterceptorsDSL covers canonical, dynamic-view and fixed-view
// final results together with ordinary and streaming payload access.
func FinalResultInterceptorsDSL() {
	Interceptor("audit", func() {
		ReadPayload(func() {
			Attribute("initial")
		})
		ReadStreamingPayload(func() {
			Attribute("chunk")
		})
		ReadStreamingResult(func() {
			Attribute("data")
		})
		WriteStreamingResult(func() {
			Attribute("data")
		})
	})
	message := ResultType("application/vnd.finalmessage", "Message", func() {
		Attribute("data", String)
		Attribute("detail", String)
		View("default", func() {
			Attribute("data")
			Attribute("detail")
		})
		View("tiny", func() {
			Attribute("data")
		})
	})
	plain := Type("PlainMessage", func() {
		Attribute("data", String)
	})
	Service("final", func() {
		ServerInterceptor("audit")
		ClientInterceptor("audit")
		for _, name := range []string{"plain", "viewed", "fixed"} {
			Method(name, func() {
				Payload(func() {
					Attribute("initial", String)
				})
				StreamingPayload(func() {
					Attribute("chunk", String)
				})
				switch name {
				case "plain":
					Result(plain)
				case "viewed":
					Result(message)
				case "fixed":
					Result(message, func() {
						View("tiny")
					})
				}
				HTTP(func() {
					GET("/" + name)
					Header("initial")
				})
			})
		}
	})
}
