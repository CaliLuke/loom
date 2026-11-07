package http

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"net/url"
	"reflect"
)

// MultipartObjectJSON projects one object alternative while retaining original
// physical part claims. Each view supplies its independently decoded value.
func MultipartObjectJSON(form *MultipartForm, fields []FormField) (FormProjection, bool, error) {
	return FormObjectJSON(multipartPairs(form), fields)
}

// MultipartJSONValue projects JSON part contents without URI decoding or typed
// conversion. Item application combines every part in source order into an array.
func MultipartJSONValue(form *MultipartForm, name string, application FieldApplication) (jsontext.Value, bool, error) {
	if err := validateFieldApplication(application); err != nil {
		return nil, false, err
	}
	values := multipartValues(form, name)
	if len(values) == 0 {
		return nil, true, nil
	}
	raw, err := jsonFieldValue(values, application)
	if err != nil || !raw.IsValid() {
		return nil, false, nil //nolint:nilerr // Malformed branch content is a schema mismatch, not a descriptor failure.
	}
	return raw, true, nil
}

// MultipartBinaryValue projects raw byte parts into their JSON schema spelling.
// One-value application rejects duplicate parts. Empty raw parts are present.
func MultipartBinaryValue(form *MultipartForm, name string, application FieldApplication) (jsontext.Value, bool, error) {
	if err := validateFieldApplication(application); err != nil {
		return nil, false, err
	}
	var values [][]byte
	if form != nil {
		for _, part := range form.Parts {
			if part.Name == name {
				values = append(values, append([]byte{}, part.Data...))
			}
		}
	}
	if len(values) == 0 {
		return nil, true, nil
	}
	var value any = values
	if application == FieldValue {
		if len(values) != 1 {
			return nil, false, nil
		}
		value = values[0]
	}
	raw, err := json.Marshal(value)
	return raw, err == nil, err
}

// DecodeMultipartJSON applies the framework JSON decoder to captured part content.
// Absence leaves target unchanged; null admission is independent of presence.
func DecodeMultipartJSON(form *MultipartForm, name string, target any, allowsNull bool, application FieldApplication) (bool, error) {
	values := make(url.Values)
	if parts := multipartValues(form, name); len(parts) > 0 {
		values[name] = parts
	}
	return DecodeJSONField(values, name, target, allowsNull, application)
}

// DecodeMultipartBinary decodes raw byte parts using the target's JSON byte carrier.
// A present empty part becomes nonnil empty bytes. Duplicate scalar parts reject.
func DecodeMultipartBinary(form *MultipartForm, name string, target any, application FieldApplication) (bool, error) {
	raw, matched, err := MultipartBinaryValue(form, name, application)
	if err != nil {
		return false, err
	}
	if !matched {
		return true, fmt.Errorf("multiple multipart files provided for field %s", name)
	}
	if raw == nil {
		return false, nil
	}
	return true, decodeNonNullableJSON(raw, target)
}

// AddJSON appends complete JSON parts with the captured media label and application.
// Encoding is atomic; failures leave the existing part set unchanged.
func (f *MultipartForm) AddJSON(name, media string, value any, application FieldApplication) (bool, error) {
	values := make(url.Values)
	present, err := EncodeJSONField(values, name, value, application)
	if err != nil {
		return false, err
	}
	for _, raw := range values[name] {
		f.Parts = append(f.Parts, MultipartPart{Name: name, ContentType: media, Data: []byte(raw)})
	}
	return present, nil
}

// AddBinary appends raw byte parts without base64 conversion. Item application
// removes exactly one array level. Nil carriers are absent; empty bytes are present.
func (f *MultipartForm) AddBinary(name, media string, value any, application FieldApplication) (bool, error) {
	if err := validateFieldApplication(application); err != nil {
		return false, err
	}
	v := reflect.ValueOf(value)
	for v.IsValid() && (v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface) {
		if v.IsNil() {
			return false, nil
		}
		v = v.Elem()
	}
	if !v.IsValid() || v.Kind() == reflect.Slice && v.IsNil() {
		return false, nil
	}
	items := []reflect.Value{v}
	if application == FieldItems {
		if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
			return false, fmt.Errorf("multipart binary item application requires an array")
		}
		items = nil
		for index := 0; index < v.Len(); index++ {
			items = append(items, v.Index(index))
		}
	}
	parts := make([]MultipartPart, 0, len(items))
	for _, item := range items {
		if item.Kind() != reflect.Slice || item.Type().Elem().Kind() != reflect.Uint8 {
			return false, fmt.Errorf("multipart binary field %q requires bytes", name)
		}
		parts = append(parts, MultipartPart{Name: name, ContentType: media, Data: append([]byte{}, item.Bytes()...)})
	}
	f.Parts = append(f.Parts, parts...)
	return len(parts) > 0, nil
}

func multipartValues(form *MultipartForm, name string) []string {
	var values []string
	if form != nil {
		for _, part := range form.Parts {
			if part.Name == name {
				values = append(values, string(part.Data))
			}
		}
	}
	return values
}
