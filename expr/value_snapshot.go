package expr

import (
	"encoding"
	"encoding/json/v2"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"
)

var errValueSourcePointerCycle = errors.New("plain pointer source contains a cycle")

type (
	// valueSourceSnapshot defers invalid authored data until its source is used.
	// Opaque custom values remain borrowed; no codec runs during source capture.
	valueSourceSnapshot struct {
		raw any
		err error
	}

	valueSnapshotVisit struct {
		typeOf  reflect.Type
		pointer uintptr
		length  int
	}
)

type valueMapEntry struct {
	key, value reflect.Value
}

type valueSnapshotState struct {
	eraseSelections bool
	active          map[valueSnapshotVisit]bool
	copies          map[valueSnapshotVisit]reflect.Value
	err             error
}

func snapshotValueSource(raw any) valueSourceSnapshot {
	state := valueSnapshotState{}
	return state.snapshot(raw)
}

func (state *valueSnapshotState) snapshot(raw any) valueSourceSnapshot {
	state.active = make(map[valueSnapshotVisit]bool)
	state.copies = make(map[valueSnapshotVisit]reflect.Value)
	value := state.copy(reflect.ValueOf(raw), "$")
	if !value.IsValid() {
		return valueSourceSnapshot{err: state.err}
	}
	return valueSourceSnapshot{raw: value.Interface(), err: state.err}
}

func (s *valueSnapshotState) copy(value reflect.Value, path string) reflect.Value {
	if !value.IsValid() {
		return value
	}
	if copied, done := s.copySpecialValue(value, path); done {
		return copied
	}
	if copied, done := s.copyTrackedValue(value, path); done {
		return copied
	}
	switch value.Kind() {
	case reflect.Array:
		return s.copyArray(value, path, valueSnapshotVisit{})
	case reflect.Struct:
		return s.copyStruct(value, path)
	default:
		// Scalars copy by value. Other opaque host values are not interpreted by
		// the builtin resolver. The graph copy retains cycles so traversal detects
		// them only after body-local type/length checks, as the checked model does.
		return value
	}
}

func (s *valueSnapshotState) copySpecialValue(value reflect.Value, path string) (reflect.Value, bool) {
	if selected, ok := value.Interface().(valueSelectedInput); ok {
		if s.eraseSelections {
			return s.copy(reflect.ValueOf(selected.payload), path), true
		}
		payload := s.copy(reflect.ValueOf(selected.payload), path)
		if payload.IsValid() {
			selected.payload = payload.Interface()
		}
		return reflect.ValueOf(selected), true
	}
	if value.Kind() == reflect.Interface {
		if value.IsNil() {
			return reflect.Zero(value.Type()), true
		}
		copied := reflect.New(value.Type()).Elem()
		s.copyInto(copied, value.Elem(), path)
		return copied, true
	}
	// Custom values stay borrowed. No user method runs during source capture.
	if valueHasCustomCodec(value.Type()) {
		return value, true
	}
	return reflect.Value{}, false
}

func (s *valueSnapshotState) copyTrackedValue(value reflect.Value, path string) (reflect.Value, bool) {
	visit, tracked := valueTrackedVisit(value)
	if !tracked {
		return reflect.Value{}, false
	}
	if value.IsNil() {
		return reflect.Zero(value.Type()), true
	}
	if prior, found := s.copies[visit]; found {
		if s.active[visit] && s.err == nil {
			if visit.typeOf.Kind() == reflect.Pointer {
				s.err = fmt.Errorf("%w at %s", errValueSourcePointerCycle, path)
			} else {
				s.err = fmt.Errorf("%s: cyclic value", path)
			}
		}
		return prior, true
	}
	s.active[visit] = true
	defer delete(s.active, visit)
	switch value.Kind() {
	case reflect.Map:
		return s.copyMap(value, path, visit), true
	case reflect.Slice:
		return s.copyArray(value, path, visit), true
	case reflect.Pointer:
		copied := reflect.New(value.Type().Elem())
		s.copies[visit] = copied
		s.copyInto(copied.Elem(), value.Elem(), path+"*")
		return copied, true
	default:
		panic("unreachable tracked value kind")
	}
}

func valueTrackedVisit(value reflect.Value) (valueSnapshotVisit, bool) {
	if !value.IsValid() || (value.Kind() != reflect.Map && value.Kind() != reflect.Slice && value.Kind() != reflect.Pointer) {
		return valueSnapshotVisit{}, false
	}
	length := 0
	if value.Kind() != reflect.Pointer {
		length = value.Len()
	}
	return valueSnapshotVisit{value.Type(), value.Pointer(), length}, true
}

func (s *valueSnapshotState) copyStruct(value reflect.Value, path string) reflect.Value {
	copied := reflect.New(value.Type()).Elem()
	copied.Set(value)
	for index := range value.NumField() {
		if value.Type().Field(index).IsExported() {
			s.copyInto(copied.Field(index), value.Field(index), path+"."+value.Type().Field(index).Name)
		}
	}
	return copied
}

func valueHasCustomCodec(typ reflect.Type) bool {
	for _, protocol := range []reflect.Type{reflect.TypeFor[json.MarshalerTo](), reflect.TypeFor[json.Marshaler](), reflect.TypeFor[encoding.TextAppender](), reflect.TypeFor[encoding.TextMarshaler]()} {
		if typ.Implements(protocol) || (typ.Kind() != reflect.Pointer && reflect.PointerTo(typ).Implements(protocol)) {
			return true
		}
	}
	return false
}

// Capture key/value pairs before ordering. Looking a key up again loses
// entries whose key is not reflexive, such as NaN or a struct containing NaN.
func valueSortedMapEntries(value reflect.Value) []valueMapEntry {
	entries := make([]valueMapEntry, 0, value.Len())
	iterator := value.MapRange()
	for iterator.Next() {
		entries = append(entries, valueMapEntry{key: iterator.Key(), value: iterator.Value()})
	}
	sort.SliceStable(entries, func(i, j int) bool { return valueKeyOrder(entries[i].key) < valueKeyOrder(entries[j].key) })
	return entries
}

func valueKeyOrder(value reflect.Value) string {
	if value.Kind() == reflect.Interface && !value.IsNil() {
		return valueKeyOrder(value.Elem())
	}
	prefix := value.Type().String() + ":"
	switch value.Kind() {
	case reflect.String:
		return prefix + strconv.Quote(value.String())
	case reflect.Bool:
		return prefix + strconv.FormatBool(value.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return prefix + strconv.FormatInt(value.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return prefix + strconv.FormatUint(value.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return prefix + strconv.FormatFloat(value.Float(), 'g', -1, value.Type().Bits())
	default:
		return prefix
	}
}

func (s *valueSnapshotState) copyMap(value reflect.Value, path string, visit valueSnapshotVisit) reflect.Value {
	copied := reflect.MakeMapWithSize(value.Type(), value.Len())
	s.copies[visit] = copied
	for _, entry := range valueSortedMapEntries(value) {
		child := reflect.New(value.Type().Elem()).Elem()
		s.copyInto(child, entry.value, path+"["+valueKeyOrder(entry.key)+"]")
		copied.SetMapIndex(entry.key, child)
	}
	return copied
}

func (s *valueSnapshotState) copyArray(value reflect.Value, path string, visit valueSnapshotVisit) reflect.Value {
	var copied reflect.Value
	if value.Kind() == reflect.Slice {
		copied = reflect.MakeSlice(value.Type(), value.Len(), value.Len())
		s.copies[visit] = copied
	} else {
		copied = reflect.New(value.Type()).Elem()
	}
	for index := range value.Len() {
		s.copyInto(copied.Index(index), value.Index(index), path+"["+strconv.Itoa(index)+"]")
	}
	return copied
}

func (s *valueSnapshotState) copyInto(destination, source reflect.Value, path string) {
	copied := s.copy(source, path)
	if !copied.IsValid() {
		destination.SetZero()
		return
	}
	destination.Set(copied)
}
