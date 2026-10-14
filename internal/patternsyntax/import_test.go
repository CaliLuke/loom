package patternsyntax

import (
	"encoding/json/v2"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImportPatternEngineAgreement(t *testing.T) {
	initialPatterns := []string{"", "^$", "a|", "|a", "(?:)", "(?:^)*a", `\bfoo\b`, `\B`, `^[\]]$`, `^\u005e\u005b\u005c$`, `^[\u005b-\u005d]$`, `^\\u0041$`, `^[\b]$`, `[-a]`, `[a-]`, `[a-b-c]`, `[[]`}
	atoms := []string{"a", "é", "中", `\d`, `\w`, `\s`, `[\s]`, `[a-z]`, `[\u0041-\u005a]`, `\u0000`, `\x7f`, `\f`, `\n`, `\r`, `\t`, `\v`, `\[`, `\]`, `\-`, `\/`, "(?:a|b)"}
	patterns := make([]string, 0, len(initialPatterns)+16*len(atoms))
	patterns = append(patterns, initialPatterns...)
	for _, atom := range atoms {
		for _, repeat := range []string{"", "?", "*", "+", "{0}", "{1,2}", "{2,}", "{0,2}?"} {
			patterns = append(patterns, "^(?:"+atom+")"+repeat+"$", "x"+atom+"|"+atom+"y")
		}
	}
	values := []string{"", "a", "aa", "aaa", "b", "ab", "é", "éé", "中", "😀", "a😀", "foo", "xfoo", "foo\n", "foo\r", "foo\u2028", "1", "_", "a_b", "A", "Z", "xay", "xé", "ay", "\x00", "\a", "\b", "\t", "\n", "\v", "\f", "\r", " ", "\u0085", "\u00a0", "\u1680", "\u2000", "\u200a", "\u2028", "\u2029", "\u202f", "\u205f", "\u3000", "\ufeff", "\x7f", "[", "]", "\\", "^[\\", `\u0041`, "-", "/"}
	translations := make([]string, len(patterns))
	exports := make([]string, len(patterns))
	for i, pattern := range patterns {
		var err error
		translations[i], err = FromECMA(pattern)
		require.NoError(t, err, "source %q", pattern)
		exports[i], err = ToECMA(translations[i])
		require.NoError(t, err, "source %q translated %q", pattern, translations[i])
	}
	input, err := json.Marshal(struct {
		Patterns []string
		Exports  []string
		Values   []string
	}{patterns, exports, values})
	require.NoError(t, err)
	command := exec.Command("node", "-e", `const fs=require('fs');const x=JSON.parse(fs.readFileSync(0,'utf8'));process.stdout.write(JSON.stringify([x.Patterns,x.Exports].map(ps=>ps.map(p=>x.Values.map(v=>new RegExp(p).test(v))))));`)
	command.Stdin = strings.NewReader(string(input))
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
	var results [][][]bool
	require.NoError(t, json.Unmarshal(output, &results))
	require.Len(t, results, 2)
	for i, pattern := range patterns {
		compiled, err := regexp.Compile(translations[i])
		require.NoError(t, err)
		for j, value := range values {
			require.Equal(t, results[0][i][j], compiled.MatchString(value), "source %q translated %q value %q", pattern, translations[i], value)
			require.Equal(t, results[0][i][j], results[1][i][j], "round trip %q exported %q value %q", pattern, exports[i], value)
		}
	}
}

func TestImportPatternRejectsAmbiguousSyntax(t *testing.T) {
	for _, pattern := range []string{`(?=a)`, `(?!a)`, `(?<=a)`, `(a)\1`, `(?<name>a)`, `(?i:a)`, `\a`, `\A`, `\z`, `\0`, `\123`, `\cA`, `\p{L}`, `\u{41}`, `\uD800`, `\uDC00`, `\xG1`, `\`, `.`, `[^a]`, `\D`, `\W`, `\S`, `[]`, `[]a]`, `[a-\s]`, `[\s-a]`, `a{01}`, `a{1,02}`, `a{,2}`, `^*a`, `\b+`, `a**`, `(`, `)`, `[a`, "😀", string([]byte{0xff}), strings.Repeat("(", 257) + "a" + strings.Repeat(")", 257)} {
		t.Run(pattern, func(t *testing.T) {
			_, err := FromECMA(pattern)
			require.Error(t, err)
		})
	}
}
