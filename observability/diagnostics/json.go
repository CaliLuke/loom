package diagnostics

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
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
			if sensitiveName(name) || p.json[name] {
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
