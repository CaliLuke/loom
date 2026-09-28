package testdata

import . "github.com/CaliLuke/loom/dsl"

// MessageReservationsDSL collides a numbered endpoint message with a nested
// anonymous union in both method declaration orders.
var MessageReservationsDSL = func() {
	occupied := Type("AOrZRequest", func() {
		Field(1, "name", String)
	})
	a := Type("A", func() {
		Field(1, "text", String)
	})
	z := Type("ZRequest2", func() {
		Field(1, "count", Int)
	})
	for _, name := range []string{"endpointfirst", "anonymousfirst"} {
		Service(name, func() {
			endpoint := func() {
				Method("a_or_z", func() {
					Payload(func() {
						Field(1, "detail", occupied)
					})
					GRPC(func() {})
				})
			}
			anonymous := func() {
				Method("n", func() {
					Payload(OneOf(String, OneOf(a, z)))
					GRPC(func() {})
				})
			}
			methods := []func(){endpoint, anonymous}
			if name == "anonymousfirst" {
				methods[0], methods[1] = methods[1], methods[0]
			}
			for _, method := range methods {
				method()
			}
		})
	}
}
