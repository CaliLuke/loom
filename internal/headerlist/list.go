// Package headerlist owns schema-based HTTP simple collection framing.
package headerlist

import (
	"errors"
	"fmt"
	"strings"
)

// Encode frames logical items as a simple comma list, rejecting lossy values.
// The caller owns field presence; both nil and empty items encode as empty text.
func Encode(items []string) (string, error) {
	for i, item := range items {
		if err := ValidateItem(item); err != nil {
			return "", fmt.Errorf("item %d: %w", i, err)
		}
	}
	return strings.Join(items, ", "), nil
}

// Decode combines ordered physical lines and discards empty list members as
// required by RFC 9110. It preserves absent versus present empty collections.
// Quotes are literal characters, not an escape for the comma delimiter.
func Decode(lines []string, present bool) []string {
	if !present {
		return nil
	}
	items := make([]string, 0)
	for _, line := range lines {
		for item := range strings.SplitSeq(line, ",") {
			item = strings.Trim(item, " \t")
			if item != "" {
				items = append(items, item)
			}
		}
	}
	return items
}

// ValidateItem admits values whose identity survives simple header framing.
// Errors describe the restriction without including potentially secret values.
func ValidateItem(item string) error {
	switch {
	case item == "":
		return errors.New("empty items cannot be represented in a header list")
	case strings.Contains(item, ","):
		return errors.New("commas cannot be represented inside a header list item")
	case strings.Trim(item, " \t") != item:
		return errors.New("leading or trailing whitespace cannot be represented in a header list item")
	default:
		return nil
	}
}
