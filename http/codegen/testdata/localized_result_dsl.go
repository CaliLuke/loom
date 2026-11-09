package testdata

import . "github.com/CaliLuke/loom/dsl"

// LocalizedResultFieldsDSL retains result declarations customized before their
// normal evaluation pass, including container fields and declared views.
func LocalizedResultFieldsDSL() {
	record := ResultType("application/vnd.localized.record", func() {
		TypeName("LocalizedRecord")
		Attributes(func() {
			Attribute("id", String)
			Attribute("name", String)
			Required("id")
		})
		View("default", func() {
			Attribute("id")
			Attribute("name")
		})
		View("tiny", func() {
			Attribute("id")
		})
	})
	holder := Type("LocalizedHolder", func() {
		Attribute("custom", record, func() {
			Meta("probe:key", "value")
		})
		Attribute("plain", record)
		Attribute("items", ArrayOf(record), func() {
			Description("Customized array")
		})
		Attribute("by_key", MapOf(String, record), func() {
			Description("Customized map")
		})
	})
	Service("localized", func() {
		Method("echo", func() {
			NoAccessCheck("Public regression specimen")
			Payload(holder)
			Result(holder)
			HTTP(func() {
				POST("/localized")
			})
		})
	})
}
