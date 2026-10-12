package testdata

import . "github.com/CaliLuke/loom/dsl"

// TypedCollectionDefaultsDSL exercises compatible authored numeric collections.
func TypedCollectionDefaultsDSL(transport string) func() {
	return func() {
		API("collectiondefaults", func() {})
		count := Type("Count", Int32)
		counts := Type("Counts", ArrayOf(count))
		located := Type("Located", Int32, func() {
			Meta("struct:pkg:path", "types/counts")
		})
		Service("defaults", func() {
			if transport == "jsonrpc" {
				JSONRPC(func() {
					POST("/rpc")
				})
			}
			Method("send", func() {
				Payload(func() {
					Attribute("named", counts, func() {
						Default([]int{3})
					})
					Attribute("empty", ArrayOf(count), func() {
						Default([]int{})
					})
					Attribute("located", ArrayOf(located), func() {
						Default([]int{4})
					})
					Attribute("floats", ArrayOf(Float32), func() {
						Default([]float64{1.5})
					})
					Attribute("integers", ArrayOf(Int64), func() {
						Default([]int{3})
					})
					Attribute("nested", ArrayOf(ArrayOf(UInt32)), func() {
						Default([][]int{{3}})
					})
				})
				if transport == "jsonrpc" {
					JSONRPC(func() {})
				} else {
					HTTP(func() {
						POST("/send")
					})
				}
			})
		})
	}
}
