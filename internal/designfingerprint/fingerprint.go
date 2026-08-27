// Package designfingerprint computes deterministic fingerprints of evaluated designs.
package designfingerprint

import (
	"bytes"
	"crypto/sha256"
	"encoding"
	"encoding/hex"
	"fmt"
	"reflect"
	"sort"
	"strconv"

	"github.com/CaliLuke/loom/expr"
)

const formatVersion = "loom-design-v1"

var reflectType = reflect.TypeFor[reflect.Type]()

type (
	encoder struct {
		buffer bytes.Buffer
		active map[uintptr]string
	}

	mapEntry struct {
		key   []byte
		value []byte
	}
)

// Digest returns the deterministic SHA-256 digest of an evaluated design and
// the generation inputs that can change its output.
func Digest(root *expr.RootExpr, command, genpkg string, designVersion int) (string, error) {
	enc := &encoder{active: make(map[uintptr]string)}
	enc.writeString(formatVersion)
	enc.writeString(command)
	enc.writeString(genpkg)
	enc.writeString(strconv.Itoa(designVersion))
	if err := enc.encode(reflect.ValueOf(root), "design"); err != nil {
		return "", fmt.Errorf("encode evaluated design: %w", err)
	}
	sum := sha256.Sum256(enc.buffer.Bytes())
	return hex.EncodeToString(sum[:]), nil
}

func (e *encoder) encode(value reflect.Value, path string) error {
	if !value.IsValid() {
		e.writeString("invalid")
		return nil
	}

	typeName := value.Type().PkgPath() + ":" + value.Type().String()
	e.writeString(typeName)
	switch value.Kind() {
	case reflect.Interface:
		return e.encodeInterface(value, path)
	case reflect.Pointer:
		return e.encodePointer(value, path)
	case reflect.Struct:
		return e.encodeStruct(value, path)
	case reflect.Map:
		return e.encodeMap(value, path)
	case reflect.Slice, reflect.Array:
		return e.encodeSequence(value, path)
	default:
		return e.encodeScalar(value, path)
	}
}

func (e *encoder) encodeInterface(value reflect.Value, path string) error {
	if value.IsNil() {
		e.writeString("nil")
		return nil
	}
	if value.Type() == reflectType {
		typeValue := value.Interface().(reflect.Type)
		e.writeString(typeValue.PkgPath() + ":" + typeValue.String())
		return nil
	}
	return e.encode(value.Elem(), path+".interface")
}

func (e *encoder) encodePointer(value reflect.Value, path string) error {
	if value.IsNil() {
		e.writeString("nil")
		return nil
	}
	pointer := value.Pointer()
	if firstPath, exists := e.active[pointer]; exists {
		e.writeString("cycle:" + firstPath)
		return nil
	}
	e.active[pointer] = path
	defer delete(e.active, pointer)
	return e.encode(value.Elem(), path+".value")
}

func (e *encoder) encodeStruct(value reflect.Value, path string) error {
	if text, ok, err := marshaledText(value); ok {
		if err != nil {
			return err
		}
		e.writeBytes(text)
		return nil
	}
	for i := range value.NumField() {
		field := value.Type().Field(i)
		if !field.IsExported() || field.Name == "ExampleGenerator" {
			continue
		}
		e.writeString(field.Name)
		if err := e.encode(value.Field(i), path+"."+field.Name); err != nil {
			return err
		}
	}
	return nil
}

func (e *encoder) encodeMap(value reflect.Value, path string) error {
	if value.IsNil() {
		e.writeString("nil")
		return nil
	}
	entries := make([]mapEntry, 0, value.Len())
	iterator := value.MapRange()
	for iterator.Next() {
		keyEncoder := e.fork()
		if err := keyEncoder.encode(iterator.Key(), path+".key"); err != nil {
			return err
		}
		valueEncoder := e.fork()
		if err := valueEncoder.encode(iterator.Value(), path+".value"); err != nil {
			return err
		}
		entries = append(entries, mapEntry{
			key:   keyEncoder.buffer.Bytes(),
			value: valueEncoder.buffer.Bytes(),
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		if comparison := bytes.Compare(entries[i].key, entries[j].key); comparison != 0 {
			return comparison < 0
		}
		return bytes.Compare(entries[i].value, entries[j].value) < 0
	})
	for _, entry := range entries {
		e.writeBytes(entry.key)
		e.writeBytes(entry.value)
	}
	return nil
}

func (e *encoder) fork() *encoder {
	active := make(map[uintptr]string, len(e.active))
	for pointer, path := range e.active {
		active[pointer] = path
	}
	return &encoder{active: active}
}

func (e *encoder) encodeSequence(value reflect.Value, path string) error {
	if value.Kind() == reflect.Slice && value.IsNil() {
		e.writeString("nil")
		return nil
	}
	e.writeString(strconv.Itoa(value.Len()))
	for i := range value.Len() {
		if err := e.encode(value.Index(i), path+"["+strconv.Itoa(i)+"]"); err != nil {
			return err
		}
	}
	return nil
}

func (e *encoder) encodeScalar(value reflect.Value, path string) error {
	switch value.Kind() {
	case reflect.String:
		e.writeString(value.String())
	case reflect.Bool:
		e.writeString(strconv.FormatBool(value.Bool()))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		e.writeString(strconv.FormatInt(value.Int(), 10))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		e.writeString(strconv.FormatUint(value.Uint(), 10))
	case reflect.Float32, reflect.Float64:
		e.writeString(strconv.FormatFloat(value.Float(), 'g', -1, value.Type().Bits()))
	case reflect.Complex64, reflect.Complex128:
		complexValue := value.Complex()
		e.writeString(strconv.FormatFloat(real(complexValue), 'g', -1, value.Type().Bits()/2))
		e.writeString(strconv.FormatFloat(imag(complexValue), 'g', -1, value.Type().Bits()/2))
	case reflect.Func, reflect.Chan, reflect.UnsafePointer:
		e.writeString("runtime-only")
	default:
		return fmt.Errorf("unsupported %s at %s", value.Kind(), path)
	}
	return nil
}

func marshaledText(value reflect.Value) ([]byte, bool, error) {
	if value.CanAddr() && value.Addr().CanInterface() {
		if marshaler, ok := value.Addr().Interface().(encoding.TextMarshaler); ok {
			text, err := marshaler.MarshalText()
			return text, true, err
		}
	}
	if value.CanInterface() {
		if marshaler, ok := value.Interface().(encoding.TextMarshaler); ok {
			text, err := marshaler.MarshalText()
			return text, true, err
		}
	}
	return nil, false, nil
}

func (e *encoder) writeString(value string) {
	e.writeBytes([]byte(value))
}

func (e *encoder) writeBytes(value []byte) {
	e.buffer.WriteString(strconv.Itoa(len(value)))
	e.buffer.WriteByte(':')
	e.buffer.Write(value)
}
