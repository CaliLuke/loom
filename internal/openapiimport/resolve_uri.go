package openapiimport

import (
	"encoding/json/jsontext"
	"errors"
	"net/url"
	"strings"
	"unicode/utf8"
)

// normalizeURI normalizes identity syntax without interpreting a JSON Pointer.
// Fragment decoding belongs exclusively to decodePointer.
func normalizeURI(value string) (docURI, error) {
	u, err := parseDescriptionURI(value)
	if err != nil {
		return "", err
	}
	if !u.IsAbs() {
		return "", errors.New("document identity requires an absolute URI")
	}
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	if u.Scheme == "http" && u.Port() == "80" || u.Scheme == "https" && u.Port() == "443" {
		u.Host = strings.TrimSuffix(u.Host, ":"+u.Port())
	}
	path, err := normalizeURIEscapes(u.EscapedPath())
	if err != nil {
		return "", err
	}
	path = removeURIDotSegments(path)
	if (u.Scheme == "http" || u.Scheme == "https") && path == "" {
		path = "/"
	}
	u.Path, err = url.PathUnescape(path)
	if err != nil {
		return "", errors.New("invalid URI path escape")
	}
	u.RawPath = path
	u.RawQuery, err = normalizeURIEscapes(u.RawQuery)
	if err != nil {
		return "", err
	}
	u.Opaque, err = normalizeURIEscapes(u.Opaque)
	if err != nil {
		return "", err
	}
	u.Fragment, u.RawFragment = "", ""
	return docURI(u.String()), nil
}

func decodePointer(fragment string) (string, error) {
	decoded, err := url.PathUnescape(fragment)
	if err != nil || !utf8.ValidString(decoded) || !jsontext.Pointer(decoded).IsValid() {
		return "", errors.New("invalid JSON Pointer fragment")
	}
	return decoded, nil
}

func emitPointer(pointer string) string {
	const hex = "0123456789ABCDEF"
	var result strings.Builder
	for i := range len(pointer) {
		b := pointer[i]
		if uriUnreserved(b) || strings.ContainsRune("!$&'()*+,;=:@/?", rune(b)) {
			result.WriteByte(b)
		} else {
			result.WriteByte('%')
			result.WriteByte(hex[b>>4])
			result.WriteByte(hex[b&15])
		}
	}
	return result.String()
}

func resolveReference(value string, base docURI) (docURI, string, error) {
	u, err := parseDescriptionURI(value)
	if err != nil {
		return "", "", err
	}
	pointer, err := decodePointer(u.EscapedFragment())
	if err != nil {
		return "", "", err
	}
	if value == "" || strings.HasPrefix(value, "#") {
		return "", pointer, nil
	}
	if !u.IsAbs() {
		if base == "" {
			return "", "", errors.New("the entry has no base URI to resolve this reference")
		}
		parent, err := parseDescriptionURI(string(base))
		if err != nil {
			return "", "", err
		}
		u = parent.ResolveReference(u)
	}
	identity, err := normalizeURI(u.String())
	return identity, pointer, err
}

// parseDescriptionURI never includes userinfo or a possibly secret-bearing URI
// in an error. It accepts URI references, not unescaped IRI characters.
func parseDescriptionURI(value string) (*url.URL, error) {
	for _, b := range []byte(value) {
		if b <= ' ' || b >= 127 || strings.ContainsRune("<>\"{}|\\^`", rune(b)) {
			return nil, errors.New("invalid URI reference syntax")
		}
	}
	u, err := url.Parse(value)
	if err != nil {
		return nil, errors.New("invalid URI reference syntax")
	}
	if u.User != nil {
		return nil, errors.New("URI userinfo is not permitted")
	}
	if _, err := normalizeURIEscapes(u.RawQuery); err != nil {
		return nil, err
	}
	return u, nil
}

func normalizeURIEscapes(value string) (string, error) {
	const hex = "0123456789ABCDEF"
	var result strings.Builder
	for i := 0; i < len(value); i++ {
		if value[i] != '%' {
			result.WriteByte(value[i])
			continue
		}
		if i+2 >= len(value) {
			return "", errors.New("invalid URI escape")
		}
		decoded, err := url.PathUnescape(value[i : i+3])
		if err != nil {
			return "", errors.New("invalid URI escape")
		}
		b := decoded[0]
		if uriUnreserved(b) {
			result.WriteByte(b)
		} else {
			result.WriteByte('%')
			result.WriteByte(hex[b>>4])
			result.WriteByte(hex[b&15])
		}
		i += 2
	}
	return result.String(), nil
}

func uriUnreserved(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' || strings.ContainsRune("-._~", rune(b))
}

// removeURIDotSegments implements RFC 3986 section 5.2.4 without collapsing
// repeated slashes, which identify different paths.
func removeURIDotSegments(input string) string {
	var output string
	for input != "" {
		switch {
		case strings.HasPrefix(input, "../"):
			input = input[3:]
		case strings.HasPrefix(input, "./"):
			input = input[2:]
		case strings.HasPrefix(input, "/./"):
			input = input[2:]
		case input == "/.":
			input = "/"
		case strings.HasPrefix(input, "/../") || input == "/..":
			if input == "/.." {
				input = "/"
			} else {
				input = input[3:]
			}
			if i := strings.LastIndexByte(output, '/'); i >= 0 {
				output = output[:i]
			} else {
				output = ""
			}
		case input == "." || input == "..":
			input = ""
		default:
			start := 0
			if input[0] == '/' {
				start = 1
			}
			end := strings.IndexByte(input[start:], '/')
			if end < 0 {
				output += input
				input = ""
			} else {
				end += start
				output += input[:end]
				input = input[end:]
			}
		}
	}
	return output
}
