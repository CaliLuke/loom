package loom

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"math/big"
	"strings"
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
	return err == nil && equalEncodedJSON(value, encoded)
}

func equalEncodedJSON(left, right jsontext.Value) bool {
	if !left.IsValid() || !right.IsValid() || left.Kind() != right.Kind() {
		return false
	}
	switch left.Kind() {
	case 'n':
		return true
	case 't', 'f':
		return strings.TrimSpace(string(left)) == strings.TrimSpace(string(right))
	case '"':
		var leftString, rightString string
		return json.Unmarshal(left, &leftString) == nil && json.Unmarshal(right, &rightString) == nil && leftString == rightString
	case '[':
		var leftValues, rightValues []jsontext.Value
		if json.Unmarshal(left, &leftValues) != nil || json.Unmarshal(right, &rightValues) != nil || len(leftValues) != len(rightValues) {
			return false
		}
		for index := range leftValues {
			if !equalEncodedJSON(leftValues[index], rightValues[index]) {
				return false
			}
		}
		return true
	case '{':
		var leftValues, rightValues map[string]jsontext.Value
		if json.Unmarshal(left, &leftValues) != nil || json.Unmarshal(right, &rightValues) != nil || len(leftValues) != len(rightValues) {
			return false
		}
		for key, leftValue := range leftValues {
			rightValue, ok := rightValues[key]
			if !ok || !equalEncodedJSON(leftValue, rightValue) {
				return false
			}
		}
		return true
	default:
		leftNumber, leftOK := new(big.Rat).SetString(strings.TrimSpace(string(left)))
		rightNumber, rightOK := new(big.Rat).SetString(strings.TrimSpace(string(right)))
		return leftOK && rightOK && leftNumber.Cmp(rightNumber) == 0
	}
}
