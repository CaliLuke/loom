package codegen

import (
	"go/doc"
	"go/token"
	"strings"
	"testing"
	"unicode"
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

// FuzzSnakeKebabCase checks SnakeCase and KebabCase on ASCII design names.
// Non-ASCII input is excluded on purpose: SnakeCase processes bytes, not
// runes, so it currently splits or corrupts multi-byte UTF-8 input. Widen the
// seeds and the filter once SnakeCase handles runes.
func FuzzSnakeKebabCase(f *testing.F) {
	for _, s := range identifierFuzzSeeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		if !isPrintableASCII(s) {
			return
		}
		snake := SnakeCase(s)
		kebab := KebabCase(s)
		if strings.ContainsAny(snake, "-/ \tABCDEFGHIJKLMNOPQRSTUVWXYZ") {
			t.Errorf("SnakeCase(%q) = %q contains a separator or upper case letter", s, snake)
		}
		if strings.ContainsAny(kebab, "_/ \tABCDEFGHIJKLMNOPQRSTUVWXYZ") {
			t.Errorf("KebabCase(%q) = %q contains a separator or upper case letter", s, kebab)
		}
		if again := SnakeCase(snake); isDesignWord(s) && again != snake {
			t.Errorf("SnakeCase is not idempotent: %q -> %q -> %q", s, snake, again)
		}
	})
}

func isPrintableASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < 0x20 || s[i] > 0x7e {
			return false
		}
	}
	return true
}

// isDesignWord reports whether s only uses ASCII letters, digits, and
// underscores, the alphabet of conventional design names.
func isDesignWord(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !('a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9' || c == '_') {
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
