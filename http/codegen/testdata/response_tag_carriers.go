package testdata

import . "github.com/CaliLuke/loom/dsl"

// ResponseTagCarriersDSL selects responses using each supported String field carrier.
var ResponseTagCarriersDSL = func() {
	marker := Type("Marker", String)
	kinds := []string{"ordinary", "required", "alias", "required_alias", "nullable", "named_nullable", "renamed", "mapped", "viewed", "viewed_alias", "bodyless"}
	methods := make([]func(), 0, len(kinds))
	for _, kind := range kinds {
		attributes := func() {
			field := "kind"
			if kind == "mapped" {
				field = "kind:wire_kind"
			}
			var carrier any = String
			if kind == "alias" || kind == "required_alias" || kind == "named_nullable" || kind == "viewed_alias" {
				carrier = marker
			}
			Attribute(field, carrier, func() {
				if kind == "nullable" || kind == "named_nullable" {
					Nullable()
				}
				if kind == "renamed" {
					Meta("struct:field:name", "Disposition")
				}
			})
			Attribute("message", String)
			Required("message")
			if kind == "required" || kind == "required_alias" {
				Required(field)
			}
		}
		var result any
		viewed := kind == "viewed" || kind == "viewed_alias"
		if viewed {
			result = ResultType("application/vnd."+kind, func() {
				TypeName(kind + "Result")
				Attributes(attributes)
				View("default", func() {
					Attribute("kind")
					Attribute("message")
				})
			})
		} else {
			result = Type(kind+"Result", attributes)
		}
		methods = append(methods, func() {
			Method(kind, func() {
				Result(result, func() {
					if viewed {
						View("default")
					}
				})
				HTTP(func() {
					GET("/" + kind)
					for _, status := range []int{StatusOK, StatusAccepted} {
						Response(status, func() {
							if status == StatusAccepted {
								Tag("kind", "accepted")
							}
							if kind == "bodyless" {
								Body(Empty)
							} else {
								Body("message")
							}
						})
					}
				})
			})
		})
	}
	Service("tagged", func() {
		for _, method := range methods {
			method()
		}
	})
}
