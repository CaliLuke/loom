package openapiv3

import (
	"strconv"
	"strings"
)

func applyMediaTypeMetadata(mediaType *MediaType, meta map[string][]string) {
	if mediaType == nil {
		return
	}
	if values, ok := meta["openapi:itemSchema"]; ok && metaValuesEnabled(values) {
		mediaType.UseItemSchema = true
	}
	for key, values := range meta {
		if len(values) == 0 {
			continue
		}
		value := values[len(values)-1]
		switch {
		case strings.HasPrefix(key, "openapi:encoding:"):
			path := strings.Split(strings.TrimPrefix(key, "openapi:encoding:"), ":")
			if len(path) < 2 {
				continue
			}
			if mediaType.Encoding == nil {
				mediaType.Encoding = make(map[string]*Encoding)
			}
			encoding := mediaType.Encoding[path[0]]
			if encoding == nil {
				encoding = new(Encoding)
				mediaType.Encoding[path[0]] = encoding
			}
			applyEncodingMetadata(encoding, path[1:], value)
		case strings.HasPrefix(key, "openapi:prefixEncoding:"):
			path := strings.Split(strings.TrimPrefix(key, "openapi:prefixEncoding:"), ":")
			if len(path) < 2 {
				continue
			}
			index, err := strconv.Atoi(path[0])
			if err != nil || index < 0 {
				continue
			}
			for len(mediaType.PrefixEncoding) <= index {
				mediaType.PrefixEncoding = append(mediaType.PrefixEncoding, new(Encoding))
			}
			applyEncodingMetadata(mediaType.PrefixEncoding[index], path[1:], value)
		case strings.HasPrefix(key, "openapi:itemEncoding:"):
			if mediaType.ItemEncoding == nil {
				mediaType.ItemEncoding = new(Encoding)
			}
			path := strings.Split(strings.TrimPrefix(key, "openapi:itemEncoding:"), ":")
			applyEncodingMetadata(mediaType.ItemEncoding, path, value)
		}
	}
}

func applyEncodingMetadata(encoding *Encoding, path []string, value string) {
	if encoding == nil || len(path) == 0 {
		return
	}
	if path[0] == "encoding" && len(path) >= 3 {
		if encoding.Encoding == nil {
			encoding.Encoding = make(map[string]*Encoding)
		}
		nested := encoding.Encoding[path[1]]
		if nested == nil {
			nested = new(Encoding)
			encoding.Encoding[path[1]] = nested
		}
		applyEncodingMetadata(nested, path[2:], value)
		return
	}
	if path[0] == "prefixEncoding" && len(path) >= 3 {
		index, err := strconv.Atoi(path[1])
		if err != nil || index < 0 {
			return
		}
		for len(encoding.PrefixEncoding) <= index {
			encoding.PrefixEncoding = append(encoding.PrefixEncoding, new(Encoding))
		}
		applyEncodingMetadata(encoding.PrefixEncoding[index], path[2:], value)
		return
	}
	if path[0] == "itemEncoding" && len(path) >= 2 {
		if encoding.ItemEncoding == nil {
			encoding.ItemEncoding = new(Encoding)
		}
		applyEncodingMetadata(encoding.ItemEncoding, path[1:], value)
		return
	}
	switch path[0] {
	case "contentType":
		encoding.ContentType = value
	case "style":
		encoding.Style = value
	case "explode":
		enabled := value != "false"
		encoding.Explode = &enabled
	case "allowReserved":
		encoding.AllowReserved = value != "false"
	}
}

func metaValuesEnabled(values []string) bool {
	return len(values) == 0 || values[len(values)-1] != "false"
}
