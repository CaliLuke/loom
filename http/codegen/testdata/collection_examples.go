package testdata

import . "github.com/CaliLuke/loom/dsl"

// CollectionExamplesDSL exercises independent collection and element lengths
// alongside recursive map values in synthesized request and response examples.
var CollectionExamplesDSL = func() {
	node := Type("CollectionNode", func() {
		Attribute("labels", ArrayOf(String, func() {
			MaxLength(0)
		}))
		Attribute("children", MapOf(String, "CollectionNode"))
	})
	API("collections", func() {
		Server("collections", func() {
			Host("localhost", func() {
				URI("https://example.com")
			})
		})
	})
	Service("collections", func() {
		Method("echo", func() {
			NoSecurity()
			Payload(node)
			Result(node)
			HTTP(func() {
				POST("/collections")
			})
		})
	})
}
