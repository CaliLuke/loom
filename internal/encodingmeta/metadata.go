// Package encodingmeta reads authored representation overrides without choosing
// a transport codec or depending on the expression and schema packages.
package encodingmeta

// Replacement returns the authored Go type replacement. The emitter decides
// whether that replacement applies to the particular declaration or occurrence.
func Replacement(meta map[string][]string) string {
	if values := meta["struct:field:type"]; len(values) > 0 {
		return values[0]
	}
	return ""
}

// SchemaOverride reports an explicit schema encoding or format override. Empty
// authored values remain overrides, matching the metadata Last convention.
// Public type and component naming metadata does not select an encoding.
func SchemaOverride(meta map[string][]string) bool {
	for _, key := range []string{"openapi:contentEncoding", "openapi:contentMediaType", "openapi:format"} {
		if len(meta[key]) > 0 {
			return true
		}
	}
	return false
}
