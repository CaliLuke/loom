package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// RequiredOccurrenceDSL checks that local requirements refine a declaration
// without contaminating other aliases or changing the declaration's storage.
func RequiredOccurrenceDSL() {
	base := Type("Base", func() {
		Attribute("id", String)
		Attribute("left", String)
		Attribute("right", String)
		Required("id")
	})
	left := Type("Left", base, func() {
		Required("left")
	})
	right := Type("Right", base, func() {
		Required("right")
	})
	occurrences := Type("Occurrences", func() {
		Attribute("direct", base, func() {
			Required("left")
		})
		Attribute("list", ArrayOf(base, func() {
			Required("right")
		}))
		Attribute("index", MapOf(String, base, func() {
			Elem(func() {
				Required("left")
			})
		}))
		Required("direct", "list", "index")
	})
	deferred := Type("Deferred", func() {
		Attribute("name", String)
		Attribute("child", "Deferred", func() {
			Required("name")
		})
		Attribute("tail", String)
	})
	Service("occurrences", func() {
		for _, item := range []struct {
			name string
			typ  expr.UserType
		}{
			{"base", base}, {"left", left}, {"right", right},
			{"occurrences", occurrences}, {"deferred", deferred},
		} {
			Method(item.name, func() {
				Payload(item.typ)
				Result(item.typ)
				HTTP(func() {
					POST("/" + item.name)
					Response(StatusOK)
				})
			})
		}
	})
}
