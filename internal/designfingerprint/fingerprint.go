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
	"strings"

	"github.com/CaliLuke/loom/expr"
)

const formatVersion = "loom-design-v2"

var reflectType = reflect.TypeFor[reflect.Type]()
var metaExprType = reflect.TypeFor[expr.MetaExpr]()

type (
	encoder struct {
		buffer        bytes.Buffer
		active        map[pointerIdentity]int
		frame         *pointerFrame
		cache         *pointerCache
		depth         int
		minCycleDepth int
		cycleMembers  *pointerSet
	}

	pointerIdentity struct {
		typeOf  reflect.Type
		pointer uintptr
	}

	pointerFrame struct {
		identity pointerIdentity
		depth    int
		parent   *pointerFrame
	}

	pathFrame struct {
		parent    *pathFrame
		component string
	}

	pointerCache struct {
		parent *pointerCache
		values map[pointerIdentity]pointerDigest
	}

	pointerDigest struct {
		sum          [sha256.Size]byte
		cycleMembers *pointerSet
	}

	pointerSet struct {
		identities []pointerIdentity
		left       *pointerSet
		right      *pointerSet
	}

	mapEntry struct {
		key   []byte
		value []byte
	}
)

// Digest returns the deterministic SHA-256 digest of an evaluated design and
// the generation inputs that can change its output.
func Digest(root *expr.RootExpr, command, genpkg string, designVersion int) (string, error) {
	enc := &encoder{
		active:        make(map[pointerIdentity]int),
		cache:         newPointerCache(nil),
		minCycleDepth: -1,
	}
	enc.writeString(formatVersion)
	enc.writeString(command)
	enc.writeString(genpkg)
	enc.writeString(strconv.Itoa(designVersion))
	if err := enc.encode(reflect.ValueOf(root), &pathFrame{component: "design"}); err != nil {
		return "", fmt.Errorf("encode evaluated design: %w", err)
	}
	sum := sha256.Sum256(enc.buffer.Bytes())
	return hex.EncodeToString(sum[:]), nil
}

func (e *encoder) encode(value reflect.Value, path *pathFrame) error {
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

func (e *encoder) encodeInterface(value reflect.Value, path *pathFrame) error {
	if value.IsNil() {
		e.writeString("nil")
		return nil
	}
	if value.Type() == reflectType {
		typeValue := value.Interface().(reflect.Type)
		e.writeString(typeValue.PkgPath() + ":" + typeValue.String())
		return nil
	}
	return e.encode(value.Elem(), path.child(".interface"))
}

func (e *encoder) encodePointer(value reflect.Value, path *pathFrame) error {
	if value.IsNil() {
		e.writeString("nil")
		return nil
	}
	identity := pointerIdentity{typeOf: value.Type(), pointer: value.Pointer()}
	if digest, exists := e.cache.lookup(identity); exists && !digest.intersects(e.active) {
		e.writeString("pointer")
		e.writeBytes(digest.sum[:])
		e.addCycleMembers(digest.cycleMembers)
		return nil
	}
	if activeDepth, exists := e.active[identity]; exists {
		e.recordCycle(activeDepth)
		e.writeString("cycle")
		e.writeString(strconv.Itoa(e.depth - activeDepth))
		return nil
	}

	e.active[identity] = e.depth
	child := e.pointerChild(identity)
	err := child.encode(value.Elem(), path.child(".value"))
	delete(e.active, identity)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(child.buffer.Bytes())
	e.addCycleMembers(child.cycleMembers)
	if child.minCycleDepth < 0 || child.minCycleDepth >= e.depth {
		e.cache.values[identity] = pointerDigest{
			sum:          digest,
			cycleMembers: child.cycleMembers,
		}
	} else {
		e.recordCycleDepth(child.minCycleDepth)
	}
	e.writeString("pointer")
	e.writeBytes(digest[:])
	return nil
}

func (e *encoder) encodeStruct(value reflect.Value, path *pathFrame) error {
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
		if err := e.encode(value.Field(i), path.child("."+field.Name)); err != nil {
			return err
		}
	}
	return nil
}

func (e *encoder) encodeMap(value reflect.Value, path *pathFrame) error {
	if value.IsNil() {
		e.writeString("nil")
		return nil
	}
	entries := make([]mapEntry, 0, value.Len())
	iterator := value.MapRange()
	for iterator.Next() {
		if value.Type() == metaExprType && strings.HasPrefix(iterator.Key().String(), "loom:vet:") {
			continue
		}
		keyEncoder := e.fork()
		if err := keyEncoder.encode(iterator.Key(), path.child(".key")); err != nil {
			return err
		}
		valueEncoder := e.fork()
		if err := valueEncoder.encode(iterator.Value(), path.child(".value")); err != nil {
			return err
		}
		e.recordCycleDepth(keyEncoder.minCycleDepth)
		e.recordCycleDepth(valueEncoder.minCycleDepth)
		e.addCycleMembers(keyEncoder.cycleMembers)
		e.addCycleMembers(valueEncoder.cycleMembers)
		entries = append(entries, mapEntry{
			key:   keyEncoder.buffer.Bytes(),
			value: valueEncoder.buffer.Bytes(),
		})
	}
	if value.Type() == metaExprType && len(entries) == 0 {
		e.writeString("nil")
		return nil
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
	active := make(map[pointerIdentity]int, len(e.active))
	for identity, depth := range e.active {
		active[identity] = depth
	}
	return &encoder{
		active:        active,
		frame:         e.frame,
		cache:         newPointerCache(e.cache),
		depth:         e.depth,
		minCycleDepth: -1,
	}
}

func (e *encoder) pointerChild(identity pointerIdentity) *encoder {
	return &encoder{
		active:        e.active,
		frame:         &pointerFrame{identity: identity, depth: e.depth, parent: e.frame},
		cache:         e.cache,
		depth:         e.depth + 1,
		minCycleDepth: -1,
	}
}

func (e *encoder) recordCycle(depth int) {
	if depth < 0 {
		return
	}
	e.recordCycleDepth(depth)
	members := make([]pointerIdentity, 0, e.depth-depth)
	for frame := e.frame; frame != nil && frame.depth >= depth; frame = frame.parent {
		members = append(members, frame.identity)
	}
	e.addCycleMembers(&pointerSet{identities: members})
}

func (e *encoder) recordCycleDepth(depth int) {
	if depth < 0 {
		return
	}
	if e.minCycleDepth < 0 || depth < e.minCycleDepth {
		e.minCycleDepth = depth
	}
}

func (e *encoder) addCycleMembers(members *pointerSet) {
	if members == nil || members == e.cycleMembers {
		return
	}
	if e.cycleMembers == nil {
		e.cycleMembers = members
		return
	}
	e.cycleMembers = &pointerSet{left: e.cycleMembers, right: members}
}

func newPointerCache(parent *pointerCache) *pointerCache {
	return &pointerCache{
		parent: parent,
		values: make(map[pointerIdentity]pointerDigest),
	}
}

func (c *pointerCache) lookup(identity pointerIdentity) (pointerDigest, bool) {
	for current := c; current != nil; current = current.parent {
		if digest, exists := current.values[identity]; exists {
			return digest, true
		}
	}
	return pointerDigest{}, false
}

func (d pointerDigest) intersects(active map[pointerIdentity]int) bool {
	sets := []*pointerSet{d.cycleMembers}
	for len(sets) > 0 {
		last := len(sets) - 1
		set := sets[last]
		sets = sets[:last]
		if set == nil {
			continue
		}
		for _, identity := range set.identities {
			if _, exists := active[identity]; exists {
				return true
			}
		}
		sets = append(sets, set.left, set.right)
	}
	return false
}

func (e *encoder) encodeSequence(value reflect.Value, path *pathFrame) error {
	if value.Kind() == reflect.Slice && value.IsNil() {
		e.writeString("nil")
		return nil
	}
	e.writeString(strconv.Itoa(value.Len()))
	for i := range value.Len() {
		if err := e.encode(value.Index(i), path.child("["+strconv.Itoa(i)+"]")); err != nil {
			return err
		}
	}
	return nil
}

func (e *encoder) encodeScalar(value reflect.Value, path *pathFrame) error {
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

func (p *pathFrame) child(component string) *pathFrame {
	return &pathFrame{parent: p, component: component}
}

func (p *pathFrame) String() string {
	components := make([]string, 0, 8)
	for current := p; current != nil; current = current.parent {
		components = append(components, current.component)
	}
	var path strings.Builder
	for index := len(components) - 1; index >= 0; index-- {
		path.WriteString(components[index])
	}
	return path.String()
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
