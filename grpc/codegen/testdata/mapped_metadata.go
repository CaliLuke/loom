package testdata

import (
	"fmt"

	. "github.com/CaliLuke/loom/dsl"
)

// MappedMetadataDSL covers attribute-name resolution independently of JSON names,
// metadata names, requiredness, defaults, and explicit message declarations.
func MappedMetadataDSL() {
	i := 0
	for _, suffix := range []string{"", ":json_name"} {
		for _, required := range []bool{false, true} {
			for _, defaults := range []bool{false, true} {
				for _, explicit := range []bool{false, true} {
					MappedMetadataCaseDSL(fmt.Sprintf("mapped%d", i), suffix, required, defaults, explicit)
					i++
				}
			}
		}
	}
}

// MappedMetadataCaseDSL declares one service whose metadata and message fields
// exercise the selected naming and presence combinations.
func MappedMetadataCaseDSL(name, suffix string, required, defaults, explicit bool) {
	value := Type(name+"Value", func() {
		Field(1, "body"+suffix, String)
		for _, name := range []string{"tok", "tail"} {
			Attribute(name+suffix, String, func() {
				if defaults {
					Default("fallback")
				}
			})
		}
		if required {
			Required("body"+suffix, "tok"+suffix, "tail"+suffix)
		}
	})
	Service(name, func() {
		Method("echo", func() {
			Payload(value)
			Result(value)
			GRPC(func() {
				if explicit {
					Message(func() {
						Attribute("body")
					})
				}
				Metadata(func() {
					Attribute("tok:x-token")
					Attribute("tail")
				})
				Response(CodeOK, func() {
					if explicit {
						Message(func() {
							Attribute("body")
						})
					}
					Headers(func() {
						Attribute("tok:x-token")
					})
					Trailers(func() {
						Attribute("tail:x-tail")
					})
				})
			})
		})
	})
}
