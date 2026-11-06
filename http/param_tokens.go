package http

import (
	"fmt"
	"slices"
	"strings"
)

type paramInputError struct {
	name, reason string
}

func (e *paramInputError) Error() string {
	return fmt.Sprintf("parameter %q: %s", e.name, e.reason)
}

func paramError(name, reason string) error {
	return &paramInputError{name: name, reason: reason}
}

func paramKindValid(kind ParamKind) bool {
	return kind <= ParamPairs
}

func paramParts(name string, value ParamValue, explode bool, escape func(string) string) ([]string, error) {
	switch value.Kind {
	case ParamScalar:
		return []string{escape(value.Text)}, nil
	case ParamList:
		parts := make([]string, len(value.Items))
		for i, item := range value.Items {
			parts[i] = escape(item)
		}
		return parts, nil
	case ParamPairs:
		parts := make([]string, 0, len(value.Pairs)*2)
		seen := make(map[string]bool, len(value.Pairs))
		for _, pair := range value.Pairs {
			if seen[pair.Key] {
				return nil, paramError(name, "duplicate associative key")
			}
			seen[pair.Key] = true
			key, val := escape(pair.Key), escape(pair.Value)
			if explode {
				parts = append(parts, key+"="+val)
			} else {
				parts = append(parts, key, val)
			}
		}
		return parts, nil
	default:
		return nil, fmt.Errorf("parameter %q: invalid value kind", name)
	}
}

func decodeParamParts(name string, parts []string, kind ParamKind, explode bool, unescape func(string) (string, error)) (ParamValue, error) {
	value := ParamValue{Kind: kind, Present: true}
	decode := func(raw string) (string, error) {
		text, err := unescape(raw)
		if err != nil {
			return "", paramError(name, "invalid percent encoding")
		}
		return text, nil
	}
	switch kind {
	case ParamScalar:
		if len(parts) != 1 {
			return ParamValue{}, paramError(name, "invalid scalar framing")
		}
		text, err := decode(parts[0])
		value.Text = text
		return value, err
	case ParamList:
		value.Items = make([]string, 0, len(parts))
		for _, part := range parts {
			text, err := decode(part)
			if err != nil {
				return ParamValue{}, err
			}
			value.Items = append(value.Items, text)
		}
	case ParamPairs:
		value.Pairs = make([]ParamPair, 0)
		if !explode && len(parts)%2 != 0 {
			return ParamValue{}, paramError(name, "associative value requires key/value pairs")
		}
		seen := make(map[string]bool)
		for i := 0; i < len(parts); i++ {
			var rawKey, rawValue string
			if explode {
				var found bool
				rawKey, rawValue, found = strings.Cut(parts[i], "=")
				if !found {
					return ParamValue{}, paramError(name, "associative entry requires equals sign")
				}
			} else {
				rawKey, rawValue = parts[i], parts[i+1]
				i++
			}
			key, err := decode(rawKey)
			if err != nil {
				return ParamValue{}, err
			}
			val, err := decode(rawValue)
			if err != nil {
				return ParamValue{}, err
			}
			if seen[key] {
				return ParamValue{}, paramError(name, "duplicate associative key")
			}
			seen[key] = true
			value.Pairs = append(value.Pairs, ParamPair{Key: key, Value: val})
		}
	default:
		return ParamValue{}, fmt.Errorf("parameter %q: invalid value kind", name)
	}
	return value, nil
}

func paramIdentity(value string) string {
	return value
}

func paramNoUnescape(value string) (string, error) {
	return value, nil
}

func paramSiblingOwns(namespace ParamNamespace, key string) bool {
	if slices.Contains(namespace.Literals, key) {
		return true
	}
	for _, prefix := range namespace.Prefixes {
		if strings.HasPrefix(key, prefix+"[") && strings.HasSuffix(key, "]") {
			return true
		}
	}
	return false
}

func paramOwns(namespace ParamNamespace, key string) bool {
	return !paramSiblingOwns(namespace, key) && (namespace.OwnedLiterals == nil || slices.Contains(namespace.OwnedLiterals, key))
}
