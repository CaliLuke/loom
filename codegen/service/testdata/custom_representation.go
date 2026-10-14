package testdata

import . "github.com/CaliLuke/loom/dsl"

// CustomRepresentationViewsDSL exercises custom String semantics without a
// transport. module supplies the generated fixture's authored scalar package.
func CustomRepresentationViewsDSL(module string) func() {
	return func() {
		custom := func(name string) {
			Meta("struct:field:type", "scalar."+name, module+"/scalar")
		}
		uuid := func() {
			Meta("struct:field:type", "uuid.UUID", "github.com/google/uuid")
		}
		record := ResultType("application/vnd.custom-record", customRepresentationRecord(custom, uuid))
		defaults := ResultType("application/vnd.custom-defaults", customRepresentationDefaults(custom, uuid))
		node := Type("Node", func() {
			Attribute("next", "Node")
			Attribute("token", String, func() {
				custom("Token")
				Default("reject")
			})
		})
		branch := ResultType("application/vnd.custom-branch", func() {
			TypeName("Branch")
			Attribute("head", node)
			Attribute("children", ArrayOf(node))
			Attribute("lookup", MapOf(String, node))
			Attribute("tag", String)
			View("default", func() {
				Attribute("head")
				Attribute("children")
				Attribute("lookup")
				Attribute("tag")
			})
			View("tiny", func() {
				Attribute("tag")
			})
		})
		Service("customviews", func() {
			Method("record", func() {
				Result(record)
			})
			Method("defaults", func() {
				Result(defaults)
			})
			Method("branch", func() {
				Result(branch)
			})
		})
	}
}

func customRepresentationRecord(custom func(string), uuid func()) func() {
	return func() {
		TypeName("Record")
		Attribute("id", String, func() {
			uuid()
			MinLength(36)
			MaxLength(36)
			Format(FormatUUID)
		})
		Attribute("token", String, func() {
			custom("Token")
			MinLength(6)
			MaxLength(6)
			Pattern("^ok-")
			Enum("ok-one", "ok-two")
		})
		Attribute("label", String, func() {
			custom("Label")
			MinLength(2)
		})
		Attribute("email", String, func() {
			custom("Token")
			Format(FormatEmail)
		})
		Attribute("opaque", String, func() {
			custom("Opaque")
		})
		Required("id")
		View("default", func() {
			Attribute("id")
			Attribute("token")
			Attribute("label")
			Attribute("email")
			Attribute("opaque")
		})
		View("tiny", func() {
			Attribute("opaque")
		})
	}
}

func customRepresentationDefaults(custom func(string), uuid func()) func() {
	return func() {
		TypeName("Defaults")
		Attribute("id", String, func() {
			uuid()
			Default("550e8400-e29b-41d4-a716-446655440000")
		})
		Attribute("token", String, func() {
			custom("Token")
			Default("ok-default")
		})
		Attribute("label", String, func() {
			custom("Label")
			Default("native-label")
		})
		Attribute("bad", String, func() {
			custom("Token")
			Default("reject")
		})
		Attribute("bad_uuid", String, func() {
			uuid()
			Default("not-a-uuid")
		})
		Attribute("native", String, func() {
			Default("native-default")
		})
		View("default", func() {
			Attribute("id")
			Attribute("token")
			Attribute("label")
		})
		View("bad", func() {
			Attribute("bad")
		})
		View("baduuid", func() {
			Attribute("bad_uuid")
		})
		View("tiny", func() {
			Attribute("native")
		})
	}
}
