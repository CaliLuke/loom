package testdata

import . "github.com/CaliLuke/loom/dsl"

// ObjectDefaultsDSL exercises one object-default contract across transports.
func ObjectDefaultsDSL(transport string) func() {
	return func() {
		API("objectdefaults", func() {})
		leaf := Type("Leaf", func() {
			Meta("struct:pkg:path", "types/leaf")
			Field(1, "x", Int, func() {
				Minimum(0)
				Meta("struct:field:name", "Count")
			})
			Field(2, "label", String, func() {
				Default("nested")
			})
			Required("x")
		})
		container := Type("Container", func() {
			Field(1, "child", leaf)
			Field(2, "items", ArrayOf(leaf))
		})
		Service("defaults", func() {
			if transport == "jsonrpc" {
				JSONRPC(func() {
					POST("/rpc")
				})
			}
			Method("send", func() {
				Payload(func() {
					if transport == "jsonrpc" {
						ID("id", String)
					}
					Field(1, "value", leaf, func() {
						Default(map[string]any{"x": 1})
					})
					Field(2, "items", ArrayOf(leaf), func() {
						Default([]any{map[string]any{"x": 2}})
					})
					Field(3, "indexed", MapOf(String, leaf), func() {
						Default(map[string]any{"key": map[string]any{"x": 3}})
					})
					if transport != "grpc" {
						Field(4, "maybe", leaf, func() {
							Nullable()
							Default(map[string]any{"x": 4})
						})
					}
					Field(5, "nested", container, func() {
						Default(map[string]any{"child": map[string]any{"x": 0}, "items": []any{map[string]any{"x": 5}}})
					})
				})
				switch transport {
				case "jsonrpc":
					JSONRPC(func() {})
				case "grpc":
					GRPC(func() {})
				default:
					HTTP(func() {
						POST("/send")
					})
				}
			})
		})
	}
}
