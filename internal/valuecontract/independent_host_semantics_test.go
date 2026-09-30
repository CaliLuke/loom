package valuecontract

import (
	"encoding"
	"encoding/json/v2"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

type (
	independentOpaqueFixture struct {
		Name    string
		Version int
	}
	independentOpaqueOther struct {
		Name    string
		Version int
	}
	independentDeclaredObject struct {
		Score float64 `json:"rating"`
		Blob  string  `json:"b"`
	}
	// IndependentEmbeddedObject exercises the adapter's exported anonymous field subset.
	IndependentEmbeddedObject struct {
		Score float64 `json:"rating"`
	}
	independentObjectEnvelope struct {
		IndependentEmbeddedObject
		Blob string `json:"b"`
	}
	independentPointerEnvelope struct {
		*IndependentEmbeddedObject
		Blob string `json:"b"`
	}
	// IndependentAmbiguousLeft supplies one equal-depth embedded JSON field.
	IndependentAmbiguousLeft struct {
		Value string `json:"shared"`
	}
	// IndependentAmbiguousRight supplies the conflicting embedded JSON field.
	IndependentAmbiguousRight struct {
		Value string `json:"shared"`
	}
	independentCyclicObject struct {
		Name string `json:"name"`
		Next *independentCyclicObject
	}
	independentTextCodec struct {
		Text  string
		Calls *int
	}
)

func (value independentTextCodec) MarshalText() ([]byte, error) {
	*value.Calls++
	return []byte(value.Text), nil
}

func TestIndependentOpaqueNormalizationUsesRawDeepIdentity(t *testing.T) {
	attribute := &expr.AttributeExpr{Type: expr.Any}
	left, err := independentResolve(attribute, independentOpaqueFixture{Name: "same", Version: 1})
	require.NoError(t, err)
	equal, err := independentResolve(attribute, independentOpaqueFixture{Name: "same", Version: 1})
	require.NoError(t, err)
	different, err := independentResolve(attribute, independentOpaqueFixture{Name: "different", Version: 1})
	require.NoError(t, err)
	otherType, err := independentResolve(attribute, independentOpaqueOther{Name: "same", Version: 1})
	require.NoError(t, err)
	known, err := independentResolve(attribute, "same")
	require.NoError(t, err)
	require.True(t, independentResolvedEqual(left, equal))
	require.False(t, independentResolvedEqual(left, different))
	require.False(t, independentResolvedEqual(left, otherType))
	require.False(t, independentResolvedEqual(left, known))

	pointerLeft, err := independentResolve(attribute, &independentOpaqueFixture{Name: "same", Version: 1})
	require.NoError(t, err)
	pointerEqual, err := independentResolve(attribute, &independentOpaqueFixture{Name: "same", Version: 1})
	require.NoError(t, err)
	require.True(t, independentResolvedEqual(pointerLeft, pointerEqual))
	require.False(t, independentResolvedEqual(left, pointerLeft), "pointer and value host types stay distinct")

	var nilPointer *independentOpaqueFixture
	_, err = independentResolve(attribute, nilPointer)
	require.ErrorContains(t, err, "null is not admitted")
	nullable := &expr.AttributeExpr{Type: expr.Any, Nullable: true}
	nullValue, err := independentResolve(nullable, nilPointer)
	require.NoError(t, err)
	require.Equal(t, "null", nullValue.presence)
}

func TestIndependentPlainStructNormalizationUsesDeclaredObjectSemantics(t *testing.T) {
	attribute := &expr.AttributeExpr{Type: &expr.Object{
		{Name: "score:rating", Attribute: &expr.AttributeExpr{Type: expr.Float32}},
		{Name: "blob:b", Attribute: &expr.AttributeExpr{Type: expr.Bytes}},
	}}
	want, err := independentResolve(attribute, map[string]any{"rating": float32(0.1), "b": []byte("hi")})
	require.NoError(t, err)

	for _, raw := range []any{
		independentDeclaredObject{Score: 0.1, Blob: "hi"},
		&independentDeclaredObject{Score: 0.1, Blob: "hi"},
		independentObjectEnvelope{IndependentEmbeddedObject: IndependentEmbeddedObject{Score: 0.1}, Blob: "hi"},
		independentPointerEnvelope{IndependentEmbeddedObject: &IndependentEmbeddedObject{Score: 0.1}, Blob: "hi"},
	} {
		resolved, resolveErr := independentResolve(attribute, raw)
		require.NoError(t, resolveErr)
		require.True(t, independentResolvedEqual(want, resolved), "%T", raw)
	}
	nilEmbedded, err := independentResolve(attribute, independentPointerEnvelope{Blob: "hi"})
	require.NoError(t, err)
	wantNil, err := independentResolve(attribute, map[string]any{"b": []byte("hi")})
	require.NoError(t, err)
	require.True(t, independentResolvedEqual(wantNil, nilEmbedded))
}

func TestIndependentPlainStructNormalizationOwnsSnapshotsAndRejectsCycles(t *testing.T) {
	attribute := &expr.AttributeExpr{Type: &expr.Object{
		{Name: "score:rating", Attribute: &expr.AttributeExpr{Type: expr.Float32}},
		{Name: "blob:b", Attribute: &expr.AttributeExpr{Type: expr.Bytes}},
	}}
	raw := &independentDeclaredObject{Score: 0.1, Blob: "hi"}
	resolved, err := independentResolve(attribute, raw)
	require.NoError(t, err)
	raw.Score, raw.Blob = 0.2, "changed"
	want, err := independentResolve(attribute, map[string]any{"rating": float32(0.1), "b": []byte("hi")})
	require.NoError(t, err)
	require.True(t, independentResolvedEqual(want, resolved))

	cycle := &independentCyclicObject{Name: "root"}
	cycle.Next = cycle
	_, err = independentResolve(&expr.AttributeExpr{Type: &expr.Object{
		{Name: "name", Attribute: &expr.AttributeExpr{Type: expr.String}},
	}}, cycle)
	require.ErrorContains(t, err, "cyclic value")
}

func TestIndependentPlainStructNormalizationOmitsAmbiguityAndKeepsCodecsOpaque(t *testing.T) {
	optional := &expr.AttributeExpr{Type: &expr.Object{
		{Name: "shared", Attribute: &expr.AttributeExpr{Type: expr.String}},
	}}
	ambiguousType := reflect.StructOf([]reflect.StructField{
		{Name: "IndependentAmbiguousLeft", Type: reflect.TypeFor[IndependentAmbiguousLeft](), Anonymous: true},
		{Name: "IndependentAmbiguousRight", Type: reflect.TypeFor[IndependentAmbiguousRight](), Anonymous: true},
	})
	ambiguousValue := reflect.New(ambiguousType).Elem()
	ambiguousValue.Field(0).Set(reflect.ValueOf(IndependentAmbiguousLeft{Value: "left"}))
	ambiguousValue.Field(1).Set(reflect.ValueOf(IndependentAmbiguousRight{Value: "right"}))
	ambiguous := ambiguousValue.Interface()
	resolved, err := independentResolve(optional, ambiguous)
	require.NoError(t, err)
	require.Equal(t, "absent", resolved.fields[0].value.presence)
	required := &expr.AttributeExpr{
		Type:       optional.Type,
		Validation: &expr.ValidationExpr{Required: []string{"shared"}},
	}
	_, err = independentResolve(required, ambiguous)
	require.ErrorContains(t, err, "required field \"shared\"")

	calls := 0
	codec := independentTextCodec{Text: "opaque", Calls: &calls}
	resolved, err = independentResolve(optional, codec)
	require.NoError(t, err)
	require.Equal(t, "opaque", resolved.kind)
	require.Equal(t, codec, resolved.opaque)
	require.Zero(t, calls)

	knownSibling := &expr.AttributeExpr{Type: &expr.Object{
		{Name: "blob", Attribute: &expr.AttributeExpr{Type: expr.Bytes}},
		{Name: "custom", Attribute: &expr.AttributeExpr{Type: expr.Any}},
	}}
	_, err = independentResolve(knownSibling, map[string]any{"blob": 42, "custom": codec})
	require.ErrorContains(t, err, "field \"blob\"")
	require.Zero(t, calls)
}

type (
	independentSnapshotVisit struct {
		typeOf  reflect.Type
		pointer uintptr
		length  int
	}
	independentSnapshotState struct {
		active map[independentSnapshotVisit]bool
		copies map[independentSnapshotVisit]reflect.Value
	}
	independentStructField struct {
		value  reflect.Value
		depth  int
		tagged bool
	}
)

func independentSnapshot(raw any) (any, error) {
	state := independentSnapshotState{
		active: make(map[independentSnapshotVisit]bool),
		copies: make(map[independentSnapshotVisit]reflect.Value),
	}
	value, err := state.copy(reflect.ValueOf(raw))
	if err != nil || !value.IsValid() {
		return nil, err
	}
	return value.Interface(), nil
}

func (state *independentSnapshotState) copy(value reflect.Value) (reflect.Value, error) {
	if !value.IsValid() || independentHasCustomCodec(value.Type()) {
		return value, nil
	}
	if value.Kind() == reflect.Interface {
		if value.IsNil() {
			return reflect.Zero(value.Type()), nil
		}
		child, err := state.copy(value.Elem())
		if err != nil {
			return reflect.Value{}, err
		}
		result := reflect.New(value.Type()).Elem()
		result.Set(child)
		return result, nil
	}
	visit := independentSnapshotVisit{typeOf: value.Type()}
	switch value.Kind() {
	case reflect.Map, reflect.Slice, reflect.Pointer:
		if value.IsNil() {
			return reflect.Zero(value.Type()), nil
		}
		visit.pointer = value.Pointer()
		if value.Kind() != reflect.Pointer {
			visit.length = value.Len()
		}
		if state.active[visit] {
			return reflect.Value{}, fmt.Errorf("cyclic value")
		}
		if copy, exists := state.copies[visit]; exists {
			return copy, nil
		}
		state.active[visit] = true
		defer delete(state.active, visit)
	}
	switch value.Kind() {
	case reflect.Map:
		result := reflect.MakeMapWithSize(value.Type(), value.Len())
		state.copies[visit] = result
		iterator := value.MapRange()
		for iterator.Next() {
			child, err := state.copy(iterator.Value())
			if err != nil {
				return reflect.Value{}, err
			}
			result.SetMapIndex(iterator.Key(), child)
		}
		return result, nil
	case reflect.Slice:
		result := reflect.MakeSlice(value.Type(), value.Len(), value.Len())
		state.copies[visit] = result
		for index := range value.Len() {
			child, err := state.copy(value.Index(index))
			if err != nil {
				return reflect.Value{}, err
			}
			result.Index(index).Set(child)
		}
		return result, nil
	case reflect.Array:
		result := reflect.New(value.Type()).Elem()
		for index := range value.Len() {
			child, err := state.copy(value.Index(index))
			if err != nil {
				return reflect.Value{}, err
			}
			result.Index(index).Set(child)
		}
		return result, nil
	case reflect.Pointer:
		result := reflect.New(value.Type().Elem())
		state.copies[visit] = result
		child, err := state.copy(value.Elem())
		if err != nil {
			return reflect.Value{}, err
		}
		result.Elem().Set(child)
		return result, nil
	case reflect.Struct:
		result := reflect.New(value.Type()).Elem()
		result.Set(value)
		for index := range value.NumField() {
			if !value.Type().Field(index).IsExported() {
				continue
			}
			child, err := state.copy(value.Field(index))
			if err != nil {
				return reflect.Value{}, err
			}
			result.Field(index).Set(child)
		}
		return result, nil
	default:
		return value, nil
	}
}

func independentHasCustomCodec(typ reflect.Type) bool {
	protocols := []reflect.Type{
		reflect.TypeFor[json.MarshalerTo](),
		reflect.TypeFor[json.Marshaler](),
		reflect.TypeFor[encoding.TextAppender](),
		reflect.TypeFor[encoding.TextMarshaler](),
	}
	for _, protocol := range protocols {
		if typ.Implements(protocol) || (typ.Kind() != reflect.Pointer && reflect.PointerTo(typ).Implements(protocol)) {
			return true
		}
	}
	return false
}

func independentPlainStructMap(value reflect.Value) map[string]any {
	candidates := make(map[string][]independentStructField)
	independentCollectStructFields(value, 0, candidates)
	result := make(map[string]any, len(candidates))
	for name, fields := range candidates {
		depth := fields[0].depth
		for _, field := range fields[1:] {
			depth = min(depth, field.depth)
		}
		var selected []independentStructField
		for _, field := range fields {
			if field.depth == depth {
				selected = append(selected, field)
			}
		}
		var tagged []independentStructField
		for _, field := range selected {
			if field.tagged {
				tagged = append(tagged, field)
			}
		}
		if len(tagged) == 1 {
			selected = tagged
		} else if len(tagged) > 1 || len(selected) != 1 {
			continue
		}
		result[name] = selected[0].value.Interface()
	}
	return result
}

func independentCollectStructFields(value reflect.Value, depth int, fields map[string][]independentStructField) {
	for index := range value.NumField() {
		fieldType := value.Type().Field(index)
		if !fieldType.IsExported() {
			continue
		}
		fieldValue := value.Field(index)
		name, _, _ := strings.Cut(fieldType.Tag.Get("json"), ",")
		if name == "-" {
			continue
		}
		tagged := name != ""
		if fieldType.Anonymous && !tagged {
			embedded := fieldValue
			for embedded.Kind() == reflect.Pointer && !embedded.IsNil() && !independentHasCustomCodec(embedded.Type()) {
				embedded = embedded.Elem()
			}
			if embedded.Kind() == reflect.Pointer && embedded.IsNil() {
				continue
			}
			if embedded.Kind() == reflect.Struct && !independentHasCustomCodec(embedded.Type()) {
				independentCollectStructFields(embedded, depth+1, fields)
				continue
			}
		}
		if !fieldType.IsExported() {
			continue
		}
		if name == "" {
			name = fieldType.Name
		}
		fields[name] = append(fields[name], independentStructField{value: fieldValue, depth: depth, tagged: tagged})
	}
}
