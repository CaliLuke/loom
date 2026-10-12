package testdata

import . "github.com/CaliLuke/loom/dsl"

// RecursiveValidationDSL exercises finite recursive JSON trees through presence wrappers.
func RecursiveValidationDSL(transport, shape string, recursiveFirst, nullable bool) func() {
	return func() {
		API("recursivevalidation", func() {
		})
		label := Type("Label", String, func() {
			Pattern("^[a-z]+$")
		})
		node := Type("Node", func() {
			value := func() {
				Attribute("value", label)
			}
			edge := func() {
				switch shape {
				case "array":
					Attribute("next", ArrayOf("Node"))
				case "mutual":
					Attribute("next", "Other")
				case "named indirection":
					Attribute("next", "AliasNode")
				case "map":
					Attribute("next", MapOf(String, "Node"))
				default:
					Attribute("next", "Node", func() {
						if nullable {
							Nullable()
						}
					})
				}
			}
			if recursiveFirst {
				edge()
				value()
			} else {
				value()
				edge()
			}
			Required("value")
		})
		if shape == "mutual" {
			Type("Other", func() {
				Attribute("value", label)
				Attribute("next", node)
				Required("value")
			})
		}
		if shape == "named indirection" {
			Type("AliasNode", node)
		}
		Service("trees", func() {
			if transport == "jsonrpc" {
				JSONRPC(func() {
					POST("/rpc")
				})
			}
			Method("check", func() {
				Payload(node)
				if transport == "jsonrpc" {
					JSONRPC(func() {
					})
				} else {
					HTTP(func() {
						POST("/check")
						Response(StatusNoContent)
					})
				}
			})
		})
	}
}
