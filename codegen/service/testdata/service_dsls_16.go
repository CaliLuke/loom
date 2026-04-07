package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

var UnionDefaultKeysDSL = func() {
	var DefaultUnion = Type("DefaultUnion", func() {
		OneOf("Values", func() {
			Attribute("Int", Int)
			Attribute("String", String)
		})
	})
	Service("DefaultKeysService", func() {
		Method("DefaultUnion", func() {
			Payload(DefaultUnion)
			Result(DefaultUnion)
		})
	})
}
