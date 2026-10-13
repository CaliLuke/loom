package testdata

import . "github.com/CaliLuke/loom/dsl"

// PreparedEnumProvenanceDSL exercises a refined object enum in implicit bodies.
var PreparedEnumProvenanceDSL = func() {
	preparedEnumProvenance(false, false)
}

// PreparedEnumMappedDSL exercises logical field names with mapped JSON spelling.
var PreparedEnumMappedDSL = func() {
	preparedEnumProvenance(true, false)
}

// PreparedEnumExplicitDSL preserves the already working explicit body contract.
var PreparedEnumExplicitDSL = func() {
	preparedEnumProvenance(false, true)
}

func preparedEnumProvenance(mapped, explicit bool) {
	API("objectenum", func() {
	})
	name, wire := "n", "n"
	if mapped {
		name, wire = "n:wire_n", "wire_n"
	}
	base := Type("EnumBase", func() {
		Attribute(name, Int)
		Enum(map[string]any{wire: 1}, map[string]any{wire: 2})
	})
	refined := Type("EnumRefined", func() {
		Extend(base)
		Attribute(name, Int, func() {
			Minimum(2)
		})
	})
	alias := Type("EnumAlias", refined)
	Service("numbers", func() {
		Method("exchange", func() {
			Payload(alias)
			Result(alias)
			HTTP(func() {
				POST("/numbers")
				if explicit {
					Body(alias)
				}
				Response(StatusOK, func() {
					if explicit {
						Body(alias)
					}
				})
			})
		})
	})
}
