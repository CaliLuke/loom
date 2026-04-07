package testdata

import . "github.com/CaliLuke/loom/dsl"


var JSONPrefixDSL = func() {
	var _ = API("test", func() {
		Server("test", func() {
			Host("localhost", func() {
				URI("https://loom.design")
			})
		})
		Meta("openapi:json:prefix", "  ")
	})
	var _ = Service("service-name", func() {
		Files("path1", "filename")
		Files("path2", "filename", func() {
			Meta("openapi:tag:user-tag")
		})
	})
}


