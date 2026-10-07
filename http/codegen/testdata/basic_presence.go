package testdata

import . "github.com/CaliLuke/loom/dsl"

// BasicPresenceDSL covers independent requiredness and presence of the two
// components of a Basic Authorization header, including named strings.
var BasicPresenceDSL = func() {
	credential := Type("Credential", String)
	basic := BasicAuthSecurity("presence_basic")
	Service("presence", func() {
		for _, name := range []string{"both", "user", "pass", "neither"} {
			Method(name, func() {
				Security(basic)
				Payload(func() {
					Username("user", credential)
					Password("pass", credential)
					if name == "both" || name == "user" {
						Required("user")
					}
					if name == "both" || name == "pass" {
						Required("pass")
					}
				})
				HTTP(func() { GET("/" + name) })
			})
		}
	})
}
