package main

import (
	"bytes"
	"encoding/json/v2"

	"github.com/CaliLuke/loom/internal/exactnum"
)

// unmarshalComparisonLiteral retains exact numbers inside interface-valued
// defaults, enums, examples and extensions as well as typed schema bounds.
// Objects remain structured so key order cannot affect overlay equality.
func unmarshalComparisonLiteral(data []byte, target *any) error {
	data = bytes.TrimSpace(data)
	options := json.WithUnmarshalers(json.UnmarshalFunc(unmarshalComparisonLiteral))
	switch data[0] {
	case '{':
		var value map[string]any
		if err := json.Unmarshal(data, &value, options); err != nil {
			return err
		}
		*target = value
	case '[':
		var value []any
		if err := json.Unmarshal(data, &value, options); err != nil {
			return err
		}
		*target = value
	case '"':
		var value string
		if err := json.Unmarshal(data, &value); err != nil {
			return err
		}
		*target = value
	case 't', 'f':
		var value bool
		if err := json.Unmarshal(data, &value); err != nil {
			return err
		}
		*target = value
	case 'n':
		*target = nil
	default:
		value, err := exactnum.ParseJSON(string(data))
		if err != nil {
			return err
		}
		*target = value
	}
	return nil
}
