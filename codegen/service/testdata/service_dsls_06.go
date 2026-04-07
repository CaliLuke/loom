package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

var PkgPathNoDirDSL = func() {
	var NoDir = Type("NoDir", func() {
		Attribute("IntField", Int)
		Meta("struct:pkg:path", "")
	})

	Service("NoDirMethod", func() {
		Method("A", func() {
			Payload(NoDir)
			Result(NoDir)
		})
	})
}
