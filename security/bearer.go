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
