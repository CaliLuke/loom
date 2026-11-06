package http

import (
	"fmt"
	"net/url"
	"strings"
)

type (
	// RawPairs retains query or form pairs before value percent decoding.
	RawPairs struct {
		pairs []rawParamPair
	}
	// RawPairsBuilder assembles styled fields alongside ordinary URL values.
	RawPairsBuilder struct {
		pairs []string
	}
	rawParamPair struct {
		name, value string
	}
)

// ParseRawPairs validates raw query/form syntax while retaining value escapes.
func ParseRawPairs(raw string) (RawPairs, error) {
	var result RawPairs
	for pair := range strings.SplitSeq(raw, "&") {
		if pair == "" {
			continue
		}
		if strings.Contains(pair, ";") {
			return RawPairs{}, fmt.Errorf("invalid query/form pair separator")
		}
		key, value, _ := strings.Cut(pair, "=")
		name, err := url.QueryUnescape(key)
		if err != nil {
			return RawPairs{}, fmt.Errorf("invalid query/form name encoding")
		}
		if _, err := url.QueryUnescape(value); err != nil {
			return RawPairs{}, paramError(name, "invalid percent encoding")
		}
		result.pairs = append(result.pairs, rawParamPair{name: name, value: value})
	}
	return result, nil
}

// DecodePairsParam reads the binding's owned keys using its resolved grammar.
// Unknown fixed-object keys and sibling keys are ignored before value parsing.
func DecodePairsParam(pairs RawPairs, name string, codec ParamCodec, kind ParamKind, siblings ParamNamespace) (ParamValue, error) {
	if err := validatePairsCodec(name, codec, kind); err != nil {
		return ParamValue{}, err
	}
	if codec.Style == ParamDeepObject || kind == ParamPairs && codec.Explode {
		return decodeOwnedPairs(pairs, name, codec, siblings)
	}
	var values []string
	for _, pair := range pairs.pairs {
		if pair.name == name {
			values = append(values, pair.value)
		}
	}
	if len(values) == 0 {
		return ParamValue{Kind: kind}, nil
	}
	if kind == ParamScalar {
		return decodeParamParts(name, values[:1], kind, false, url.QueryUnescape)
	}
	if codec.Explode {
		return decodeParamParts(name, values, kind, false, url.QueryUnescape)
	}
	if len(values) != 1 {
		return ParamValue{}, paramError(name, "repeated delimited binding")
	}
	var parts []string
	unescape := url.QueryUnescape
	if values[0] != "" {
		switch codec.Style {
		case ParamForm:
			parts = strings.Split(values[0], ",")
		case ParamSpaceDelimited, ParamPipeDelimited:
			text, err := url.QueryUnescape(values[0])
			if err != nil {
				return ParamValue{}, paramError(name, "invalid percent encoding")
			}
			delimiter := " "
			if codec.Style == ParamPipeDelimited {
				delimiter = "|"
			}
			parts = strings.Split(text, delimiter)
			unescape = paramNoUnescape
		}
	}
	return decodeParamParts(name, parts, kind, false, unescape)
}

// Add validates and appends one binding atomically, rejecting sibling key claims.
func (b *RawPairsBuilder) Add(name string, codec ParamCodec, value ParamValue, siblings ParamNamespace) error {
	if err := validatePairsCodec(name, codec, value.Kind); err != nil {
		return err
	}
	if !value.Present {
		return nil
	}
	var pairs []string
	add := func(key, encoded string) error {
		if paramSiblingOwns(siblings, key) {
			return paramError(name, "encoded key belongs to another binding")
		}
		pairs = append(pairs, url.QueryEscape(key)+"="+encoded)
		return nil
	}
	switch {
	case codec.Style == ParamDeepObject || value.Kind == ParamPairs && codec.Explode:
		seen := make(map[string]bool)
		for _, pair := range value.Pairs {
			if seen[pair.Key] {
				return paramError(name, "duplicate associative key")
			}
			seen[pair.Key] = true
			key := pair.Key
			if codec.Style == ParamDeepObject {
				if strings.ContainsAny(key, "[]") {
					return paramError(name, "deepObject key contains brackets")
				}
				key = name + "[" + key + "]"
			}
			if err := add(key, url.QueryEscape(pair.Value)); err != nil {
				return err
			}
		}
	case value.Kind == ParamList && codec.Explode:
		for _, item := range value.Items {
			if err := add(name, url.QueryEscape(item)); err != nil {
				return err
			}
		}
	default:
		if value.Kind == ParamList && len(value.Items) == 1 && value.Items[0] == "" {
			return paramError(name, "singleton empty element is indistinguishable from an empty list")
		}
		encoded, err := encodeFlatPairs(name, codec, value)
		if err != nil {
			return err
		}
		if err := add(name, encoded); err != nil {
			return err
		}
	}
	b.pairs = append(b.pairs, pairs...)
	return nil
}

// Encode appends styled pairs after the deterministic encoding of ordinary fields.
func (b *RawPairsBuilder) Encode(values url.Values) string {
	ordinary, styled := values.Encode(), strings.Join(b.pairs, "&")
	if ordinary == "" {
		return styled
	}
	if styled == "" {
		return ordinary
	}
	return ordinary + "&" + styled
}

func validatePairsCodec(name string, codec ParamCodec, kind ParamKind) error {
	if !paramKindValid(kind) {
		return fmt.Errorf("parameter %q: invalid value kind", name)
	}
	switch codec.Style {
	case ParamForm:
		return nil
	case ParamDeepObject:
		if kind == ParamPairs && !strings.ContainsAny(name, "[]") {
			return nil
		}
	case ParamSpaceDelimited, ParamPipeDelimited:
		if kind != ParamScalar && !codec.Explode {
			return nil
		}
	}
	return fmt.Errorf("parameter %q: invalid query/form codec", name)
}

func decodeOwnedPairs(pairs RawPairs, name string, codec ParamCodec, siblings ParamNamespace) (ParamValue, error) {
	value := ParamValue{Kind: ParamPairs}
	seen := make(map[string]bool)
	for _, pair := range pairs.pairs {
		if !paramOwns(siblings, pair.name) {
			continue
		}
		key := pair.name
		if codec.Style == ParamDeepObject {
			prefix := name + "["
			if !strings.HasPrefix(key, prefix) || !strings.HasSuffix(key, "]") {
				continue
			}
			key = key[len(prefix) : len(key)-1]
			if strings.ContainsAny(key, "[]") {
				return ParamValue{}, paramError(name, "deepObject key contains brackets")
			}
		}
		if seen[key] {
			return ParamValue{}, paramError(name, "duplicate associative key")
		}
		text, err := url.QueryUnescape(pair.value)
		if err != nil {
			return ParamValue{}, paramError(name, "invalid percent encoding")
		}
		seen[key] = true
		value.Present = true
		value.Pairs = append(value.Pairs, ParamPair{Key: key, Value: text})
	}
	return value, nil
}

func encodeFlatPairs(name string, codec ParamCodec, value ParamValue) (string, error) {
	delimiter := ","
	if codec.Style == ParamSpaceDelimited || codec.Style == ParamPipeDelimited {
		dataDelimiter := " "
		delimiter = "%20"
		if codec.Style == ParamPipeDelimited {
			dataDelimiter, delimiter = "|", "%7C"
		}
		parts, err := paramParts(name, value, false, paramIdentity)
		if err != nil {
			return "", err
		}
		for _, part := range parts {
			if strings.Contains(part, dataDelimiter) {
				return "", paramError(name, "data contains the collection delimiter")
			}
		}
	}
	parts, err := paramParts(name, value, false, url.QueryEscape)
	if err != nil {
		return "", err
	}
	return strings.Join(parts, delimiter), nil
}
