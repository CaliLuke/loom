package testdata

import . "github.com/CaliLuke/loom/dsl"

// NamedNullableValidationDSL distinguishes declaration-owned nullable carriers
// from field presence at required and optional body roots and nested occurrences.
func NamedNullableValidationDSL(transport, shape string) func() {
	return func() {
		API("namednullablevalidation", func() {
		})
		value := Type("Value", func() {
			Attribute("value", String, func() {
				Pattern("^[a-z]+$")
			})
			Required("value")
		})
		var node any = value
		switch shape {
		case "array":
			node = ArrayOf(value)
		case "map":
			node = MapOf(String, value)
		}
		named := Type("Node", node, func() {
			Nullable()
		})
		Service("trees", func() {
			if transport == "jsonrpc" {
				JSONRPC(func() {
					POST("/rpc")
				})
			}
			for _, method := range []string{"nested_required", "nested_optional", "root_required", "root_optional"} {
				Method(method, func() {
					Payload(func() {
						Attribute("tree", named)
						if method == "nested_required" || method == "root_required" {
							Required("tree")
						}
					})
					if transport == "jsonrpc" {
						JSONRPC(func() {
							if method == "root_required" || method == "root_optional" {
								Body("tree")
							}
						})
					} else {
						HTTP(func() {
							POST("/" + method)
							if method == "root_required" || method == "root_optional" {
								Body("tree")
							}
							Response(StatusNoContent)
						})
					}
				})
			}
		})
	}
}
