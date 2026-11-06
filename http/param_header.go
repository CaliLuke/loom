package http

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/internal/headerlist"
)

// EncodeHeaderParam frames an admitted value as one simple header field.
func EncodeHeaderParam(name string, codec ParamCodec, value ParamValue) (string, error) {
	if codec.Style != ParamSimple || !paramKindValid(value.Kind) {
		return "", fmt.Errorf("parameter %q: invalid header codec", name)
	}
	if !value.Present {
		return "", nil
	}
	if value.Kind == ParamScalar {
		if strings.ContainsAny(value.Text, "\r\n") {
			return "", paramError(name, "invalid header scalar")
		}
		return value.Text, nil
	}
	parts, err := paramParts(name, value, codec.Explode, paramIdentity)
	if err != nil {
		return "", err
	}
	for _, part := range parts {
		if err := headerlist.ValidateItem(part); err != nil {
			return "", paramError(name, "value cannot be represented in a header list")
		}
		if strings.ContainsAny(part, "\r\n") {
			return "", paramError(name, "invalid header item")
		}
	}
	if value.Kind == ParamPairs && codec.Explode {
		for _, pair := range value.Pairs {
			if pair.Key == "" || strings.Contains(pair.Key, "=") {
				return "", paramError(name, "invalid exploded header key")
			}
		}
	}
	return strings.Join(parts, ", "), nil
}

// DecodeHeaderParam combines physical lines using HTTP simple-list framing.
func DecodeHeaderParam(name string, lines []string, present bool, codec ParamCodec, kind ParamKind) (ParamValue, error) {
	if codec.Style != ParamSimple || !paramKindValid(kind) {
		return ParamValue{}, fmt.Errorf("parameter %q: invalid header codec", name)
	}
	if !present {
		return ParamValue{Kind: kind}, nil
	}
	if kind == ParamScalar {
		text := ""
		if len(lines) > 0 {
			text = lines[0]
		}
		return ParamValue{Kind: kind, Present: true, Text: text}, nil
	}
	return decodeParamParts(name, headerlist.Decode(lines, true), kind, codec.Explode, paramNoUnescape)
}
