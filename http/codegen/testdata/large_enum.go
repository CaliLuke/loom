package testdata

import (
	"fmt"

	. "github.com/CaliLuke/loom/dsl"
)

// LargeEnumResponseDSL exercises a shared nested enum in raw response schemas.
func LargeEnumResponseDSL() {
	values := make([]any, 433)
	for index := range values {
		values[index] = fmt.Sprintf("category-%04d", index)
	}
	category := Type("Category", String, func() {
		Enum(values...)
	})
	entry := Type("Entry", func() {
		Attribute("category", category)
		Attribute("name", String)
		Required("category", "name")
	})
	envelope := Type("Envelope", func() {
		Attribute("entries", ArrayOf(entry))
		Attribute("total", Int)
		Required("total")
	})
	Service("catalog", func() {
		for index := range 6 {
			Method(fmt.Sprintf("list%d", index), func() {
				NoSecurity()
				HTTP(func() {
					GET(fmt.Sprintf("/catalog/%d", index))
					SkipResponseBodyEncodeDecode()
					Response(StatusOK, func() {
						ContentType("application/json")
						OpenAPIBody(envelope)
					})
				})
			})
		}
	})
}
