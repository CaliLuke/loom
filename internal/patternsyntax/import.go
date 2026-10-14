package patternsyntax

import (
	"fmt"
	"regexp"
	"regexp/syntax"
	"strconv"
	"strings"
	"unicode/utf8"
)

// ecmaParser admits a deliberately bounded grammar before Go can interpret
// ECMAScript spelling with different semantics. The Go parser validates the
// resulting syntax, and the common portable tree check validates its domain.
type ecmaParser struct {
	source string
	index  int
	depth  int
}

// FromECMA translates a portable, unflagged ECMAScript pattern to Go regexp
// syntax, preserving acceptance over Unicode scalar strings. Unsupported
// constructs return an error instead of approximating their meaning.
func FromECMA(pattern string) (string, error) {
	p := ecmaParser{source: pattern}
	translated, err := p.expression()
	if err != nil {
		return "", err
	}
	if p.index != len(pattern) {
		return "", fmt.Errorf("unmatched closing group at byte %d", p.index)
	}
	parsed, err := syntax.Parse(translated, syntax.Perl)
	if err != nil {
		return "", fmt.Errorf("pattern is outside the supported Go grammar: %w", err)
	}
	if err := portablePatternTree(parsed); err != nil {
		return "", err
	}
	return translated, nil
}

func (p *ecmaParser) expression() (string, error) {
	var out strings.Builder
	for p.index < len(p.source) && p.source[p.index] != ')' {
		if p.source[p.index] == '|' {
			out.WriteByte('|')
			p.index++
			continue
		}
		atom, assertion, err := p.atom()
		if err != nil {
			return "", err
		}
		out.WriteString(atom)
		if p.index < len(p.source) && strings.ContainsRune("*+?{", rune(p.source[p.index])) {
			if assertion {
				return "", fmt.Errorf("directly quantified assertions are not in the portable pattern subset")
			}
			quantifier, err := p.quantifier()
			if err != nil {
				return "", err
			}
			out.WriteString(quantifier)
		}
	}
	return out.String(), nil
}

func (p *ecmaParser) atom() (string, bool, error) {
	ch := p.source[p.index]
	p.index++
	switch ch {
	case '(':
		p.depth++
		if p.depth > 256 {
			return "", false, fmt.Errorf("pattern nesting exceeds the portable limit of 256")
		}
		prefix := "("
		if p.index < len(p.source) && p.source[p.index] == '?' {
			if !strings.HasPrefix(p.source[p.index:], "?:") {
				return "", false, fmt.Errorf("lookaround, flags and named captures are not in the portable pattern subset")
			}
			prefix = "(?:"
			p.index += 2
		}
		inner, err := p.expression()
		if err != nil {
			return "", false, err
		}
		if p.index == len(p.source) || p.source[p.index] != ')' {
			return "", false, fmt.Errorf("unterminated group")
		}
		p.index++
		p.depth--
		return prefix + inner + ")", false, nil
	case '[':
		class, err := p.class()
		return class, false, err
	case '\\':
		value, _, assertion, err := p.escape(false)
		return value, assertion, err
	case '^', '$':
		return string(ch), true, nil
	case '.':
		return "", false, fmt.Errorf("wildcards have different newline and Unicode semantics; use an explicit positive character class")
	case '*', '+', '?', '{', '}', ']':
		return "", false, fmt.Errorf("unexpected pattern metacharacter %q at byte %d", ch, p.index-1)
	default:
		p.index--
		r, err := p.literal()
		return goPatternLiteral(r, false), false, err
	}
}

func (p *ecmaParser) class() (string, error) {
	if p.index == len(p.source) || p.source[p.index] == ']' || p.source[p.index] == '^' {
		return "", fmt.Errorf("empty and complemented character classes are not in the portable pattern subset")
	}
	var out strings.Builder
	out.WriteByte('[')
	for p.index < len(p.source) && p.source[p.index] != ']' {
		first, single, err := p.classItem()
		if err != nil {
			return "", err
		}
		out.WriteString(first)
		if p.index+1 < len(p.source) && p.source[p.index] == '-' && p.source[p.index+1] != ']' {
			p.index++
			last, lastSingle, err := p.classItem()
			if err != nil {
				return "", err
			}
			if !single || !lastSingle {
				return "", fmt.Errorf("character class range endpoints must be single literals")
			}
			out.WriteByte('-')
			out.WriteString(last)
		}
	}
	if p.index == len(p.source) {
		return "", fmt.Errorf("unterminated character class")
	}
	p.index++
	out.WriteByte(']')
	return out.String(), nil
}

func (p *ecmaParser) classItem() (string, bool, error) {
	if p.index >= len(p.source) {
		return "", false, fmt.Errorf("unterminated character class")
	}
	if p.source[p.index] == '\\' {
		p.index++
		value, single, _, err := p.escape(true)
		return value, single, err
	}
	r, err := p.literal()
	return goPatternLiteral(r, true), true, err
}

func (p *ecmaParser) literal() (rune, error) {
	r, size := utf8.DecodeRuneInString(p.source[p.index:])
	if r == utf8.RuneError && size == 1 || r > 0xffff || r >= 0xd800 && r <= 0xdfff {
		return 0, fmt.Errorf("only BMP Unicode scalar literals are in the portable pattern subset")
	}
	p.index += size
	return r, nil
}

func (p *ecmaParser) escape(inClass bool) (string, bool, bool, error) {
	if p.index == len(p.source) {
		return "", false, false, fmt.Errorf("trailing pattern escape")
	}
	ch := p.source[p.index]
	p.index++
	var class string
	switch ch {
	case 'd':
		class = "0-9"
	case 'w':
		class = "0-9A-Z_a-z"
	case 's':
		// ECMAScript WhiteSpace and LineTerminator, without flags.
		class = "\\x09-\\x0d    -     　\ufeff"
	case 'b':
		if inClass {
			return `\x08`, true, false, nil
		}
		return `\b`, false, true, nil
	case 'B':
		if !inClass {
			return `\B`, false, true, nil
		}
	case 'x', 'u':
		digits := 2
		if ch == 'u' {
			digits = 4
		}
		if p.index+digits > len(p.source) {
			return "", false, false, fmt.Errorf("incomplete hexadecimal escape")
		}
		value, err := strconv.ParseUint(p.source[p.index:p.index+digits], 16, 16)
		if err != nil || value >= 0xd800 && value <= 0xdfff {
			return "", false, false, fmt.Errorf("hexadecimal escapes must denote a BMP Unicode scalar")
		}
		p.index += digits
		return goPatternLiteral(rune(value), inClass), true, false, nil
	case 'f', 'n', 'r', 't', 'v':
		return "\\" + string(ch), true, false, nil
	default:
		if strings.ContainsRune(`\.^$|?*+()[]{}-/`, rune(ch)) {
			return goPatternLiteral(rune(ch), inClass), true, false, nil
		}
	}
	if class != "" {
		if !inClass {
			class = "[" + class + "]"
		}
		return class, false, false, nil
	}
	return "", false, false, fmt.Errorf("escape \\%c is not in the portable pattern subset", ch)
}

func (p *ecmaParser) quantifier() (string, error) {
	start := p.index
	if p.source[p.index] == '{' {
		p.index++
		if err := p.count(); err != nil {
			return "", err
		}
		if p.index < len(p.source) && p.source[p.index] == ',' {
			p.index++
			if p.index < len(p.source) && p.source[p.index] != '}' {
				if err := p.count(); err != nil {
					return "", err
				}
			}
		}
		if p.index == len(p.source) || p.source[p.index] != '}' {
			return "", fmt.Errorf("invalid repetition bound")
		}
	}
	p.index++
	if p.index < len(p.source) && p.source[p.index] == '?' {
		p.index++
	}
	return p.source[start:p.index], nil
}

func (p *ecmaParser) count() error {
	start := p.index
	for p.index < len(p.source) && p.source[p.index] >= '0' && p.source[p.index] <= '9' {
		p.index++
	}
	if p.index == start || p.index-start > 1 && p.source[start] == '0' {
		return fmt.Errorf("repetition counts require decimal digits without leading zeroes")
	}
	return nil
}

func goPatternLiteral(r rune, inClass bool) string {
	if r < 0x20 || r == 0x7f {
		return fmt.Sprintf(`\x%02x`, r)
	}
	if inClass && r == '-' {
		return `\-`
	}
	return regexp.QuoteMeta(string(r))
}
