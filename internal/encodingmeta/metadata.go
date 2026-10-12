// Package encodingmeta reads authored representation overrides without choosing
// a transport codec or depending on the expression and schema packages.
package encodingmeta

// GoTypeOverride is the positional custom Go representation tuple.
type GoTypeOverride struct {
	// Name is the emitted type expression.
	Name string
	// Import is the package import path.
	Import string
	// Alias is the explicit package identifier.
	Alias string
	// HasImport distinguishes an omitted import from an authored empty import.
	HasImport bool
}

// ProtoTypeOverride is the positional custom protobuf representation tuple.
type ProtoTypeOverride struct {
	// Name is the emitted protobuf type.
	Name string
	// Import is the protobuf source import.
	Import string
	// GoName is the generated Go type name.
	GoName string
	// GoImport is the Go package import.
	GoImport string
	// Count records the supplied tuple positions, capped at four.
	Count int
}

// Replacement returns the authored Go type replacement. The emitter decides
// whether that replacement applies to the particular declaration or occurrence.
func Replacement(meta map[string][]string) string {
	return GoType(meta).Name
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

// GoType reads the custom Go type tuple used by generation and composition.
func GoType(meta map[string][]string) GoTypeOverride {
	values := meta["struct:field:type"]
	var result GoTypeOverride
	if len(values) > 0 {
		result.Name = values[0]
	}
	if len(values) > 1 {
		result.Import, result.HasImport = values[1], true
	}
	if len(values) > 2 {
		result.Alias = values[2]
	}
	return result
}

// ProtoType reads the custom protobuf tuple without choosing a transport.
func ProtoType(meta map[string][]string) ProtoTypeOverride {
	values := meta["struct:field:proto"]
	result := ProtoTypeOverride{Count: min(len(values), 4)}
	if len(values) > 0 {
		result.Name = values[0]
	}
	if len(values) > 1 {
		result.Import = values[1]
	}
	if len(values) > 2 {
		result.GoName = values[2]
	}
	if len(values) > 3 {
		result.GoImport = values[3]
	}
	return result
}
