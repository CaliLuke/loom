package security

import (
	"strings"
	"unicode"
)

// DecodeBearer extracts an Authorization bearer credential. The scheme is
// case-insensitive and separated from the nonempty token by one or more spaces.
// Tokens cannot contain whitespace. Raw tokens and other schemes are rejected;
// token content constraints and authentication remain the caller's responsibility.
func DecodeBearer(value string) (string, bool) {
	scheme, token, found := strings.Cut(value, " ")
	if !found || !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}
	token = strings.TrimLeft(token, " ")
	if token == "" || strings.ContainsFunc(token, unicode.IsSpace) {
		return "", false
	}
	return token, true
}

// IsOtherAuthorizationScheme reports whether value has a syntactically valid
// Authorization scheme token other than expected, followed by a space and a
// nonempty credential. It classifies framing only; it does not authenticate or
// validate the foreign credential. Generated decoders use it only when the
// optional credential has an independent authentication alternative.
func IsOtherAuthorizationScheme(value, expected string) bool {
	scheme, credential, found := strings.Cut(value, " ")
	if !found || scheme == "" || strings.EqualFold(scheme, expected) || strings.TrimSpace(credential) == "" {
		return false
	}
	for _, c := range scheme {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune("!#$%&'*+-.^_`|~", c) {
			continue
		}
		return false
	}
	return true
}
