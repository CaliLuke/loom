package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

var FilesValidDSL = func() {
	Service("files-dsl", func() {
		Files("path", "filename")
	})
}

var FilesIncompatibleDSL = func() {
	API("files-incompatile", func() {
		Files("path", "filename")
	})
}

var FilesTooManyArgErrorDSL = func() {
	API("files-too-many-arg-error", func() {
		Files("path", "filename", func() {}, func() {})
	})
}
