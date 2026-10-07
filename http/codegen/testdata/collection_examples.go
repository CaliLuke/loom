package testdata

import . "github.com/CaliLuke/loom/dsl"

// CollectionExamplesDSL exercises independent collection and element lengths
// alongside recursive map values and bounded numeric aliases in synthesized
// request and response examples.
var CollectionExamplesDSL = func() {
	node := Type("CollectionNode", func() {
		Attribute("labels", ArrayOf(String, func() {
			MaxLength(0)
		}))
		Attribute("children", MapOf(String, "CollectionNode"))
	})
	count := Type("CollectionCount", Int, func() {
		Minimum(1)
	})
	boundedCount := Type("BoundedCollectionCount", count, func() {
		Maximum(4)
	})
	counts := Type("CollectionCounts", ArrayOf(boundedCount))
	API("collections", func() {
		Server("collections", func() {
			Host("localhost", func() {
				URI("https://example.com")
			})
		})
	})
	Service("collections", func() {
		Method("counts", func() {
			NoSecurity()
			Result(counts)
			HTTP(func() {
				GET("/counts")
			})
		})
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
