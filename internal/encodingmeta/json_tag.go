package encodingmeta

import "strings"

// JSONTagPolicy describes omission selected by the field representation owner.
// Authored full tags override ordinary omission, while explicit carriers and
// protocol-required omission normalize it. Exact exclusion tags remain excluded.
type JSONTagPolicy struct {
	// Optional requests omission for an absent or empty optional field.
	Optional bool
	// Presence selects omitzero rather than omitempty for an explicit carrier.
	Presence bool
	// ForceOmission applies the selected omission even to an authored full tag.
	ForceOmission bool
}

// ExplicitPresence reports the existing Go carrier override policy from shared
// semantic facts and authored representation metadata.
func ExplicitPresence(meta map[string][]string, nullable, unconstrained bool) bool {
	if nullable || unconstrained {
		return true
	}
	values := meta["openapi:nullable"]
	typeName := GoType(meta).Name
	return typeName != "" && (len(values) > 0 && values[len(values)-1] != "false" || strings.HasPrefix(typeName, "loom.Nullable["))
}

// JSONTag resolves authored full/name-only metadata and the existing omission
// policy. defaultName is the field identity; policy is supplied by its owning
// representation or semantic composition boundary.
// The bool reports whether a tag should be emitted, including an explicit empty
// full tag. Callers retain ownership of semantic requiredness and nullability.
func JSONTag(meta map[string][]string, defaultName string, policy JSONTagPolicy) (string, bool) {
	values, full := meta["struct:tag:json"]
	tag := strings.Join(values, ",")
	if !full {
		tag = strings.Join(meta["struct:tag:json:name"], ",")
		if tag == "" {
			tag = defaultName
		}
		if tag == "" {
			return "", false
		}
	}
	if tag != "-" && policy.Optional && (!full || policy.Presence || policy.ForceOmission) {
		tag = appendJSONOmitOption(tag, policy.Presence)
	}
	return tag, true
}

func appendJSONOmitOption(tag string, zeroOmit bool) string {
	option := "omitempty"
	if zeroOmit {
		option = "omitzero"
	}
	parts := strings.Split(tag, ",")
	options := make([]string, 0, len(parts))
	seen := false
	for _, part := range parts[1:] {
		if part == "omitempty" || part == "omitzero" {
			if !seen {
				options = append(options, option)
				seen = true
			}
			continue
		}
		options = append(options, part)
	}
	if !seen {
		options = append(options, option)
	}
	return strings.Join(append([]string{parts[0]}, options...), ",")
}
