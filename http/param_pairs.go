package http

import (
	"fmt"
	"net/url"
	"strings"
)

type (
	// RawPairs retains query or form pairs before value percent decoding.
	RawPairs struct {
		pairs    []rawParamPair
		hasInput bool
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
	result := RawPairs{hasInput: raw != ""}
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

// RawPairsFromValue retains one already bounded query/form parameter value.
// name is decoded; rawValue retains percent escapes and collection delimiters.
// Ampersands and semicolons are data, because the caller has already separated
// this value from other parameters. Invalid percent escapes are rejected.
func RawPairsFromValue(name, rawValue string) (RawPairs, error) {
	if _, err := url.QueryUnescape(rawValue); err != nil {
		return RawPairs{}, paramError(name, "invalid percent encoding")
	}
	return RawPairs{pairs: []rawParamPair{{name: name, value: rawValue}}, hasInput: true}, nil
}

// DecodePairsParam reads the binding's owned keys using its resolved grammar.
// Unknown fixed-object keys and sibling keys are ignored before value parsing.
func DecodePairsParam(pairs RawPairs, name string, codec ParamCodec, kind ParamKind, siblings ParamNamespace) (ParamValue, error) {
	return decodePairsParam(pairs, name, codec, kind, siblings, url.QueryUnescape)
}

func decodePairsParam(pairs RawPairs, name string, codec ParamCodec, kind ParamKind, siblings ParamNamespace, unescape func(string) (string, error)) (ParamValue, error) {
	if err := validatePairsCodec(name, codec, kind); err != nil {
		return ParamValue{}, err
	}
	if codec.Style == ParamDeepObject || kind == ParamPairs && codec.Explode {
		return decodeOwnedPairs(pairs, name, codec, siblings, unescape)
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
		return decodeParamParts(name, values[:1], kind, false, unescape)
	}
	if codec.Explode {
		return decodeParamParts(name, values, kind, false, unescape)
	}
	if len(values) != 1 {
		return ParamValue{}, paramError(name, "repeated delimited binding")
	}
	var parts []string
	if values[0] != "" {
		switch codec.Style {
		case ParamForm:
			parts = strings.Split(values[0], ",")
		case ParamSpaceDelimited, ParamPipeDelimited:
			text, err := unescape(values[0])
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

// HasInput reports whether the carrier was parsed from nonempty input. It is
// independent of its decoded key count: "&&" has input but no pairs. Form body
// owners use this fact to distinguish an absent optional body from a supplied
// empty value set. Named parameter presence still comes from its owned keys.
func (p RawPairs) HasInput() bool {
	return p.hasInput
}

// Add validates and appends one binding atomically, rejecting sibling key claims.
func (b *RawPairsBuilder) Add(name string, codec ParamCodec, value ParamValue, siblings ParamNamespace) error {
	pairs, err := encodeParamPairs(name, codec, value, siblings, false)
	if err != nil {
		return err
	}
	for _, pair := range pairs {
		b.pairs = append(b.pairs, url.QueryEscape(pair.name)+"="+pair.value)
	}
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

func decodeOwnedPairs(pairs RawPairs, name string, codec ParamCodec, siblings ParamNamespace, unescape func(string) (string, error)) (ParamValue, error) {
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
		text, err := unescape(pair.value)
		if err != nil {
			return ParamValue{}, paramError(name, "invalid percent encoding")
		}
		seen[key] = true
		value.Present = true
		value.Pairs = append(value.Pairs, ParamPair{Key: key, Value: text})
	}
	return value, nil
}

func encodeFlatPairs(name string, codec ParamCodec, value ParamValue, literal bool) (string, error) {
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
	escape := url.QueryEscape
	if literal {
		escape = paramIdentity
		if codec.Style == ParamSpaceDelimited {
			delimiter = " "
		} else if codec.Style == ParamPipeDelimited {
			delimiter = "|"
		}
	}
	parts, err := paramParts(name, value, false, escape)
	if err != nil {
		return "", err
	}
	if literal && value.Kind != ParamScalar {
		for _, part := range parts {
			if strings.Contains(part, delimiter) {
				return "", paramError(name, "data contains the collection delimiter")
			}
		}
	}
	return strings.Join(parts, delimiter), nil
}

func encodeParamPairs(name string, codec ParamCodec, value ParamValue, siblings ParamNamespace, literal bool) ([]rawParamPair, error) {
	if err := validatePairsCodec(name, codec, value.Kind); err != nil {
		return nil, err
	}
	if !value.Present {
		return nil, nil
	}
	escape := url.QueryEscape
	if literal {
		escape = paramIdentity
	}
	var pairs []rawParamPair
	add := func(key, encoded string) error {
		if paramSiblingOwns(siblings, key) {
			return paramError(name, "encoded key belongs to another binding")
		}
		pairs = append(pairs, rawParamPair{name: key, value: encoded})
		return nil
	}
	switch {
	case codec.Style == ParamDeepObject || value.Kind == ParamPairs && codec.Explode:
		seen := make(map[string]bool)
		for _, pair := range value.Pairs {
			if seen[pair.Key] {
				return nil, paramError(name, "duplicate associative key")
			}
			seen[pair.Key] = true
			key := pair.Key
			if codec.Style == ParamDeepObject {
				if strings.ContainsAny(key, "[]") {
					return nil, paramError(name, "deepObject key contains brackets")
				}
				key = name + "[" + key + "]"
			}
			if err := add(key, escape(pair.Value)); err != nil {
				return nil, err
			}
		}
	case value.Kind == ParamList && codec.Explode:
		for _, item := range value.Items {
			if err := add(name, escape(item)); err != nil {
				return nil, err
			}
		}
	default:
		if value.Kind == ParamList && len(value.Items) == 1 && value.Items[0] == "" {
			return nil, paramError(name, "singleton empty element is indistinguishable from an empty list")
		}
		encoded, err := encodeFlatPairs(name, codec, value, literal)
		if err != nil {
			return nil, err
		}
		if err := add(name, encoded); err != nil {
			return nil, err
		}
	}
	return pairs, nil
}
