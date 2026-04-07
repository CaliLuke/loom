package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

var UnionCustomKeysDSL = func() {
	var CustomUnion = Type("CustomUnion", func() {
		OneOf("Values", func() {
			Meta("oneof:type:field", "kind")
			Meta("oneof:value:field", "data")
			Attribute("Int", Int)
			Attribute("String", String)
		})
	})
	Service("CustomKeysService", func() {
		Method("CustomUnion", func() {
			Payload(CustomUnion)
			Result(CustomUnion)
		})
	})
}

// UnionCustomKeysMultiTypeDSL tests union with custom keys and user-defined types.
