package testdata

import . "github.com/CaliLuke/loom/dsl"

// NamedObjectExamplesDSL keeps authored examples across closed named objects,
// aliases and collection fields without losing their declared values.
func NamedObjectExamplesDSL() {
	address := Type("ExampleAddress", func() {
		Attribute("street", String)
		Required("street")
		Meta("openapi:additionalProperties", "false")
	})
	alias := Type("ExampleAddressAlias", address)
	holder := Type("ExampleAddresses", func() {
		Attribute("direct", address)
		Attribute("alias", alias)
		Attribute("list", ArrayOf(address))
		Attribute("by_name", MapOf(String, alias))
		Example(map[string]any{
			"direct":  map[string]any{"street": "Direct Street"},
			"alias":   map[string]any{"street": "Alias Street"},
			"list":    []any{map[string]any{"street": "List Street"}},
			"by_name": map[string]any{"home": map[string]any{"street": "Map Street"}},
		})
	})
	Service("addresses", func() {
		Method("echo", func() {
			NoAccessCheck("Public example regression specimen")
			Payload(holder)
			Result(holder)
			HTTP(func() {
				POST("/addresses")
			})
		})
	})
}
