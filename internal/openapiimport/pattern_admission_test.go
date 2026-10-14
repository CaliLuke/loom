package openapiimport

import (
	"encoding/json/v2"
	"fmt"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImportedPatternRejectsUnrepresentableMeaning(t *testing.T) {
	for _, pattern := range []string{`^a(?=b)`, `(a)\1`, `(?i)a`, `\Afoo\z`, `a{01}`, `^.$`, `^[^a]$`, `^\D$`, `^[\s-a]$`, `^*a`, `\uD800`, `😀`, `[]`, `[]a]`} {
		t.Run(pattern, func(t *testing.T) {
			document, diagnostics, err := Analyze(importedPatternSource(t, pattern))
			require.NoError(t, err)
			requireDiagnosticCode(t, diagnostics, "schema-pattern")
			require.Contains(t, diagnostics.Error(), "/components/schemas/Value/pattern")
			fatal, warnings := diagnostics.Classify(true)
			require.NotEmpty(t, fatal)
			require.Empty(t, warnings)
			_, err = Render(document, Options{PackageName: "design"})
			require.ErrorContains(t, err, "schema-pattern")
		})
	}
}

func TestImportedPatternPreservesAcceptance(t *testing.T) {
	for _, tc := range []struct {
		pattern string
		accept  []string
		reject  []string
	}{
		{`^foo$`, []string{"foo"}, []string{"Afooz", "foo\n"}},
		{`^[a-z]{1,3}$`, []string{"a", "abc"}, []string{"", "abcd"}},
		{`^(?:foo|bar)+$`, []string{"foo", "foobar"}, []string{"", "foobaz"}},
		{`^\d\w$`, []string{"1a", "2_"}, []string{"aa", "1é"}},
		{`^\s$`, []string{"\v", "\u00a0", "\ufeff", "\u2028"}, []string{"x", "\u0085"}},
		{`^[\s]+$`, []string{"\v\u00a0"}, []string{"x"}},
		{`^\u0041$`, []string{"A"}, []string{"u0041"}},
		{`^[\b]$`, []string{"\b"}, []string{"b"}},
		{`^[\]]$`, []string{"]"}, []string{"["}},
		{`^\u005e\u005b\u005c$`, []string{"^[\\"}, []string{"u005e"}},
		{`^[\u005b-\u005d]$`, []string{"[", "\\", "]"}, []string{"^"}},
		{`^\\u0041$`, []string{`\u0041`}, []string{"A"}},
		{`^a{0,2}?$`, []string{"", "aa"}, []string{"aaa"}},
	} {
		t.Run(tc.pattern, func(t *testing.T) {
			document, diagnostics, err := Analyze(importedPatternSource(t, tc.pattern))
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			pattern, err := regexp.Compile(document.Components.Schemas[0].Schema.Pattern)
			require.NoError(t, err)
			for _, value := range tc.accept {
				require.True(t, pattern.MatchString(value), "pattern %q must accept %q", tc.pattern, value)
			}
			for _, value := range tc.reject {
				require.False(t, pattern.MatchString(value), "pattern %q must reject %q", tc.pattern, value)
			}
		})
	}
}

func TestImportedPatternAdmissionSurvivesSchemaNormalization(t *testing.T) {
	for _, shape := range []string{
		`{"type":"object","properties":{"value":%s}}`,
		`{"type":"array","items":%s}`,
		`{"type":"object","additionalProperties":%s}`,
		`{"anyOf":[%s,{"type":"null"}]}`,
	} {
		t.Run(shape, func(t *testing.T) {
			value := fmt.Sprintf(shape, `{"type":"string","pattern":"a(?=b)"}`)
			source := fmt.Appendf(nil, `{"openapi":"3.1.1","info":{"title":"Patterns","version":"1"},"paths":{},"components":{"schemas":{"Value":%s}}}`, value)
			document, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			requireDiagnosticCode(t, diagnostics, "schema-pattern")
			_, err = Render(document, Options{PackageName: "design"})
			require.ErrorContains(t, err, "schema-pattern")
		})
	}
}

func importedPatternSource(t *testing.T, pattern string) []byte {
	t.Helper()
	encoded, err := json.Marshal(pattern)
	require.NoError(t, err)
	return fmt.Appendf(nil, `{"openapi":"3.1.1","info":{"title":"Patterns","version":"1"},"paths":{},"components":{"schemas":{"Value":{"type":"string","pattern":%s}}}}`, encoded)
}
