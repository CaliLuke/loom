package jsonkey

import (
	"encoding"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"errors"
	"fmt"
	"reflect"
	"strconv"
)

// BooleanKeys extends JSON encoding and decoding with canonical boolean map
// keys, named "true" and "false". Boolean values and types with custom JSON
// or text representations retain the JSON library's behavior.
var BooleanKeys = json.JoinOptions(
	json.WithMarshalers(json.MarshalToFunc[any](marshalBooleanKey)),
	json.WithUnmarshalers(json.UnmarshalFromFunc[any](unmarshalBooleanKey)),
)

// HasCustomEncoding reports whether JSON serialization owns the type's
// representation. JSON v2 also checks pointer methods on addressable copies.
func HasCustomEncoding(valueType reflect.Type) bool {
	if valueType == nil {
		return false
	}
	for _, candidate := range []reflect.Type{valueType, reflect.PointerTo(valueType)} {
		if candidate.Implements(reflect.TypeFor[json.MarshalerTo]()) ||
			candidate.Implements(reflect.TypeFor[json.Marshaler]()) ||
			candidate.Implements(reflect.TypeFor[encoding.TextAppender]()) ||
			candidate.Implements(reflect.TypeFor[encoding.TextMarshaler]()) {
			return true
		}
	}
	return false
}

func marshalBooleanKey(encoder *jsontext.Encoder, value any) error {
	actual := reflect.ValueOf(value)
	for actual.IsValid() && (actual.Kind() == reflect.Pointer || actual.Kind() == reflect.Interface) {
		actual = actual.Elem()
	}
	kind, index := encoder.StackIndex(encoder.StackDepth())
	if !actual.IsValid() || actual.Kind() != reflect.Bool || kind != jsontext.KindBeginObject || index%2 != 0 || HasCustomEncoding(actual.Type()) {
		return errors.ErrUnsupported
	}
	return encoder.WriteToken(jsontext.String(strconv.FormatBool(actual.Bool())))
}

func unmarshalBooleanKey(decoder *jsontext.Decoder, value any) error {
	actual := reflect.ValueOf(value).Elem()
	kind, index := decoder.StackIndex(decoder.StackDepth())
	if actual.Kind() != reflect.Bool || kind != jsontext.KindBeginObject || index%2 != 0 || hasCustomDecoding(actual.Type()) {
		return errors.ErrUnsupported
	}
	token, err := decoder.ReadToken()
	if err != nil {
		return err
	}
	if token.Kind() != jsontext.KindString || token.String() != "true" && token.String() != "false" {
		return fmt.Errorf("invalid boolean map key %q: expected true or false", token.String())
	}
	actual.SetBool(token.String() == "true")
	return nil
}

func hasCustomDecoding(valueType reflect.Type) bool {
	for _, candidate := range []reflect.Type{valueType, reflect.PointerTo(valueType)} {
		if candidate.Implements(reflect.TypeFor[json.UnmarshalerFrom]()) ||
			candidate.Implements(reflect.TypeFor[json.Unmarshaler]()) ||
			candidate.Implements(reflect.TypeFor[encoding.TextUnmarshaler]()) {
			return true
		}
	}
	return false
}
