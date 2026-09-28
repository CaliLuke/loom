package testdata

import . "github.com/CaliLuke/loom/dsl"

// MessageDeclarationsDSL exercises nested collection wrappers whose internal
// names are also design type names, in both field declaration orders.
var MessageDeclarationsDSL = func() {
	array := Type("ArrayOfString_", func() {
		Field(1, "count", Int)
	})
	mapping := Type("MapOfString_Sint32_", func() {
		Field(1, "label", String)
	})
	for _, name := range []string{"wrappersfirst", "designfirst"} {
		Service(name, func() {
			Method("m", func() {
				Payload(func() {
					wrappers := func() {
						Field(1, "matrix", ArrayOf(ArrayOf(String)))
						Field(2, "maps", ArrayOf(MapOf(String, Int32)))
						Field(3, "again", MapOf(String, ArrayOf(String)))
					}
					named := func() {
						Field(4, "array", array)
						Field(5, "mapping", mapping)
					}
					if name == "wrappersfirst" {
						wrappers()
						named()
					} else {
						named()
						wrappers()
					}
				})
				GRPC(func() {})
			})
		})
	}
}
