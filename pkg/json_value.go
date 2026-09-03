package loom

import (
	"bytes"
	stdjson "encoding/json"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"io"
	"math/big"
)

// JSONValue contains one unparsed JSON value. Generated Any fields use
// JSONValue so decoding and re-encoding preserve number spellings exactly.
type JSONValue = jsontext.Value

// JSONValueFromString returns a JSON string value. Invalid UTF-8 is replaced
// with the Unicode replacement character.
func JSONValueFromString(value string) JSONValue {
	encoded, err := json.Marshal(value, jsontext.AllowInvalidUTF8(true))
	if err != nil {
		panic("encode JSON string: " + err.Error())
	}
	return encoded
}

// JSONValueFrom returns the JSON encoding of value.
func JSONValueFrom(value any) (JSONValue, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return encoded, nil
}

// MustJSONValueFrom returns the JSON encoding of value and panics if value is
// not representable as JSON. Generated code uses it only for validated design
// defaults.
func MustJSONValueFrom(value any) JSONValue {
	encoded, err := JSONValueFrom(value)
	if err != nil {
		panic("encode JSON value: " + err.Error())
	}
	return encoded
}

// JSONValueString returns the decoded string or the raw JSON text.
func JSONValueString(value JSONValue) string {
	var decoded string
	if err := json.Unmarshal(value, &decoded); err == nil {
		return decoded
	}
	return string(value)
}

// JSONValueEqual reports whether value and expected are semantically equal JSON.
func JSONValueEqual(value JSONValue, expected any) bool {
	encoded, err := json.Marshal(expected)
	if err != nil {
		return false
	}
	actualValue, err := decodeJSONValue(value)
	if err != nil {
		return false
	}
	expectedValue, err := decodeJSONValue(encoded)
	return err == nil && equalJSONValue(actualValue, expectedValue)
}

func decodeJSONValue(encoded []byte) (any, error) {
	if !JSONValue(encoded).IsValid() {
		return nil, errors.New("invalid JSON value")
	}
	decoder := stdjson.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, err
	}
	return value, nil
}

func equalJSONValue(left, right any) bool {
	switch left := left.(type) {
	case nil:
		return right == nil
	case bool:
		right, ok := right.(bool)
		return ok && left == right
	case string:
		right, ok := right.(string)
		return ok && left == right
	case stdjson.Number:
		right, ok := right.(stdjson.Number)
		if !ok {
			return false
		}
		leftNumber, leftOK := new(big.Rat).SetString(string(left))
		rightNumber, rightOK := new(big.Rat).SetString(string(right))
		return leftOK && rightOK && leftNumber.Cmp(rightNumber) == 0
	case []any:
		right, ok := right.([]any)
		if !ok || len(left) != len(right) {
			return false
		}
		for i := range left {
			if !equalJSONValue(left[i], right[i]) {
				return false
			}
		}
		return true
	case map[string]any:
		right, ok := right.(map[string]any)
		if !ok || len(left) != len(right) {
			return false
		}
		for key, leftValue := range left {
			rightValue, ok := right[key]
			if !ok || !equalJSONValue(leftValue, rightValue) {
				return false
			}
		}
		return true
	default:
		return false
	}
}
