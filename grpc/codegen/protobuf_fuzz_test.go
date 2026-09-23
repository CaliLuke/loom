package codegen

import (
	"regexp"
	"testing"
	"unicode"

	"github.com/CaliLuke/loom/codegen"
)

// protoIdentifier is the protocol buffers identifier grammar.
var protoIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// FuzzProtoBufify checks protoBufify never panics, only emits identifier
// runes, and, for ASCII design names that start with a letter, produces
// message and field names that protoc accepts.
func FuzzProtoBufify(f *testing.F) {
	for _, s := range []string{"", "lower", "UPPER", "foo_bar", "123foo", "foo123bar", "foo_jwt", "user-id", "a.b", "type", "string", "message", "Int32Value", "日本", "\xff"} {
		f.Add(s, true, true)
		f.Add(s, false, false)
	}
	f.Fuzz(func(t *testing.T, s string, firstUpper, acronym bool) {
		got := protoBufify(s, firstUpper, acronym)
		if s == "" {
			if got != "" {
				t.Errorf("protoBufify(%q) = %q, want empty", s, got)
			}
			return
		}
		if got == "" {
			t.Errorf("protoBufify(%q) is empty", s)
		}
		for _, r := range got {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
				t.Errorf("protoBufify(%q) = %q contains rune %q", s, got, r)
			}
		}
		if !startsWithASCIILetterWord(s) {
			return
		}
		if !protoIdentifier.MatchString(got) {
			t.Errorf("protoBufify(%q, %v, %v) = %q is not a protobuf identifier", s, firstUpper, acronym, got)
		}
		if field := codegen.SnakeCase(protoBufify(s, false, false)); !protoIdentifier.MatchString(field) {
			t.Errorf("field name for %q = %q is not a protobuf identifier", s, field)
		}
	})
}

// startsWithASCIILetterWord reports whether s is an ASCII design name that
// starts with a letter.
func startsWithASCIILetterWord(s string) bool {
	if s == "" || !('a' <= s[0] && s[0] <= 'z' || 'A' <= s[0] && s[0] <= 'Z') {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < 0x20 || s[i] > 0x7e {
			return false
		}
	}
	return true
}
