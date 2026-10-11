package ir

import (
	"encoding/json/v2"
	"os/exec"
	"regexp"
	"testing"

	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/require"
)

func TestPatternProjectionEngineAgreement(t *testing.T) {
	for _, pattern := range []string{`\Afoo\z`, `^foo$`, `^\s$`, `^[\s]+$`, `^\d\w$`, `\bfoo\b`, `^(foo|bar){1,2}$`, `^[a-z]+$`, `^\x41$`, `^[]a\s]+$`, `^\a$`, `^\#$`, `^\-$`, `^[\]a\s]+$`, `[\s-a]`, `[\s-\x7f]`, `[(?]`, `[:abc]`, `[[:a]`, `[[:xyz]`, `[[:]]`, `^*a`, `$?`, `\b+foo`, `\A{2}foo`, `a{01}`, `a{00}`, `a{01,02}`, `a{1,02}`, `(?:^)*a`} {
		t.Run(pattern, func(t *testing.T) {
			projected, err := projectPattern(pattern)
			require.NoError(t, err)
			values := []string{"foo", "Afooz", "foo\n", "foo\r", "foo\u2028", "", " ", "\t", "\n", "\r", "\f", "\v", "\u00a0", "\u2028", "1a", "foox", "foobar", "xfoo", "foofoo", "A", "😀", "]", "a]", "\a", "#", "-", "@", "a", "a{01}", "a{00}", "a{01,02}", "a{1,02}", ":", "?", "(", "[", "[a", ":]"}
			input, err := json.Marshal(struct {
				Pattern string
				Values  []string
			}{projected, values})
			require.NoError(t, err)
			command := exec.Command("node", "-e", `const input=JSON.parse(process.argv[1]); process.stdout.write(JSON.stringify(input.Values.map(s=>new RegExp(input.Pattern).test(s))))`, string(input))
			output, err := command.CombinedOutput()
			require.NoError(t, err, string(output))
			var actual []bool
			require.NoError(t, json.Unmarshal(output, &actual))
			runtimePattern := regexp.MustCompile(pattern)
			for i, value := range values {
				require.Equal(t, runtimePattern.MatchString(value), actual[i], "pattern %q projected %q value %q", pattern, projected, value)
			}
		})
	}
}

func TestPatternProjectionRejectsUnsupportedSyntax(t *testing.T) {
	for _, pattern := range []string{`^.$`, `^[^a]$`, `^\S+$`, `\D`, `\W`, `(?i)foo`, `(?m)^foo$`, `(?s:foo)`, `(?P<name>foo)`, `[[:alpha:]]`, `\p{Greek}`, `\x{41}`, `\123`, `\Qfoo\E`, `😀+`, "[\ud7ff-\ue000]"} {
		t.Run(pattern, func(t *testing.T) {
			require.NotPanics(t, func() {
				regexp.MustCompile(pattern)
			})
			analyzer := NewAnalyzer(expr.NewRandom("rejected-pattern"), false)
			analyzer.AnalyzeSchema(&expr.AttributeExpr{Type: expr.String, Validation: &expr.ValidationExpr{Pattern: pattern}})
			require.ErrorContains(t, analyzer.Err(), "cannot preserve Go validation")
		})
	}
}

func TestStandaloneLocationRejectsUnsupportedPattern(t *testing.T) {
	attribute := &expr.AttributeExpr{Type: expr.String, Validation: &expr.ValidationExpr{Pattern: "^.$"}}
	require.Panics(t, func() {
		preparedLocationSchema(attribute, "standalone parameter", expr.NewRandom("pattern"), false, nil)
	})
}
