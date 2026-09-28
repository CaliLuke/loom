package ir

import (
	"encoding/base64"
	"unicode/utf8"

	"github.com/CaliLuke/loom/expr"
)

// serializeBinaryMediaExamples turns JSON byte values into literal stream
// examples for raw binary bodies. Schema/dataValue examples retain the JSON
// representation; the media example describes the serialized body. Arbitrary
// binary data cannot be embedded as a UTF-8 JSON/YAML string, so omit that
// inline media example instead of corrupting its bytes.
func serializeBinaryMediaExamples(media *MediaType, attribute *expr.AttributeExpr) {
	datatype := attribute.Type
	for {
		userType, ok := datatype.(expr.UserType)
		if !ok {
			break
		}
		datatype = userType.Attribute().Type
	}
	if datatype != expr.Bytes {
		return
	}
	media.Example = binaryMediaExample(media.Example)
	for name, reference := range media.Examples {
		if reference.Value == nil || reference.Value.Value == nil {
			continue
		}
		value := binaryMediaExample(reference.Value.Value)
		if value == nil && reference.Value.DataValue == nil && reference.Value.SerializedValue == "" {
			delete(media.Examples, name)
			continue
		}
		reference.Value.Value = value
	}
}

func binaryMediaExample(value any) any {
	encoded, ok := value.(string)
	if !ok {
		return value
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || !utf8.Valid(decoded) {
		return nil
	}
	return string(decoded)
}
