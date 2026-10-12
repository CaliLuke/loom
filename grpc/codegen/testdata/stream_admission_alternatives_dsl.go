package testdata

import . "github.com/CaliLuke/loom/dsl"

// StreamAdmissionAlternativesDSL preserves complete security requirements at admission.
func StreamAdmissionAlternativesDSL() {
	primary := APIKeySecurity("primary", func() {
	})
	secondary := APIKeySecurity("secondary", func() {
	})
	jwt := JWTSecurity("jwt", func() {
	})
	credential := Type("Credential", String, func() {
		MinLength(2)
	})
	Service("up", func() {
		for _, name := range []string{"MetaFirst", "MetaLast", "MixedFirst", "MixedLast", "And", "Anonymous", "Named", "Bearer"} {
			Method(name, func() {
				switch name {
				case "MetaLast", "MixedFirst":
					Security(secondary)
					Security(primary)
				case "Bearer":
					Security(jwt)
					Security(primary)
				case "And":
					Security(primary, secondary)
				case "Anonymous":
					Security(primary)
					Security()
				default:
					Security(primary)
					Security(secondary)
				}
				Payload(func() {
					Field(1, "name", String)
					if name == "Named" {
						APIKeyField(2, "primary", "key", credential)
					} else {
						APIKeyField(2, "primary", "key", String)
					}
					APIKeyField(3, "secondary", "second", String)
					if name == "Bearer" {
						TokenField(4, "token", String)
					}
					Required("name")
				})
				StreamingPayload(func() {
					Field(1, "chunk", String)
				})
				Result(String)
				GRPC(func() {
					Metadata(func() {
						Attribute("key:x-key")
						if name == "Bearer" {
							Attribute("token:authorization")
						}
						if name != "MixedFirst" && name != "MixedLast" {
							Attribute("second:y-key")
						}
					})
					Message(func() {
						Attribute("name")
						if name == "MixedFirst" || name == "MixedLast" {
							Attribute("second")
						}
					})
				})
			})
		}
		// Reserve the natural generated authentication accessor name.
		Method("AuthenticateMetaFirst", func() {
			Result(String)
			GRPC(func() {
			})
		})
	})
}
