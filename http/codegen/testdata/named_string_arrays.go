package testdata

import . "github.com/CaliLuke/loom/dsl"

// NamedStringArraysDSL covers whole and field array mappings with native,
// named, and transitively named string elements and named array containers.
var NamedStringArraysDSL = func() {
	label := Type("Label", String)
	chain := Type("LabelChain", label)
	lists := map[string]any{
		"plain": Type("PlainList", ArrayOf(String)),
		"label": Type("LabelList", ArrayOf(label)),
		"chain": Type("ChainList", ArrayOf(chain)),
	}
	Service("namedarrays", func() {
		for _, element := range []struct {
			name   string
			typeOf any
		}{{"plain", String}, {"label", label}, {"chain", chain}} {
			for _, named := range []bool{false, true} {
				var array any = ArrayOf(element.typeOf)
				name := element.name
				if named {
					name += "list"
					array = lists[element.name]
				}
				for _, field := range []bool{false, true} {
					if named && !field {
						continue
					}
					prefix := name
					if field {
						prefix += "field"
					}
					for _, location := range []string{"header", "query"} {
						Method(prefix+location, func() {
							if field {
								Payload(func() {
									Attribute("values", array)
									Required("values")
								})
							} else {
								Payload(array)
							}
							HTTP(func() {
								path := "/" + prefix + location
								switch location {
								case "header":
									Header("values")
								case "query":
									Param("values")
								}
								GET(path)
							})
						})
					}
				}
			}
		}
	})
}
