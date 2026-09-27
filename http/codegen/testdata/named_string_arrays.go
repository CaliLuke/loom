package testdata

import . "github.com/CaliLuke/loom/dsl"

// NamedStringArraysDSL covers whole and field array mappings with native,
// named, and transitively named string elements and named array containers.
var NamedStringArraysDSL = func() {
	namedStringArrayMappings("header", "query")
}

// NamedStringArrayPathsDSL covers array element conversions in path requests,
// including a wildcard whose name matches the conversion loop's value local.
var NamedStringArrayPathsDSL = func() {
	namedStringArrayMappings("path")
}

func namedStringArrayMappings(locations ...string) {
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
					for _, location := range locations {
						Method(prefix+location, func() {
							attribute := "values"
							if location == "path" {
								attribute = "val"
							}
							if field {
								Payload(func() {
									Attribute(attribute, array, func() {
										if location == "path" {
											Meta("struct:field:name", "Entries")
										}
									})
									Required(attribute)
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
								case "path":
									path += "/{" + attribute + "}"
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
