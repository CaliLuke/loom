package codegen

import (
	"regexp"
	"testing"

	"github.com/CaliLuke/loom/codegen"
)

// protoIdentifier is the identifier grammar protoBufify guarantees: an ASCII
// letter followed by ASCII letters, digits, or underscores.
var protoIdentifier = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)

// FuzzProtoBufify checks protoBufify never panics and, for any non-empty
// design name, produces message and field names that protoc accepts.
func FuzzProtoBufify(f *testing.F) {
	for _, s := range []string{"", "lower", "UPPER", "foo_bar", "123foo", "foo123bar", "foo_jwt", "user-id", "a.b", "type", "string", "message", "Int32Value", "日本", "\xff", "_", "_1foo", "9Lives", "é1", "foo日bar"} {
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
		if !protoIdentifier.MatchString(got) {
			t.Errorf("protoBufify(%q, %v, %v) = %q is not a protobuf identifier", s, firstUpper, acronym, got)
		}
		if field := codegen.SnakeCase(protoBufify(s, false, false)); !protoIdentifier.MatchString(field) {
			t.Errorf("field name for %q = %q is not a protobuf identifier", s, field)
		}
	})
}
