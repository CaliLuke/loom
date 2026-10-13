package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// ProblemPropertiesDSL extends the default error fixture with authored body and
// header projections of the shared error-property contract.
func ProblemPropertiesDSL() {
	DefaultErrorResponseDSL()
	Service("ProblemProperties", func() {
		for _, name := range []string{"pruned", "timeout", "header", "custom"} {
			Method(name, func() {
				if name == "custom" {
					Error("oops", func() {
						Attribute("reason", String)
						Required("reason")
					})
				} else {
					Error("oops", ErrorResult)
				}
				HTTP(func() {
					GET("/properties/" + name)
					Response("oops", StatusBadRequest, func() {
						switch name {
						case "pruned":
							Body(func() {
								Attribute("detail")
							})
						case "timeout":
							Body("timeout")
						case "header":
							Header("temporary:X-Temporary")
						}
					})
				})
			})
		}
	})
}
