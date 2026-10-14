package patternsyntax

import (
	"fmt"
	"regexp/syntax"
	"strings"
)

// ToECMA preserves Go MatchString semantics over Unicode scalar strings
// in an ECMA-262 engine without flags. Unsupported constructs fail closed.
func ToECMA(pattern string) (string, error) {
	parsed, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		return "", fmt.Errorf("invalid Go pattern: %w", err)
	}
	if err := portablePatternTree(parsed); err != nil {
		return "", err
	}
	if err := portablePatternSyntax(pattern); err != nil {
		return "", err
	}
	return renderPattern(parsed), nil
}

// portablePatternSyntax checks authored extensions before the parser erases them.
func portablePatternSyntax(pattern string) error {
	inClass := false
	classStart := -1
	for i := 0; i < len(pattern); i++ {
		ch := pattern[i]
		if !inClass && ch == '(' && strings.HasPrefix(pattern[i:], "(?") && !strings.HasPrefix(pattern[i:], "(?:") {
			return fmt.Errorf("inline flags and named captures are not in the portable pattern subset")
		}
		if ch == '[' {
			if inClass && isPOSIXClass(pattern[i:]) {
				return fmt.Errorf("POSIX character classes are not in the portable pattern subset")
			}
			if !inClass {
				inClass, classStart = true, i
			}
		} else if ch == ']' && inClass && i != classStart+1 && !(i == classStart+2 && pattern[classStart+1] == '^') {
			inClass = false
		}
		if ch != '\\' {
			continue
		}
		i++ // syntax.Parse has already rejected a trailing escape.
		escaped := pattern[i]
		if escaped == 'x' && i+1 < len(pattern) && pattern[i+1] == '{' {
			return fmt.Errorf("braced hex escapes are not in the portable pattern subset")
		}
		if strings.ContainsRune("AzsadwbBnrtfvx", rune(escaped)) {
			continue
		}
		if escaped >= '0' && escaped <= '9' || escaped >= 'A' && escaped <= 'Z' || escaped >= 'a' && escaped <= 'z' {
			return fmt.Errorf("escape \\%c is not in the portable pattern subset", escaped)
		}
	}
	return nil
}

func isPOSIXClass(fragment string) bool {
	if !strings.HasPrefix(fragment, "[:") {
		return false
	}
	name := fragment[2:]
	name = strings.TrimPrefix(name, "^")
	for i, r := range name {
		if r < 'a' || r > 'z' {
			return i > 0 && strings.HasPrefix(name[i:], ":]")
		}
	}
	return false
}

// renderPattern derives target syntax from parsed meaning, never interpolating
// authored class fragments or repeat spellings into another dialect.
func renderPattern(pattern *syntax.Regexp) string {
	switch pattern.Op {
	case syntax.OpNoMatch:
		return "(?!)"
	case syntax.OpEmptyMatch:
		return "(?:)"
	case syntax.OpLiteral:
		var out strings.Builder
		for _, r := range pattern.Rune {
			out.WriteString(patternRune(r, false))
		}
		return out.String()
	case syntax.OpCharClass:
		var out strings.Builder
		out.WriteByte('[')
		for i := 0; i < len(pattern.Rune); i += 2 {
			out.WriteString(patternRune(pattern.Rune[i], true))
			if pattern.Rune[i] != pattern.Rune[i+1] {
				out.WriteByte('-')
				out.WriteString(patternRune(pattern.Rune[i+1], true))
			}
		}
		out.WriteByte(']')
		return out.String()
	case syntax.OpBeginText:
		return "^"
	case syntax.OpEndText:
		return "$"
	case syntax.OpWordBoundary:
		return `\b`
	case syntax.OpNoWordBoundary:
		return `\B`
	case syntax.OpCapture:
		return "(" + renderPattern(pattern.Sub[0]) + ")"
	case syntax.OpConcat, syntax.OpAlternate:
		parts := make([]string, len(pattern.Sub))
		for i, child := range pattern.Sub {
			parts[i] = renderPattern(child)
			if pattern.Op == syntax.OpConcat && child.Op == syntax.OpAlternate {
				parts[i] = "(?:" + parts[i] + ")"
			}
		}
		separator := ""
		if pattern.Op == syntax.OpAlternate {
			separator = "|"
		}
		return strings.Join(parts, separator)
	case syntax.OpStar, syntax.OpPlus, syntax.OpQuest, syntax.OpRepeat:
		return renderPatternRepeat(pattern)
	default:
		panic("unvalidated pattern operation")
	}
}

func renderPatternRepeat(pattern *syntax.Regexp) string {
	child := pattern.Sub[0]
	operand := renderPattern(child)
	if child.Op != syntax.OpCapture && child.Op != syntax.OpCharClass && !(child.Op == syntax.OpLiteral && len(child.Rune) == 1) {
		operand = "(?:" + operand + ")"
	}
	suffix := map[syntax.Op]string{syntax.OpStar: "*", syntax.OpPlus: "+", syntax.OpQuest: "?"}[pattern.Op]
	if pattern.Op == syntax.OpRepeat {
		switch {
		case pattern.Min == pattern.Max:
			suffix = fmt.Sprintf("{%d}", pattern.Min)
		case pattern.Max < 0:
			suffix = fmt.Sprintf("{%d,}", pattern.Min)
		default:
			suffix = fmt.Sprintf("{%d,%d}", pattern.Min, pattern.Max)
		}
	}
	if pattern.Flags&syntax.NonGreedy != 0 {
		suffix += "?"
	}
	return operand + suffix
}

func patternRune(r rune, inClass bool) string {
	if r < 0x20 || r == 0x7f || r == 0x2028 || r == 0x2029 {
		return fmt.Sprintf(`\u%04x`, r)
	}
	if strings.ContainsRune(`\.^$|?*+()[]{}`, r) || inClass && r == '-' {
		return "\\" + string(r)
	}
	return string(r)
}

func portablePatternTree(pattern *syntax.Regexp) error {
	if pattern.Flags&syntax.FoldCase != 0 || pattern.Op == syntax.OpBeginLine || pattern.Op == syntax.OpEndLine || pattern.Name != "" {
		return fmt.Errorf("flags and named captures are not in the portable pattern subset")
	}
	if pattern.Op == syntax.OpAnyChar || pattern.Op == syntax.OpAnyCharNotNL {
		return fmt.Errorf("wildcards have different newline and Unicode semantics in Go and ECMA-262; use an explicit character class")
	}
	for _, r := range pattern.Rune {
		if r > 0xffff {
			return fmt.Errorf("character sets containing non-BMP code points are not in the portable pattern subset")
		}
	}
	if pattern.Op == syntax.OpCharClass {
		for i := 0; i < len(pattern.Rune); i += 2 {
			if pattern.Rune[i] <= 0xdfff && pattern.Rune[i+1] >= 0xd800 {
				return fmt.Errorf("character sets containing UTF-16 surrogates are not in the portable pattern subset")
			}
		}
	}
	for _, child := range pattern.Sub {
		if err := portablePatternTree(child); err != nil {
			return err
		}
	}
	return nil
}
