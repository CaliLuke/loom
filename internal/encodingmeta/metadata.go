// Package encodingmeta reads authored representation overrides without choosing
// a transport codec or depending on the expression and schema packages.
package encodingmeta

import "strings"

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

// StructTags returns the authored full tag set. A nonempty set replaces the
// framework defaults; JSON's name-only override is not a full tag.
func StructTags(meta map[string][]string) map[string]string {
	tags := make(map[string]string)
	for key, values := range meta {
		if strings.HasPrefix(key, "struct:tag:") && key != "struct:tag:json:name" {
			tags[strings.TrimPrefix(key, "struct:tag:")] = strings.Join(values, ",")
		}
	}
	return tags
}

// FormFieldName resolves the emitted form name and visibility. Full tags own
// the complete tag set, so an absent, empty or '-' form tag omits the field.
func FormFieldName(meta map[string][]string, defaultName string) (string, bool) {
	tags := StructTags(meta)
	if len(tags) == 0 {
		return defaultName, true
	}
	return FormTagName(tags["form"])
}

// FormTagName reads the name from an emitted form tag. Options do not affect
// field ownership; an empty or '-' name is excluded from the form codec.
func FormTagName(tag string) (string, bool) {
	name, _, _ := strings.Cut(tag, ",")
	return name, name != "" && name != "-"
}
