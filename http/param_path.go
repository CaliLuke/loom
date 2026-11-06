package http

import (
	"fmt"
	"net/url"
	"strings"
)

// EncodePathParam serializes one complete path segment and protects dot segments.
// Errors identify the binding without disclosing its value.
func EncodePathParam(name string, codec ParamCodec, value ParamValue) (string, error) {
	if codec.Style > ParamLabel || !paramKindValid(value.Kind) {
		return "", fmt.Errorf("parameter %q: invalid path codec", name)
	}
	if !value.Present {
		return "", paramError(name, "path value is absent")
	}
	escape := func(text string) string {
		return strings.ReplaceAll(url.PathEscape(text), "=", "%3D")
	}
	if codec.Style == ParamSimple && value.Kind != ParamPairs {
		escape = EscapePathSegment
	}
	if codec.Style == ParamLabel && codec.Explode && value.Kind != ParamScalar {
		parts, err := paramParts(name, value, false, paramIdentity)
		if err != nil {
			return "", err
		}
		for _, part := range parts {
			if strings.Contains(part, ".") {
				return "", paramError(name, "data dots cannot be represented in exploded labels")
			}
		}
	}
	parts, err := paramParts(name, value, codec.Explode, escape)
	if err != nil {
		return "", err
	}
	raw, err := encodePathParts(name, codec, value, parts, escape)
	if err != nil {
		return "", err
	}
	if raw == "" {
		return "", paramError(name, "empty path value has no segment")
	}
	switch raw {
	case ".":
		return "%2E", nil
	case "..":
		return "%2E%2E", nil
	default:
		return raw, nil
	}
}

// DecodePathParam reads a raw path capture, splitting before percent decoding.
// Complete protected dot segments are restored before processing the style.
func DecodePathParam(name, raw string, codec ParamCodec, kind ParamKind) (ParamValue, error) {
	if codec.Style > ParamLabel || !paramKindValid(kind) {
		return ParamValue{}, fmt.Errorf("parameter %q: invalid path codec", name)
	}
	if raw == "" {
		return ParamValue{}, paramError(name, "empty path segment")
	}
	switch strings.ToUpper(raw) {
	case "%2E":
		raw = "."
	case "%2E%2E":
		raw = ".."
	}
	if codec.Style == ParamMatrix {
		return decodeMatrixParam(name, raw, codec.Explode, kind)
	}
	separator := ","
	if codec.Style == ParamLabel {
		if !strings.HasPrefix(raw, ".") {
			return ParamValue{}, paramError(name, "label prefix is absent")
		}
		raw = raw[1:]
		if codec.Explode {
			separator = "."
		}
	}
	parts := []string{raw}
	if kind != ParamScalar {
		parts = strings.Split(raw, separator)
		if kind == ParamPairs && raw == "" {
			parts = nil
		}
	}
	value, err := decodeParamParts(name, parts, kind, codec.Explode, url.PathUnescape)
	if err != nil {
		return ParamValue{}, err
	}
	if codec.Style == ParamLabel && codec.Explode && kind != ParamScalar {
		tokens, err := paramParts(name, value, false, paramIdentity)
		if err != nil {
			return ParamValue{}, err
		}
		for _, token := range tokens {
			if strings.Contains(token, ".") {
				return ParamValue{}, paramError(name, "data dots cannot be represented in exploded labels")
			}
		}
	}
	return value, nil
}

func decodeMatrixParam(name, raw string, explode bool, kind ParamKind) (ParamValue, error) {
	if !strings.HasPrefix(raw, ";") {
		return ParamValue{}, paramError(name, "matrix prefix is absent")
	}
	entries := strings.Split(raw[1:], ";")
	if kind == ParamPairs && explode {
		return decodeParamParts(name, entries, kind, true, url.PathUnescape)
	}
	parts := make([]string, 0, len(entries))
	for i, entry := range entries {
		key, value, equals := strings.Cut(entry, "=")
		decodedKey, err := url.PathUnescape(key)
		if err != nil || decodedKey != name {
			return ParamValue{}, paramError(name, "invalid matrix binding name")
		}
		if len(entries) > 1 && (kind != ParamList || !explode || !equals) {
			return ParamValue{}, paramError(name, "repeated matrix binding")
		}
		switch {
		case kind == ParamScalar:
			parts = append(parts[:0], value)
		case !equals:
			if i != 0 {
				return ParamValue{}, paramError(name, "invalid empty matrix marker")
			}
		case kind == ParamList && explode:
			parts = append(parts, value)
		case kind == ParamPairs && value == "":
			parts = nil
		default:
			parts = strings.Split(value, ",")
		}
	}
	return decodeParamParts(name, parts, kind, false, url.PathUnescape)
}

func encodePathParts(name string, codec ParamCodec, value ParamValue, parts []string, escape func(string) string) (string, error) {
	var raw string
	switch codec.Style {
	case ParamSimple:
		raw = strings.Join(parts, ",")
	case ParamLabel:
		if len(parts) == 0 {
			return "", paramError(name, "empty path collection has no segment")
		}
		separator := ","
		if codec.Explode {
			separator = "."
		}
		raw = "." + strings.Join(parts, separator)
	case ParamMatrix:
		prefix := ";" + escape(name)
		switch {
		case value.Kind == ParamPairs && codec.Explode:
			if len(parts) == 0 {
				return "", paramError(name, "empty exploded matrix object has no segment")
			}
			raw = ";" + strings.Join(parts, ";")
		case value.Kind == ParamList && codec.Explode && len(parts) > 0:
			for _, part := range parts {
				raw += prefix + "=" + part
			}
		case len(parts) == 0 || value.Kind == ParamScalar && value.Text == "":
			raw = prefix
		default:
			raw = prefix + "=" + strings.Join(parts, ",")
		}
	}
	return raw, nil
}
