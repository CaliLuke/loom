// Package httpcodec owns the built-in HTTP runtime's role-specific codec
// selection. Media labels, negotiation and decoder support are distinct inputs.
package httpcodec

import (
	"mime"
	"strings"
)

type (
	// Kind identifies a built-in codec or a selector's explicit failure.
	Kind uint8
	// Selection retains the media spelling used by runtime headers and errors.
	Selection struct {
		// Kind is the selected codec.
		Kind Kind
		// Media is normalized when parsing succeeds, otherwise the original input.
		Media string
	}
)

const (
	// Invalid means an explicit response Content-Type failed MIME parsing.
	Invalid Kind = iota
	// Unsupported means the request decoder does not support this media type.
	Unsupported
	// JSON selects the framework JSON codec.
	JSON
	// XML selects encoding/xml.
	XML
	// GOB selects encoding/gob.
	GOB
	// Text selects the framework text codec.
	Text
)

// RequestContentType selects the server request decoder. Only exact built-in
// media types are accepted; a missing header defaults to JSON.
func RequestContentType(media string) Selection {
	if media == "" {
		media = "application/json"
	}
	media = normalized(media)
	selection := exact(media)
	if selection.Kind == Invalid {
		selection.Kind = Unsupported
	}
	return selection
}

// ResponseContentType selects an explicitly configured server response encoder.
// Invalid MIME returns Invalid; valid unknown types retain the JSON fallback.
func ResponseContentType(media string) Selection {
	parsed, _, err := mime.ParseMediaType(media)
	if err != nil {
		return Selection{Kind: Invalid, Media: media}
	}
	return suffix(parsed)
}

// ResponseAccept selects the server response encoder without a Content-Type
// override. Only exact built-in media are negotiated; all others default JSON.
func ResponseAccept(media string) Selection {
	selected := exact(normalized(media))
	if selected.Kind == Invalid {
		return Selection{Kind: JSON, Media: "application/json"}
	}
	return selected
}

// ResponseDecode selects the client response decoder. Parse failures retain the
// raw spelling for suffix dispatch. Missing or unmatched media default to JSON.
func ResponseDecode(media string) Selection {
	return suffix(normalized(media))
}

func normalized(media string) string {
	if parsed, _, err := mime.ParseMediaType(media); err == nil {
		return parsed
	}
	return media
}

func exact(media string) Selection {
	kind := Invalid
	switch media {
	case "", "application/json":
		kind = JSON
		media = "application/json"
	case "application/xml":
		kind = XML
	case "application/gob":
		kind = GOB
	case "text/html", "text/plain":
		kind = Text
	}
	return Selection{Kind: kind, Media: media}
}

func suffix(media string) Selection {
	kind := JSON
	switch {
	case media == "application/xml" || strings.HasSuffix(media, "+xml"):
		kind = XML
	case media == "application/gob" || strings.HasSuffix(media, "+gob"):
		kind = GOB
	case media == "text/html" || media == "text/plain" || strings.HasSuffix(media, "+html") || strings.HasSuffix(media, "+txt"):
		kind = Text
	}
	return Selection{Kind: kind, Media: media}
}
