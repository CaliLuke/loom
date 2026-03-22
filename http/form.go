package http

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

type (
	// FormValuesMarshaler is implemented by generated types that need custom
	// application/x-www-form-urlencoded encoding semantics.
	FormValuesMarshaler interface {
		// MarshalFormValues appends the encoded representation of the receiver to
		// the given values using the given field prefix.
		MarshalFormValues(values url.Values, prefix string) error
	}

	// FormValuesUnmarshaler is implemented by generated types that need custom
	// application/x-www-form-urlencoded decoding semantics.
	FormValuesUnmarshaler interface {
		// UnmarshalFormValues decodes the receiver from the given values using the
		// given field prefix.
		UnmarshalFormValues(values url.Values, prefix string) error
	}
)

var (
	formValuesMarshalerType   = reflect.TypeOf((*FormValuesMarshaler)(nil)).Elem()
	formValuesUnmarshalerType = reflect.TypeOf((*FormValuesUnmarshaler)(nil)).Elem()
)

// FormChildKey returns the fully qualified form key for name under prefix.
func FormChildKey(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "[" + name + "]"
}

// EncodeFormValues encodes v as application/x-www-form-urlencoded values.
func EncodeFormValues(v any) (url.Values, error) {
	values := url.Values{}
	if _, err := EncodeFormValue(values, "", v); err != nil {
		return nil, err
	}
	return values, nil
}

// EncodeFormValue appends the application/x-www-form-urlencoded representation
// of v to values using prefix.
func EncodeFormValue(values url.Values, prefix string, v any) (bool, error) {
	if v == nil {
		return false, nil
	}
	return encodeFormValue(values, prefix, reflect.ValueOf(v))
}

// DecodeFormValues decodes application/x-www-form-urlencoded values into
// target.
func DecodeFormValues(values url.Values, target any) error {
	_, err := DecodeFormValue(values, "", target)
	return err
}

// DecodeFormValue decodes application/x-www-form-urlencoded values into target
// using prefix.
func DecodeFormValue(values url.Values, prefix string, target any) (bool, error) {
	if target == nil {
		return false, fmt.Errorf("form decode target cannot be nil")
	}
	rv := reflect.ValueOf(target)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return false, fmt.Errorf("form decode target must be a non-nil pointer")
	}
	return decodeFormValue(values, prefix, rv)
}

// SetFormRequest encodes body as application/x-www-form-urlencoded and stores
// it in req.
func SetFormRequest(req *http.Request, body any) error {
	values, err := EncodeFormValues(body)
	if err != nil {
		return err
	}
	encoded := values.Encode()
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Body = io.NopCloser(strings.NewReader(encoded))
	req.ContentLength = int64(len(encoded))
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader([]byte(encoded))), nil
	}
	return nil
}

func encodeFormValue(values url.Values, prefix string, v reflect.Value) (bool, error) {
	normalized, handled, err := normalizeFormEncodeValue(values, prefix, v)
	if err != nil {
		return false, err
	}
	if handled {
		return true, nil
	}
	if !normalized.IsValid() {
		return false, nil
	}

	switch normalized.Kind() {
	case reflect.Struct:
		return encodeStructFields(values, prefix, normalized)
	case reflect.Map:
		return encodeMapEntries(values, prefix, normalized)
	case reflect.Slice, reflect.Array:
		return encodeSequenceValue(values, prefix, normalized)
	default:
		return encodeScalarFormValue(values, prefix, normalized)
	}
}

func decodeFormValue(values url.Values, prefix string, target reflect.Value) (bool, error) {
	if target.Kind() != reflect.Pointer || target.IsNil() {
		return false, fmt.Errorf("form decode target must be a non-nil pointer")
	}
	if !hasFormPrefix(values, prefix) {
		return false, nil
	}
	if target.Type().Implements(formValuesUnmarshalerType) {
		return true, target.Interface().(FormValuesUnmarshaler).UnmarshalFormValues(values, prefix)
	}

	elem := target.Elem()
	switch elem.Kind() {
	case reflect.Interface:
		return false, fmt.Errorf("unsupported form interface target %s", elem.Type())
	case reflect.Pointer:
		return decodePointerTarget(values, prefix, elem)
	case reflect.Struct:
		return decodeStructFields(values, prefix, elem)
	case reflect.Map:
		return decodeFormMap(values, prefix, elem)
	case reflect.Slice:
		return decodeSliceValue(values, prefix, elem)
	default:
		raw := values.Get(prefix)
		if err := setScalarValue(elem, raw); err != nil {
			return false, err
		}
		return true, nil
	}
}

func normalizeFormEncodeValue(values url.Values, prefix string, v reflect.Value) (reflect.Value, bool, error) {
	if !v.IsValid() {
		return reflect.Value{}, false, nil
	}
	for v.Kind() == reflect.Interface {
		if v.IsNil() {
			return reflect.Value{}, false, nil
		}
		v = v.Elem()
	}
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return reflect.Value{}, false, nil
		}
		if v.Type().Implements(formValuesMarshalerType) {
			return reflect.Value{}, true, v.Interface().(FormValuesMarshaler).MarshalFormValues(values, prefix)
		}
		return normalizeFormEncodeValue(values, prefix, v.Elem())
	}
	if v.Type().Implements(formValuesMarshalerType) {
		return reflect.Value{}, true, v.Interface().(FormValuesMarshaler).MarshalFormValues(values, prefix)
	}
	if v.CanAddr() && v.Addr().Type().Implements(formValuesMarshalerType) {
		return reflect.Value{}, true, v.Addr().Interface().(FormValuesMarshaler).MarshalFormValues(values, prefix)
	}
	return v, false, nil
}

func encodeStructFields(values url.Values, prefix string, v reflect.Value) (bool, error) {
	seen := false
	for i := range v.NumField() {
		field := v.Type().Field(i)
		if field.PkgPath != "" {
			continue
		}
		name, ok := formFieldName(field)
		if !ok {
			continue
		}
		fieldSeen, err := encodeFormValue(values, FormChildKey(prefix, name), v.Field(i))
		if err != nil {
			return seen, err
		}
		seen = seen || fieldSeen
	}
	return seen, nil
}

func encodeMapEntries(values url.Values, prefix string, v reflect.Value) (bool, error) {
	if v.IsNil() {
		return false, nil
	}
	seen := false
	iter := v.MapRange()
	for iter.Next() {
		key, err := scalarToString(iter.Key())
		if err != nil {
			return seen, err
		}
		fieldSeen, err := encodeFormValue(values, FormChildKey(prefix, key), iter.Value())
		if err != nil {
			return seen, err
		}
		seen = seen || fieldSeen
	}
	return seen, nil
}

func encodeSequenceValue(values url.Values, prefix string, v reflect.Value) (bool, error) {
	if v.Kind() == reflect.Slice && v.IsNil() {
		return false, nil
	}
	if v.Type().Elem().Kind() == reflect.Uint8 {
		if prefix == "" {
			return false, fmt.Errorf("form encoding requires a field name for bytes values")
		}
		values.Add(prefix, string(v.Bytes()))
		return true, nil
	}
	if !isScalarType(v.Type().Elem()) {
		return false, fmt.Errorf("unsupported form slice element type %s", v.Type().Elem())
	}
	for i := range v.Len() {
		raw, err := scalarToString(v.Index(i))
		if err != nil {
			return false, err
		}
		values.Add(prefix, raw)
	}
	return v.Len() > 0, nil
}

func encodeScalarFormValue(values url.Values, prefix string, v reflect.Value) (bool, error) {
	if prefix == "" {
		return false, fmt.Errorf("form encoding requires a field name for %s", v.Type())
	}
	raw, err := scalarToString(v)
	if err != nil {
		return false, err
	}
	values.Add(prefix, raw)
	return true, nil
}

func decodePointerTarget(values url.Values, prefix string, elem reflect.Value) (bool, error) {
	child := reflect.New(elem.Type().Elem())
	seen, err := decodeFormValue(values, prefix, child)
	if !seen || err != nil {
		return seen, err
	}
	elem.Set(child)
	return true, nil
}

func decodeStructFields(values url.Values, prefix string, elem reflect.Value) (bool, error) {
	seen := false
	for i := range elem.NumField() {
		field := elem.Type().Field(i)
		if field.PkgPath != "" {
			continue
		}
		name, ok := formFieldName(field)
		if !ok {
			continue
		}
		fieldSeen, err := decodeIntoField(values, FormChildKey(prefix, name), elem.Field(i))
		if err != nil {
			return seen, err
		}
		seen = seen || fieldSeen
	}
	return seen, nil
}

func decodeSliceValue(values url.Values, prefix string, elem reflect.Value) (bool, error) {
	if elem.Type().Elem().Kind() == reflect.Uint8 {
		raw := values.Get(prefix)
		elem.SetBytes([]byte(raw))
		return true, nil
	}
	raws := values[prefix]
	if !isScalarType(elem.Type().Elem()) {
		return false, fmt.Errorf("unsupported form slice element type %s", elem.Type().Elem())
	}
	slice := reflect.MakeSlice(elem.Type(), len(raws), len(raws))
	for i, raw := range raws {
		if err := setScalarValue(slice.Index(i), raw); err != nil {
			return false, err
		}
	}
	elem.Set(slice)
	return len(raws) > 0, nil
}

func decodeIntoField(values url.Values, prefix string, field reflect.Value) (bool, error) {
	if !field.CanSet() {
		return false, nil
	}
	if field.Kind() == reflect.Pointer {
		child := reflect.New(field.Type().Elem())
		seen, err := decodeFormValue(values, prefix, child)
		if !seen || err != nil {
			return seen, err
		}
		field.Set(child)
		return true, nil
	}
	return decodeFormValue(values, prefix, field.Addr())
}

func decodeFormMap(values url.Values, prefix string, target reflect.Value) (bool, error) {
	if !target.CanSet() {
		return false, fmt.Errorf("cannot set form map target %s", target.Type())
	}
	entries := make(map[string]url.Values)
	for key, vals := range values {
		child, nested, ok := parseFormChildKey(prefix, key)
		if !ok {
			continue
		}
		if entries[child] == nil {
			entries[child] = url.Values{}
		}
		if nested != "" {
			entries[child][nested] = append(entries[child][nested], vals...)
		} else {
			entries[child][child] = append(entries[child][child], vals...)
		}
	}
	if len(entries) == 0 {
		return false, nil
	}
	m := reflect.MakeMap(target.Type())
	for child, childValues := range entries {
		key := reflect.New(target.Type().Key()).Elem()
		if err := setScalarValue(key, child); err != nil {
			return false, err
		}
		elem := reflect.New(target.Type().Elem()).Elem()
		if isScalarType(target.Type().Elem()) {
			if err := setScalarValue(elem, firstFormValue(childValues, child)); err != nil {
				return false, err
			}
		} else {
			seen, err := decodeIntoField(childValues, child, elem)
			if err != nil {
				return false, err
			}
			if !seen {
				return false, fmt.Errorf("missing form value for %s", child)
			}
		}
		m.SetMapIndex(key, elem)
	}
	target.Set(m)
	return true, nil
}

func formFieldName(field reflect.StructField) (string, bool) {
	tag := field.Tag.Get("form")
	if tag == "" || tag == "-" {
		return "", false
	}
	name, _, _ := strings.Cut(tag, ",")
	if name == "" || name == "-" {
		return "", false
	}
	return name, true
}

func hasFormPrefix(values url.Values, prefix string) bool {
	if prefix == "" {
		return len(values) > 0
	}
	if _, ok := values[prefix]; ok {
		return true
	}
	pfx := prefix + "["
	for key := range values {
		if strings.HasPrefix(key, pfx) {
			return true
		}
	}
	return false
}

func parseFormChildKey(prefix, key string) (string, string, bool) {
	if prefix == "" {
		if idx := strings.IndexByte(key, '['); idx >= 0 {
			return key[:idx], key[idx:], true
		}
		return key, "", true
	}
	pfx := prefix + "["
	if !strings.HasPrefix(key, pfx) {
		return "", "", false
	}
	rest := strings.TrimPrefix(key, pfx)
	idx := strings.IndexByte(rest, ']')
	if idx < 0 {
		return "", "", false
	}
	child := rest[:idx]
	if idx == len(rest)-1 {
		return child, "", true
	}
	return child, rest[idx+1:], true
}

func firstFormValue(values url.Values, key string) string {
	if vals := values[key]; len(vals) > 0 {
		return vals[0]
	}
	for _, vals := range values {
		if len(vals) > 0 {
			return vals[0]
		}
	}
	return ""
}

func isScalarType(t reflect.Type) bool {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.String, reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64:
		return true
	case reflect.Slice:
		return t.Elem().Kind() == reflect.Uint8
	default:
		return false
	}
}

func scalarToString(v reflect.Value) (string, error) {
	for v.Kind() == reflect.Interface || v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return "", nil
		}
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.String:
		return v.String(), nil
	case reflect.Bool:
		return strconv.FormatBool(v.Bool()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.FormatUint(v.Uint(), 10), nil
	case reflect.Float32:
		return strconv.FormatFloat(v.Float(), 'f', -1, 32), nil
	case reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'f', -1, 64), nil
	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			return string(v.Bytes()), nil
		}
	}
	return "", fmt.Errorf("unsupported scalar form type %s", v.Type())
}

func setScalarValue(target reflect.Value, raw string) error {
	for target.Kind() == reflect.Interface {
		return fmt.Errorf("unsupported form interface target %s", target.Type())
	}
	switch target.Kind() {
	case reflect.String:
		target.SetString(raw)
		return nil
	case reflect.Bool:
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return err
		}
		target.SetBool(v)
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v, err := strconv.ParseInt(raw, 10, target.Type().Bits())
		if err != nil {
			return err
		}
		target.SetInt(v)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		v, err := strconv.ParseUint(raw, 10, target.Type().Bits())
		if err != nil {
			return err
		}
		target.SetUint(v)
		return nil
	case reflect.Float32, reflect.Float64:
		v, err := strconv.ParseFloat(raw, target.Type().Bits())
		if err != nil {
			return err
		}
		target.SetFloat(v)
		return nil
	case reflect.Slice:
		if target.Type().Elem().Kind() == reflect.Uint8 {
			target.SetBytes([]byte(raw))
			return nil
		}
	}
	return fmt.Errorf("unsupported form target type %s", target.Type())
}
