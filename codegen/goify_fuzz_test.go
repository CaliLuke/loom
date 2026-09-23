package codegen

import (
	"go/doc"
	"go/token"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"
)

// identifierFuzzSeeds is a realistic corpus of design names: service,
// method, attribute, and type names as authored in DSLs, plus the edge
// shapes that exercise acronyms, reserved words, digits, and separators.
var identifierFuzzSeeds = []string{
	"",
	"id",
	"user_id",
	"UserID",
	"user-id",
	"user id",
	"APIVersion",
	"api_version",
	"OAuth2Token",
	"oauth",
	"JWTAuth",
	"http_status",
	"type",
	"type_",
	"func",
	"string",
	"nil",
	"fmt",
	"123foo",
	"9",
	"_",
	"__foo__bar__",
	"foo:bar",
	":foo",
	"a.b.c",
	"x-request-id",
	"IDs",
	"UUIDv4",
	"éclair",
	"Ünïcödé",
	"日本語",
	"ǅemal",
	"١٢٣",
	"\xff\xfe",
	"CNNNews",
	"HTTPServer2XX",
}

// FuzzGoify checks that Goify always produces a valid, non-reserved Go
// identifier and never panics, whatever the design name.
func FuzzGoify(f *testing.F) {
	for _, s := range identifierFuzzSeeds {
		f.Add(s, true)
		f.Add(s, false)
	}
	f.Fuzz(func(t *testing.T, s string, firstUpper bool) {
		defer resetStringCacheForFuzz()
		got := Goify(s, firstUpper)
		if s == "" {
			if got != "" {
				t.Errorf("Goify(%q, %v) = %q, want empty", s, firstUpper, got)
			}
			return
		}
		if !token.IsIdentifier(got) {
			t.Errorf("Goify(%q, %v) = %q is not a valid Go identifier", s, firstUpper, got)
			return
		}
		if token.IsKeyword(got) || doc.IsPredeclared(got) || isPackage[got] {
			t.Errorf("Goify(%q, %v) = %q is a reserved Go name", s, firstUpper, got)
		}
		if firstUpper && !token.IsExported(got) {
			t.Errorf("Goify(%q, true) = %q is not an exported identifier", s, got)
		}
		if !firstUpper && token.IsExported(got) {
			t.Errorf("Goify(%q, false) = %q is an exported identifier", s, got)
		}
	})
}

// FuzzCamelCase checks that CamelCase only keeps letters and digits and never
// panics.
func FuzzCamelCase(f *testing.F) {
	for _, s := range identifierFuzzSeeds {
		f.Add(s, true, true)
		f.Add(s, false, false)
	}
	f.Fuzz(func(t *testing.T, s string, firstUpper, acronym bool) {
		defer resetStringCacheForFuzz()
		got := CamelCase(s, firstUpper, acronym)
		for _, r := range got {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
				t.Errorf("CamelCase(%q, %v, %v) = %q contains non identifier rune %q", s, firstUpper, acronym, got, r)
			}
		}
		if again := CamelCase(got, firstUpper, acronym); got != "" && !strings.EqualFold(again, got) {
			t.Errorf("CamelCase is not letter-preserving on its own output: %q -> %q -> %q", s, got, again)
		}
	})
}

// snakeCaseFuzzSeeds adds non-ASCII design names that exercise rune
// boundaries in SnakeCase and KebabCase: accented, title-case, combining,
// caseless, astral, and invalid UTF-8 input.
var snakeCaseFuzzSeeds = []string{
	"ÉtéFoo",
	"CaféBar",
	"HTTPÜber",
	"ÜBERHttp",
	"IDÉtat",
	"ΣίσυφοςTest",
	"ПриветМир",
	"fooǅemal",
	"Cafe\u0301Bar",
	"ABE\u0301t",
	"Foo日本Bar",
	"日本API",
	"😀Emoji",
	"été\u00a0Foo",
	"été-Foo/Bar",
	"ϒϒb",
	"a\xffB",
	"\xe6\x97",
}

// FuzzSnakeKebabCase checks that SnakeCase and KebabCase produce valid UTF-8
// without separators or upper case runes, keep every input rune intact, and
// match the historical byte-wise output for ASCII input.
func FuzzSnakeKebabCase(f *testing.F) {
	for _, s := range identifierFuzzSeeds {
		f.Add(s)
	}
	for _, s := range snakeCaseFuzzSeeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		snake := SnakeCase(s)
		kebab := KebabCase(s)
		checkCaseOutput(t, "SnakeCase", s, snake, '-')
		checkCaseOutput(t, "KebabCase", s, kebab, '_')
		if isASCII(s) {
			if want := asciiSnakeCase(s); snake != want {
				t.Errorf("SnakeCase(%q) = %q, want historical ASCII output %q", s, snake, want)
			}
			if want := asciiKebabCase(s); kebab != want {
				t.Errorf("KebabCase(%q) = %q, want historical ASCII output %q", s, kebab, want)
			}
		}
		if utf8.ValidString(s) {
			if want, got := caseRunes(s), strings.ReplaceAll(snake, "_", ""); got != want {
				t.Errorf("SnakeCase(%q) = %q changes runes: got %q, want %q", s, snake, got, want)
			}
			if want, got := caseRunes(s), strings.ReplaceAll(kebab, "-", ""); got != want {
				t.Errorf("KebabCase(%q) = %q changes runes: got %q, want %q", s, kebab, got, want)
			}
		}
		if again := SnakeCase(snake); isDesignWord(s) && again != snake {
			t.Errorf("SnakeCase is not idempotent: %q -> %q -> %q", s, snake, again)
		}
	})
}

// checkCaseOutput reports output that is not valid UTF-8 or that contains
// white space, a slash, the other case's separator, or an upper or title case
// rune.
func checkCaseOutput(t *testing.T, fn, in, out string, sep rune) {
	t.Helper()
	if !utf8.ValidString(out) {
		t.Errorf("%s(%q) = %q is not valid UTF-8", fn, in, out)
		return
	}
	for _, r := range out {
		if r == sep || r == '/' || unicode.IsSpace(r) || unicode.ToLower(r) != r {
			t.Errorf("%s(%q) = %q contains separator or upper case rune %q", fn, in, out, r)
			return
		}
	}
}

// caseRunes returns the lower case runes of s without the white space, dash,
// slash, and underscore separators that SnakeCase and KebabCase rewrite.
func caseRunes(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r == '-' || r == '_' || r == '/' || unicode.IsSpace(r) {
			continue
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

// asciiSnakeCase is the historical byte-wise SnakeCase. It is the oracle for
// ASCII input, whose output must not change.
func asciiSnakeCase(name string) string {
	for u, l := range toLower {
		name = strings.ReplaceAll(name, u, l)
	}
	name = strings.Join(strings.Fields(name), "_")
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, "/", "_")
	ln := len(name)
	if ln == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteRune(unicode.ToLower(rune(name[0])))
	var lastLower, lastUnder bool
	for i := 1; i < ln; i++ {
		r := rune(name[i])
		isLower := unicode.IsLower(r) && unicode.IsLetter(r) || unicode.IsDigit(r)
		isUnder := r == '_'
		if !isLower && !isUnder {
			if lastLower && !lastUnder {
				b.WriteRune('_')
			} else if ln > i+1 {
				rn := rune(name[i+1])
				if unicode.IsLower(rn) && rn != '_' && !lastUnder {
					b.WriteRune('_')
				}
			}
		}
		b.WriteRune(unicode.ToLower(r))
		lastLower = isLower
		lastUnder = isUnder
	}
	return b.String()
}

// asciiKebabCase is the historical KebabCase built on asciiSnakeCase.
func asciiKebabCase(name string) string {
	name = strings.TrimSuffix(asciiSnakeCase(name), "_")
	return strings.ReplaceAll(name, "_", "-")
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

// isDesignWord reports whether s is valid UTF-8 that only uses letters,
// digits, combining marks, and underscores, the alphabet of conventional
// design names in any script.
func isDesignWord(s string) bool {
	if !utf8.ValidString(s) {
		return false
	}
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.IsMark(r) && r != '_' {
			return false
		}
	}
	return true
}

// resetStringCacheForFuzz bounds the memoization cache during fuzzing so the
// fuzz workers do not accumulate every generated input.
func resetStringCacheForFuzz() {
	globalStringCache.mu.Lock()
	defer globalStringCache.mu.Unlock()
	if len(globalStringCache.cache) > 4096 {
		globalStringCache.cache = make(map[cacheKey]string)
	}
}
