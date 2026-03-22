package testdata

import . "github.com/CaliLuke/loom/dsl"

// UnionWithFormatValidationDSL defines a OneOf with format validation to test Issue #3747
var UnionWithFormatValidationDSL = func() {
	_ = Type("OneOfWithFormat", func() {
		OneOf("response", func() {
			Attribute("message", String, "Textual greeting")
			Attribute("timestamp", String, func() {
				Format(FormatDateTime)
			})
		})
	})
}
