package custom

import (
	"errors"
	"strings"
)

// Token is a shared fixture String representation whose parser rejects
// the bad prefix, so an authored default can fail only at runtime.
type Token string

// MarshalText returns the token text.
func (t Token) MarshalText() ([]byte, error) {
	return []byte(t), nil
}

// UnmarshalText admits any text without the bad prefix.
func (t *Token) UnmarshalText(text []byte) error {
	if strings.HasPrefix(string(text), "bad") {
		return errors.New("rejected token")
	}
	*t = Token(text)
	return nil
}
