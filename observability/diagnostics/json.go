package diagnostics

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"strings"
	"unicode"
)

func (p *Policy) jsonBody(data []byte) []byte {
	value := jsontext.Value(data)
	if !value.IsValid() {
		return []byte("[invalid JSON body omitted]")
	}
	redacted, err := p.jsonValue(value)
	if err != nil {
		return []byte("[JSON body redaction failed]")
	}
	return redacted
}

func (p *Policy) jsonValue(value jsontext.Value) (jsontext.Value, error) {
	switch value.Kind() {
	case '{':
		var object map[string]jsontext.Value
		if err := json.Unmarshal(value, &object); err != nil {
			return nil, err
		}
		for name, field := range object {
			if sensitiveName(name) || p.json[foldJSONName(name)] {
				object[name] = jsontext.Value(`"[REDACTED]"`)
				continue
			}
			redacted, err := p.jsonValue(field)
			if err != nil {
				return nil, err
			}
			object[name] = redacted
		}
		return json.Marshal(object, json.Deterministic(true))
	case '[':
		var items []jsontext.Value
		if err := json.Unmarshal(value, &items); err != nil {
			return nil, err
		}
		for index, item := range items {
			redacted, err := p.jsonValue(item)
			if err != nil {
				return nil, err
			}
			items[index] = redacted
		}
		return json.Marshal(items)
	default:
		return value.Clone(), nil
	}
}

// foldJSONName includes every alias accepted by case:ignore JSON fields. Policy
// metadata need not retain codec options: even strict fields are masked under
// the broader matching rule so an alias never weakens credential protection.
func foldJSONName(name string) string {
	return strings.Map(func(char rune) rune {
		if char == '-' || char == '_' {
			return -1
		}
		return foldRune(char)
	}, name)
}

// foldRune returns the smallest member of a Unicode simple-fold cycle.
func foldRune(char rune) rune {
	minimum := char
	for next := unicode.SimpleFold(char); next != char; next = unicode.SimpleFold(next) {
		if next < minimum {
			minimum = next
		}
	}
	return minimum
}
