package testdata

import . "github.com/CaliLuke/loom/dsl"

func ordinaryFormPresenceMethods() func() {
	empty := Type("OrdinaryFormEmpty", func() {
		Meta("openapi:additionalProperties", "false")
	})
	fields := Type("OrdinaryFormFields", func() {
		Attribute("name", String, func() {
			MinLength(2)
		})
	})
	required := Type("OrdinaryFormRequiredFields", func() {
		Attribute("name", String)
		Required("name")
	})
	return func() {
		for _, tc := range []struct {
			name               string
			typ                any
			selected, optional bool
		}{
			{"PlainEmpty", empty, false, false},
			{"PlainFields", fields, false, false},
			{"PlainRequiredFields", required, false, false},
			{"PlainSelectedRequired", fields, true, false},
			{"PlainOptional", fields, false, true},
			{"PlainSelected", fields, true, true},
			{"PlainSelectedEmpty", empty, true, true},
		} {
			Method(tc.name, func() {
				if tc.selected {
					Payload(func() {
						Attribute("value", tc.typ)
						if !tc.optional {
							Required("value")
						}
						Attribute("owner", String)
					})
				} else {
					Payload(tc.typ)
				}
				HTTP(func() {
					POST("/" + tc.name)
					FormRequest()
					if tc.selected {
						Body("value")
						Param("owner")
					}
					if tc.optional {
						OptionalRequestBody()
					}
					Response(StatusNoContent)
				})
			})
		}
	}
}
