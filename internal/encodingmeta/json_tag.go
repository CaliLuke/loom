package encodingmeta

import "strings"

type (
	// JSONStorage identifies the field carrier selected by its declaration owner.
	JSONStorage uint8

	// JSONOmission is the observation performed by a resolved field encoder.
	JSONOmission uint8

	// JSONFieldPolicy supplies physical storage and omission role independently.
	JSONFieldPolicy struct {
		// Storage is the actual emitted carrier, not the semantic DSL kind.
		Storage JSONStorage
		// Optional permits automatic omission for this emission role.
		Optional bool
		// Presence normalizes full tags for existing explicit-presence contracts.
		Presence bool
		// ForceOmission retains protocol-mandated omission, including RPC IDs.
		ForceOmission bool
	}

	// JSONField is the immutable result shared by tags and value projection.
	JSONField struct {
		// Tag is the normalized JSON tag, including an authored empty tag.
		Tag string
		// Present reports whether the tag is emitted.
		Present bool
		// Omission is the resolved observation for this physical carrier.
		Omission JSONOmission
	}

	// FieldLayout records declaration decisions without rendering Go type names.
	FieldLayout struct {
		// Pointer adds a pointer around the declared field type.
		Pointer bool
		// WrapNullable adds an occurrence-level Nullable wrapper.
		WrapNullable bool
		// WrapOptional adds a decoder-owned Optional wrapper.
		WrapOptional bool
		// JSON supplies the exact field carrier and omission role.
		JSON JSONFieldPolicy
	}
)

const (
	// JSONNative is an ordinary nonnilable value; every zero is supplied.
	JSONNative JSONStorage = iota
	// JSONNilable is an ordinary pointer, slice or map with nil absence.
	JSONNilable
	// JSONPresence is a framework carrier with an explicit absence operation.
	JSONPresence
	// JSONCustom is an opaque application codec, including custom IsZero.
	JSONCustom
)

const (
	// JSONRetain emits every supplied field value.
	JSONRetain JSONOmission = iota
	// JSONOmitAbsent omits only the carrier's absent state.
	JSONOmitAbsent
	// JSONOmitEmpty applies the encoded-emptiness contract of omitempty.
	JSONOmitEmpty
	// JSONOmitNativeZero applies authored omitzero to ordinary native storage.
	JSONOmitNativeZero
	// JSONOmitEmptyOrNativeZero applies both authored omission predicates.
	JSONOmitEmptyOrNativeZero
	// JSONExternalCodec delegates observation to an application codec.
	JSONExternalCodec
)

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

// ResolveJSONField resolves authored intent against the actual field carrier.
// Full tags retain their options unless an existing explicit-presence or protocol
// contract normalizes them. Automatic tags omit absence only; native values retain
// their zeros. Opaque custom carriers keep their established tag policy.
func ResolveJSONField(meta map[string][]string, defaultName string, policy JSONFieldPolicy) JSONField {
	values, full := meta["struct:tag:json"]
	tag := strings.Join(values, ",")
	if !full {
		tag = strings.Join(meta["struct:tag:json:name"], ",")
		if tag == "" {
			tag = defaultName
		}
		if tag == "" {
			return JSONField{}
		}
	}
	zero := policy.Storage == JSONNilable || policy.Storage == JSONPresence || policy.Presence
	omit := policy.Optional && (zero || policy.Storage == JSONCustom || policy.ForceOmission)
	if tag != "-" && omit && (!full || policy.Presence || policy.ForceOmission) {
		tag = appendJSONOmitOption(tag, zero && !policy.ForceOmission || policy.Presence)
	}
	field := JSONField{Tag: tag, Present: true}
	if policy.Storage == JSONCustom {
		field.Omission = JSONExternalCodec
		return field
	}
	empty, nativeZero := false, false
	for _, option := range strings.Split(tag, ",")[1:] {
		switch option {
		case "omitempty":
			empty = true
		case "omitzero":
			field.Omission = JSONOmitAbsent
			nativeZero = policy.Storage == JSONNative
		}
	}
	switch {
	case empty && nativeZero:
		field.Omission = JSONOmitEmptyOrNativeZero
	case empty:
		field.Omission = JSONOmitEmpty
	case nativeZero:
		field.Omission = JSONOmitNativeZero
	}
	return field
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
