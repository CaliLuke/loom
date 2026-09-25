package naming

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	// toLower lists the casing exceptions that SnakeCase applies before it
	// splits words.
	toLower = map[string]string{"OAuth": "oauth"}

	// commonInitialisms lists the initialisms that CamelCase keeps in a
	// single case.
	commonInitialisms = map[string]bool{
		"API":   true,
		"ASCII": true,
		"CPU":   true,
		"CSS":   true,
		"DNS":   true,
		"EOF":   true,
		"GUID":  true,
		"HTML":  true,
		"HTTP":  true,
		"HTTPS": true,
		"ID":    true,
		"IP":    true,
		"JMES":  true,
		"JSON":  true,
		"JWT":   true,
		"LHS":   true,
		"OK":    true,
		"QPS":   true,
		"RAM":   true,
		"RHS":   true,
		"RPC":   true,
		"SDK":   true,
		"SLA":   true,
		"SMTP":  true,
		"SQL":   true,
		"SSH":   true,
		"TCP":   true,
		"TLS":   true,
		"TTL":   true,
		"UDP":   true,
		"UI":    true,
		"UID":   true,
		"UUID":  true,
		"URI":   true,
		"URL":   true,
		"UTF8":  true,
		"VM":    true,
		"XML":   true,
		"XSRF":  true,
		"XSS":   true,
	}
)

// CamelCase produces the CamelCase version of the given string. It removes any
// non letter and non digit character.
//
// If firstUpper is true the first letter of the string is capitalized else
// the first letter is in lowercase.
//
// If acronym is true and a part of the string is a common acronym
// then it keeps the part capitalized (firstUpper = true)
// (e.g. APIVersion) or lowercase (firstUpper = false) (e.g. apiVersion).
func CamelCase(name string, firstUpper, acronym bool) string {
	if name == "" {
		return ""
	}

	// Use cache to avoid recomputing the same transformation
	key := cacheKey{
		input:      name,
		firstUpper: firstUpper,
		acronym:    acronym,
		operation:  "camel",
	}
	return globalStringCache.getCached(key, func() string {
		return camelCaseUncached(name, firstUpper, acronym)
	})
}

// EscapeNonASCII escapes every non-ASCII rune of name so that the result can
// serve where only ASCII is valid, such as protocol buffer package names and
// Go import paths. A rune in the Basic Multilingual Plane becomes "u" and four
// lower case hex digits; any other rune becomes "U" and eight. The fixed
// widths keep distinct names distinct, and ASCII names are returned unchanged.
func EscapeNonASCII(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r < utf8.RuneSelf:
			b.WriteRune(r)
		case r <= 0xFFFF:
			fmt.Fprintf(&b, "u%04x", r)
		default:
			fmt.Fprintf(&b, "U%08x", r)
		}
	}
	return b.String()
}

// SnakeCase produces the snake_case version of the given CamelCase string.
// News    => news
// OldNews => old_news
// CNNNews => cnn_news
// ÉtéFoo  => été_foo
//
// SnakeCase processes runes, not bytes. Letters without case, such as CJK
// ideographs, continue the current word like lower case letters do. Upper and
// title case letters that have a lower case form start a new word. Combining marks stay attached to the
// rune they follow. Invalid UTF-8 bytes become the Unicode replacement
// character, so the result is always valid UTF-8.
func SnakeCase(name string) string {
	// Special handling for single "words" starting with multiple upper case letters
	for u, l := range toLower {
		name = strings.ReplaceAll(name, u, l)
	}

	// Remove leading and trailing blank spaces and replace any blank spaces in
	// between with a single underscore
	name = strings.Join(strings.Fields(name), "_")

	// Special handling for dashes and slashes to convert them into underscores
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, "/", "_")

	runes := []rune(name)
	ln := len(runes)
	if ln == 0 {
		return ""
	}
	var b strings.Builder
	b.Grow(len(name) + ln/2)
	b.WriteRune(unicode.ToLower(runes[0]))
	var lastLower, lastUnder bool
	for i := 1; i < ln; i++ {
		r := runes[i]
		if unicode.IsMark(r) {
			b.WriteRune(r)
			continue
		}
		isLower := snakeContinuesWord(r)
		isUnder := r == '_'
		if !isLower && !isUnder {
			if lastLower && !lastUnder {
				b.WriteRune('_')
			} else if rn, ok := nextNonMark(runes, i+1); ok && unicode.IsLower(rn) && !lastUnder {
				b.WriteRune('_')
			}
		}
		b.WriteRune(unicode.ToLower(r))
		lastLower = isLower
		lastUnder = isUnder
	}
	return b.String()
}

// KebabCase produces the kebab-case version of the given CamelCase string.
func KebabCase(name string) string {
	name = SnakeCase(name)
	ln := len(name)
	if ln == 0 {
		return ""
	}
	if name[ln-1] == '_' {
		name = name[:ln-1]
	}
	return strings.ReplaceAll(name, "_", "-")
}

// Title returns s with the first letter of each word in upper case and the
// other letters unchanged.
func Title(s string) string {
	return cases.Title(language.Und, cases.NoLower).String(s)
}

// camelCaseUncached is the original implementation without caching.
func camelCaseUncached(name string, firstUpper, acronym bool) string {
	runes := []rune(name)
	runes = removeTrailingInvalid(runes)
	if len(runes) == 0 {
		return ""
	}

	w, i := 0, 0
	for i+1 <= len(runes) {
		var eow bool
		runes, i, eow = advanceCamelWord(runes, i)
		if !eow {
			continue
		}
		normalizeCamelWord(runes, w, i, firstUpper, acronym)
		w = i
	}

	return string(runes)
}

func advanceCamelWord(runes []rune, i int) ([]rune, int, bool) {
	runes = removeInvalidAtIndex(i, runes)
	eow := false
	switch {
	case i+1 == len(runes):
		eow = true
	case !validIdentifier(runes[i]):
		runes = append(runes[:i], runes[i+1:]...)
	case runes[i+1] == '_':
		eow = true
		n := countAdjacentUnderscores(runes, i)
		copy(runes[i+1:], runes[i+n+1:])
		runes = runes[:len(runes)-n]
	case isLower(runes[i]) && !isLower(runes[i+1]):
		eow = true
	}
	return runes, i + 1, eow
}

func countAdjacentUnderscores(runes []rune, i int) int {
	n := 1
	for i+n+1 < len(runes) && runes[i+n+1] == '_' {
		n++
	}
	return n
}

func normalizeCamelWord(runes []rune, start, end int, firstUpper, acronym bool) {
	word := string(runes[start:end])
	if normalizeCamelInitialism(runes, word, start, firstUpper, acronym) {
		return
	}
	if start > 0 && strings.ToLower(word) == word {
		runes[start] = unicode.ToUpper(runes[start])
	} else if start == 0 && strings.ToLower(word) == word && firstUpper {
		runes[start] = unicode.ToUpper(runes[start])
	}
	if start == 0 && !firstUpper {
		runes[start] = unicode.ToLower(runes[start])
	}
}

func normalizeCamelInitialism(runes []rune, word string, start int, firstUpper, acronym bool) bool {
	upper := strings.ToUpper(word)
	if !commonInitialisms[upper] {
		return false
	}
	switch {
	case firstUpper && acronym:
	case firstUpper && !acronym:
		upper = Title(strings.ToLower(upper))
	case start > 0 && !acronym:
		upper = Title(strings.ToLower(upper))
	case start == 0:
		upper = strings.ToLower(upper)
	}
	copy(runes[start:], []rune(upper))
	return true
}

// isLower returns true if the character is considered a lower case character
// when transforming word into CamelCase.
func isLower(r rune) bool {
	return unicode.IsDigit(r) || unicode.IsLower(r)
}

// validIdentifier returns true if the rune is a letter or number
func validIdentifier(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

// removeTrailingInvalid removes trailing invalid identifiers from runes.
func removeTrailingInvalid(runes []rune) []rune {
	valid := len(runes) - 1
	for ; valid >= 0 && !validIdentifier(runes[valid]); valid-- {
	}

	return runes[0 : valid+1]
}

// removeInvalidAtIndex removes consecutive invalid identifiers from runes starting at index i.
func removeInvalidAtIndex(i int, runes []rune) []rune {
	valid := i
	for ; valid < len(runes) && !validIdentifier(runes[valid]); valid++ {
	}

	return append(runes[:i], runes[valid:]...)
}

// snakeContinuesWord reports whether r continues the current SnakeCase word:
// a digit, or a letter that lower casing leaves unchanged. Only upper and
// title case letters that have a lower case form start a new word, so upper
// case letters without one, such as U+03D2, behave like caseless letters and
// SnakeCase stays idempotent. For ASCII this is exactly the lower case
// letters and digits.
func snakeContinuesWord(r rune) bool {
	return unicode.IsDigit(r) || unicode.IsLetter(r) && unicode.ToLower(r) == r
}

// nextNonMark returns the first rune of runes at or after index i that is not
// a combining mark, so marks do not hide the case of the next letter.
func nextNonMark(runes []rune, i int) (rune, bool) {
	for ; i < len(runes); i++ {
		if !unicode.IsMark(runes[i]) {
			return runes[i], true
		}
	}
	return 0, false
}
