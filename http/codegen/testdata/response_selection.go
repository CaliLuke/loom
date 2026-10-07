package testdata

import . "github.com/CaliLuke/loom/dsl"

// ResponseSelectionOrderDSL exercises overlapping tags, default placement, and
// mapped response bodies and headers without changing tagged declaration order.
var ResponseSelectionOrderDSL = func() {
	result := Type("SelectionResult", func() {
		Attribute("first", String)
		Attribute("second", String)
		Attribute("message", String)
		Attribute("trace", String)
		Required("first", "second", "message", "trace")
	})
	Service("selection", func() {
		for _, method := range []struct {
			name     string
			statuses []int
		}{
			{"default_first", []int{StatusOK, StatusCreated, StatusAccepted}},
			{"default_middle", []int{StatusCreated, StatusOK, StatusAccepted}},
			{"default_last", []int{StatusCreated, StatusAccepted, StatusOK}},
		} {
			Method(method.name, func() {
				Result(result)
				HTTP(func() {
					GET("/" + method.name)
					for _, status := range method.statuses {
						Response(status, func() {
							switch status {
							case StatusCreated:
								Tag("first", "yes")
							case StatusAccepted:
								Tag("second", "yes")
							}
							Body("message")
							Header("trace:X-Trace")
						})
					}
				})
			})
		}
	})
}
