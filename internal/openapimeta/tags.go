// Package openapimeta owns the authored metadata key contract for OpenAPI tags.
package openapimeta

import (
	"encoding/base64"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	tagPrefix   = "openapi:tag:"
	tagIDPrefix = "openapi:tag-id:"
)

// TagIdentityKey returns the canonical metadata key that identifies name.
func TagIdentityKey(name string) string {
	if strings.Contains(name, ":") {
		return tagIDPrefix + base64.RawURLEncoding.EncodeToString([]byte(name))
	}
	return tagPrefix + name
}

// TagAnnotationKey returns the canonical metadata key for a tag annotation.
func TagAnnotationKey(name, field string) string {
	return TagIdentityKey(name) + ":" + field
}

// ParseTagKey parses a canonical tag metadata key. A false result means key is
// unrelated to OpenAPI tag metadata; malformed tag keys return an error.
func ParseTagKey(key string) (name, field string, ok bool, err error) {
	var encoded string
	switch {
	case strings.HasPrefix(key, tagIDPrefix):
		encoded = strings.TrimPrefix(key, tagIDPrefix)
		encoded, field, err = splitField(encoded)
		if err != nil {
			return "", "", true, err
		}
		decoded, decodeErr := base64.RawURLEncoding.DecodeString(encoded)
		if decodeErr != nil || base64.RawURLEncoding.EncodeToString(decoded) != encoded {
			return "", "", true, fmt.Errorf("malformed encoded OpenAPI tag name in %q", key)
		}
		if !utf8.Valid(decoded) || !strings.Contains(string(decoded), ":") {
			return "", "", true, fmt.Errorf("encoded OpenAPI tag name in %q must be valid UTF-8 and contain a colon", key)
		}
		return string(decoded), field, true, nil
	case strings.HasPrefix(key, tagPrefix):
		encoded = strings.TrimPrefix(key, tagPrefix)
		encoded, field, err = splitField(encoded)
		if err != nil {
			return "", "", true, err
		}
		if strings.Contains(encoded, ":") {
			return "", "", true, fmt.Errorf("OpenAPI tag name in %q must contain no colon; use the encoded tag-id form", key)
		}
		return encoded, field, true, nil
	default:
		return "", "", false, nil
	}
}

func splitField(value string) (name, field string, err error) {
	name, suffix, hasSuffix := strings.Cut(value, ":")
	if !hasSuffix {
		return name, "", nil
	}
	if suffix == "url:desc" {
		return name, suffix, nil
	}
	if strings.HasPrefix(suffix, "extension:") {
		extension := strings.TrimPrefix(suffix, "extension:")
		if !strings.HasPrefix(extension, "x-") {
			return "", "", fmt.Errorf("invalid OpenAPI tag extension metadata field %q", suffix)
		}
		return name, suffix, nil
	}
	switch suffix {
	case "desc", "summary", "parent", "kind", "url":
		return name, suffix, nil
	default:
		return "", "", fmt.Errorf("unknown OpenAPI tag metadata field %q", suffix)
	}
}

// ValidateTagMetadata rejects malformed keys and values at the generation
// boundary, before tag extraction can index an invalid value list.
func ValidateTagMetadata(metadata map[string][]string) error {
	keys := make([]string, 0, len(metadata))
	for key := range metadata {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		values := metadata[key]
		name, field, ok, err := ParseTagKey(key)
		if !ok {
			continue
		}
		if err != nil {
			return err
		}
		if field == "" {
			if len(values) != 0 {
				return fmt.Errorf("OpenAPI tag identity %q must not have a value", name)
			}
			continue
		}
		if len(values) != 1 {
			return fmt.Errorf("OpenAPI tag metadata %q requires exactly one value, got %d", key, len(values))
		}
	}
	return nil
}
