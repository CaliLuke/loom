package testdata

import . "github.com/CaliLuke/loom/dsl"

// AnonymousValidationDSL exercises concrete unwrapped objects with native and
// named children, collections, and an optional object-level enum predicate.
func AnonymousValidationDSL(transport string, nullable, enum bool) func() {
	return func() {
		leaf := Type("Leaf", func() {
			Attribute("text", String, func() {
				Pattern("^[a-z]+$")
			})
			Required("text")
		})
		Service("objects", func() {
			if transport == "jsonrpc" {
				JSONRPC(func() {
					POST("/rpc")
				})
			}
			Method("send", func() {
				Payload(func() {
					Attribute("value", func() {
						Attribute("child", leaf)
						Attribute("children", ArrayOf(leaf))
						Attribute("entries", MapOf(String, leaf))
						Attribute("inline", func() {
							Attribute("text", String, func() {
								Pattern("^[a-z]+$")
							})
						})
						if nullable {
							Nullable()
						}
						if enum {
							Enum(map[string]any{"child": map[string]any{"text": "good"}})
						}
					})
				})
				if transport == "jsonrpc" {
					JSONRPC(func() {
					})
				} else {
					HTTP(func() {
						POST("/send")
						Response(StatusNoContent)
					})
				}
			})
		})
	}
}
