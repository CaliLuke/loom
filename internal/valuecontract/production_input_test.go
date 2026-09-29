package valuecontract

import (
	"encoding"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

type productionInput struct {
	t         *testing.T
	classes   []any
	active    map[productionVisit]bool
	spellings map[string]any
	literals  map[string]any
	readings  map[string]any
}

type productionVisit struct {
	typ     reflect.Type
	pointer uintptr
	length  int
}

func newProductionInput(t *testing.T) *productionInput {
	return &productionInput{t: t, active: make(map[productionVisit]bool), spellings: make(map[string]any), literals: make(map[string]any), readings: make(map[string]any)}
}

func (a *productionInput) class(raw any) uint64 {
	for index, prior := range a.classes {
		if reflect.DeepEqual(prior, raw) {
			return uint64(index + 1)
		}
	}
	a.classes = append(a.classes, raw)
	return uint64(len(a.classes))
}

func (a *productionInput) encode(raw any) any {
	if raw == nil {
		return "null"
	}
	value := reflect.ValueOf(raw)
	if value.Kind() == reflect.Pointer && value.IsNil() {
		return "null"
	}
	if value.Kind() == reflect.Map || value.Kind() == reflect.Slice {
		visit := productionVisit{value.Type(), value.Pointer(), value.Len()}
		if a.active[visit] {
			return referenceConstructor("cycle", map[string]any{"identity": a.class(raw)})
		}
		a.active[visit] = true
		defer delete(a.active, visit)
	}
	payload := a.payload(raw, value)
	return referenceConstructor("host", map[string]any{"identity": a.class(raw), "payload": payload})
}

func (a *productionInput) payload(raw any, value reflect.Value) any {
	for _, protocol := range []reflect.Type{reflect.TypeFor[json.MarshalerTo](), reflect.TypeFor[json.Marshaler](), reflect.TypeFor[encoding.TextAppender](), reflect.TypeFor[encoding.TextMarshaler]()} {
		if value.Type().Implements(protocol) || (value.Kind() != reflect.Pointer && reflect.PointerTo(value.Type()).Implements(protocol)) {
			return referenceConstructor("opaque", map[string]any{"identity": a.class(raw)})
		}
	}
	if productionNativeBytes(value) {
		return a.byteSequence(value)
	}
	switch value.Kind() {
	case reflect.Map:
		if value.IsNil() {
			return "nilMap"
		}
		entries := make([]any, 0, value.Len())
		for _, key := range productionMapKeys(value) {
			entries = append(entries, []any{a.sourceScalar(key.Interface()), a.encode(value.MapIndex(key).Interface())})
		}
		return referenceConstructor("map", map[string]any{"entries": entries})
	case reflect.Slice:

		if value.IsNil() {
			return "nilArray"
		}
		fallthrough
	case reflect.Array:
		items := make([]any, value.Len())
		for index := range value.Len() {
			items[index] = a.encode(value.Index(index).Interface())
		}
		return referenceConstructor("array", map[string]any{"items": items})
	case reflect.Bool, reflect.String, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
		return referenceConstructor("scalar", map[string]any{"value": a.sourceScalar(raw)})
	default:
		return referenceConstructor("opaque", map[string]any{"identity": a.class(raw)})
	}
}

func productionNativeBytes(value reflect.Value) bool {
	return value.IsValid() && (value.Kind() == reflect.Slice || value.Kind() == reflect.Array) && value.Type().Elem() == reflect.TypeFor[byte]()
}

func (a *productionInput) byteSequence(value reflect.Value) any {
	shape := "slice"
	fields := map[string]any{"definedContainer": value.Type().PkgPath() != ""}
	if value.Kind() == reflect.Slice && value.IsNil() {
		shape = "nilSlice"
	} else {
		if value.Kind() == reflect.Array {
			shape = "array"
		}
		items := make([]any, value.Len())
		for index := range items {
			item := value.Index(index).Interface().(byte)
			// Record actual numeric literal codecs for declared Array(Float).
			a.number(item)
			items[index] = []any{a.class(item), uint16(item)}
		}
		fields["items"] = items
	}
	return referenceConstructor("byteSequence", map[string]any{"sequence": referenceConstructor(shape, fields)})
}

// sourceScalar retains the concrete admission class separately from mathematical value.
func (a *productionInput) sourceScalar(raw any) any {
	value := reflect.ValueOf(raw)
	kind := "integer"
	switch value.Kind() {
	case reflect.Bool:
		kind = "boolean"
	case reflect.String:
		kind = "string"
	case reflect.Slice:
		kind = "bytes"
	case reflect.Float32, reflect.Float64:
		kind = "decimal"
	}
	var primitive any
	if value.Type().PkgPath() != "" || (value.Kind() == reflect.Slice && value.Type() != reflect.TypeFor[[]byte]()) {
		primitive = referenceConstructor("definedScalar", map[string]any{"kind": kind})
	} else {
		tags := map[reflect.Kind]string{
			reflect.Bool: "builtinBoolean", reflect.String: "builtinString", reflect.Slice: "nativeBytes",
			reflect.Int: "builtinInt", reflect.Int8: "builtinInt8", reflect.Int16: "builtinInt16", reflect.Int32: "builtinInt32", reflect.Int64: "builtinInt64",
			reflect.Uint: "builtinUInt", reflect.Uint8: "builtinUInt8", reflect.Uint16: "builtinUInt16", reflect.Uint32: "builtinUInt32", reflect.Uint64: "builtinUInt64",
			reflect.Float32: "builtinFloat32", reflect.Float64: "builtinFloat64",
		}
		primitive = tags[value.Kind()]
	}
	return map[string]any{"value": a.scalar(raw), "primitive": primitive}
}

// canonicalRaw removes only source admission evidence. Host equality identities,
// nil markers and each raw scalar kind remain in the canonical Any enum value.
func (a *productionInput) canonicalRaw(input any) any {
	if text, ok := input.(string); ok {
		return text
	}
	node := input.(map[string]any)
	for kind, raw := range node {
		fields := raw.(map[string]any)
		switch kind {
		case "host":
			return referenceConstructor(kind, map[string]any{"identity": fields["identity"], "payload": a.canonicalRaw(fields["payload"])})
		case "scalar":
			return referenceConstructor(kind, map[string]any{"value": fields["value"].(map[string]any)["value"]})
		case "byteSequence":
			for shape, raw := range fields["sequence"].(map[string]any) {
				if shape == "nilSlice" {
					return "nilBytes"
				}
				items := raw.(map[string]any)["items"].([]any)
				bytes := make([]uint16, len(items))
				for i, item := range items {
					bytes[i] = item.([]any)[1].(uint16)
				}
				return referenceConstructor("scalar", map[string]any{"value": referenceConstructor("bytes", map[string]any{"value": bytes})})
			}
		case "array":
			items := fields["items"].([]any)
			result := make([]any, len(items))
			for i, item := range items {
				result[i] = a.canonicalRaw(item)
			}
			return referenceConstructor(kind, map[string]any{"items": result})
		case "map":
			entries := fields["entries"].([]any)
			result := make([]any, 0, len(entries))
			for _, entry := range entries {
				pair := entry.([]any)
				result = append(result, []any{pair[0].(map[string]any)["value"], a.canonicalRaw(pair[1])})
			}
			return referenceConstructor(kind, map[string]any{"entries": result})
		default:
			a.t.Fatalf("unsupported raw enum constructor %s", kind)
		}
	}
	return nil
}

func (a *productionInput) scalar(raw any) any {
	value := reflect.ValueOf(raw)
	switch value.Kind() {
	case reflect.Bool:
		return referenceConstructor("boolean", map[string]any{"value": value.Bool()})
	case reflect.String:
		return referenceConstructor("string", map[string]any{"value": value.String()})
	case reflect.Slice:
		require.Equal(a.t, reflect.Uint8, value.Type().Elem().Kind())
		items := make([]uint16, value.Len())
		for i := range items {
			items[i] = uint16(value.Index(i).Uint())
		}
		return referenceConstructor("bytes", map[string]any{"value": items})
	default:
		return a.number(raw)
	}
}

func (a *productionInput) number(raw any) any {
	value := reflect.ValueOf(raw)
	var scalar, identity map[string]any
	var origin uint64
	if _, custom := raw.(fmt.Stringer); custom {
		origin = a.class(raw)
	}
	switch value.Kind() {
	case reflect.Float32, reflect.Float64:
		scalar, identity = referenceNumericScalar(raw, origin)
	default:
		var integer string
		if value.Kind() >= reflect.Int && value.Kind() <= reflect.Int64 {
			integer = strconv.FormatInt(value.Int(), 10)
		} else {
			integer = strconv.FormatUint(value.Uint(), 10)
		}
		scalar = referenceConstructor("integer", map[string]any{"value": jsontext.Value(integer), "literalOrigin": origin})
		identity = map[string]any{"value": referenceDecimal{Coefficient: jsontext.Value(integer)}, "format": "exact", "negativeZero": false}
	}
	wire, err := json.Marshal(raw)
	require.NoError(a.t, err)
	a.spellings[productionJSON(a.t, identity)] = map[string]any{"number": identity, "text": string(wire)}
	// Defined scalar types cannot enter declared primitive normalization. Their
	// raw Any/key numeric meaning needs JSON spelling, never a Stringer callback.
	if value.Type().PkgPath() != "" {
		return scalar
	}
	literal := fmt.Sprint(raw)
	a.literals[productionJSON(a.t, scalar)] = map[string]any{"input": scalar, "text": literal}
	for _, format := range []string{"binary32", "binary64"} {
		bits := 64
		if format == "binary32" {
			bits = 32
		}
		parsed, parseError := strconv.ParseFloat(literal, bits)
		var result any
		if parseError == nil && !math.IsInf(parsed, 0) && !math.IsNaN(parsed) {
			result = map[string]any{"value": referenceBinaryDecimal(parsed), "negativeZero": parsed == 0 && math.Signbit(parsed)}
			var normalized any = parsed
			if bits == 32 {
				normalized = float32(parsed)
			}
			_, normalizedID := referenceNumericScalar(normalized, 0)
			normalizedWire, err := json.Marshal(normalized)
			require.NoError(a.t, err)
			a.spellings[productionJSON(a.t, normalizedID)] = map[string]any{"number": normalizedID, "text": string(normalizedWire)}
		}
		a.readings[format+":"+literal] = map[string]any{"format": format, "text": literal, "result": result}
	}
	return scalar
}

func (a *productionInput) codecs() map[string]any {
	return map[string]any{"numberReadings": []any{}, "integerReadings": []any{}, "decimalReadings": []any{}, "decimalSpellings": productionSortedRows(a.spellings), "literalSpellings": productionSortedRows(a.literals), "literalReadings": productionSortedRows(a.readings)}
}

func productionJSON(t *testing.T, value any) string {
	t.Helper()
	encoded, err := json.Marshal(value, json.Deterministic(true))
	require.NoError(t, err)
	return string(encoded)
}

func productionSortedRows(rows map[string]any) []any {
	keys := make([]string, 0, len(rows))
	for key := range rows {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	values := make([]any, 0, len(keys))
	for _, key := range keys {
		values = append(values, rows[key])
	}
	return values
}

func productionMapKeys(value reflect.Value) []reflect.Value {
	keys := value.MapKeys()
	sort.Slice(keys, func(i, j int) bool {
		return productionKeyOrder(keys[i]) < productionKeyOrder(keys[j])
	})
	return keys
}

func productionKeyOrder(value reflect.Value) string {
	if value.Kind() == reflect.Interface {
		return productionKeyOrder(value.Elem())
	}
	prefix := value.Type().String() + ":"
	switch value.Kind() {
	case reflect.String:
		return prefix + strconv.Quote(value.String())
	case reflect.Float32, reflect.Float64:
		return prefix + strconv.FormatFloat(value.Float(), 'g', -1, value.Type().Bits())
	default:
		return prefix + fmt.Sprint(value.Interface())
	}
}

func productionCanonical(t *testing.T, wire jsontext.Value) string {
	t.Helper()
	switch wire.Kind() {
	case '{':
		fields := referenceDecode[map[string]jsontext.Value](t, wire)
		for name, value := range fields {
			fields[name] = jsontext.Value(productionCanonical(t, value))
		}
		return productionJSON(t, fields)
	case '[':
		items := referenceDecode[[]jsontext.Value](t, wire)
		for index, value := range items {
			items[index] = jsontext.Value(productionCanonical(t, value))
		}
		return productionJSON(t, items)
	default:
		return string(wire)
	}
}

func TestProductionSourceScalarTags(t *testing.T) {
	type namedInt int
	type namedFloat float32
	type namedString string
	type namedBool bool
	type namedBytes []byte
	for _, tc := range []struct {
		name string
		raw  any
		tag  any
	}{
		{"bool", true, "builtinBoolean"},
		{"string", "x", "builtinString"},
		{"bytes", []byte("hi"), "nativeBytes"},
		{"int", int(1), "builtinInt"},
		{"int8", int8(1), "builtinInt8"},
		{"int16", int16(1), "builtinInt16"},
		{"int32", int32(1), "builtinInt32"},
		{"int64", int64(1), "builtinInt64"},
		{"uint", uint(1), "builtinUInt"},
		{"uint8", uint8(1), "builtinUInt8"},
		{"uint16", uint16(1), "builtinUInt16"},
		{"uint32", uint32(1), "builtinUInt32"},
		{"uint64", uint64(1), "builtinUInt64"},
		{"float32", float32(0.1), "builtinFloat32"},
		{"float64", float64(0.1), "builtinFloat64"},
		{"named int", namedInt(1), referenceConstructor("definedScalar", map[string]any{"kind": "integer"})},
		{"named float", namedFloat(0.1), referenceConstructor("definedScalar", map[string]any{"kind": "decimal"})},
		{"named string", namedString("x"), referenceConstructor("definedScalar", map[string]any{"kind": "string"})},
		{"named bool", namedBool(true), referenceConstructor("definedScalar", map[string]any{"kind": "boolean"})},
		{"named bytes", namedBytes("hi"), referenceConstructor("definedScalar", map[string]any{"kind": "bytes"})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := newProductionInput(t)
			scalar := input.sourceScalar(tc.raw).(map[string]any)
			require.Equal(t, tc.tag, scalar["primitive"])
			require.Equal(t, input.scalar(tc.raw), scalar["value"], "tag capture must not alter the canonical scalar")
		})
	}
}

func TestProductionMapKeysRetainRawPrimitive(t *testing.T) {
	input := newProductionInput(t)
	host := input.encode(map[any]string{int(1): "int", int64(1): "int64"}).(map[string]any)["host"].(map[string]any)
	entries := host["payload"].(map[string]any)["map"].(map[string]any)["entries"].([]any)
	require.Len(t, entries, 2)
	tags := make([]any, 0, len(entries))
	for _, entry := range entries {
		tags = append(tags, entry.([]any)[0].(map[string]any)["primitive"])
	}
	require.ElementsMatch(t, []any{"builtinInt", "builtinInt64"}, tags)
}

func TestProductionNativeByteEvidence(t *testing.T) {
	type namedSlice []byte
	type namedArray [2]byte
	for _, tc := range []struct {
		name    string
		raw     any
		shape   string
		defined bool
	}{
		{"slice", []byte{104, 105}, "slice", false},
		{"array", [2]byte{104, 105}, "array", false},
		{"named slice", namedSlice{104, 105}, "slice", true},
		{"named array", namedArray{104, 105}, "array", true},
		{"nil", []byte(nil), "nilSlice", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := newProductionInput(t)
			host := input.encode(tc.raw).(map[string]any)["host"].(map[string]any)
			payload := host["payload"].(map[string]any)
			require.Contains(t, payload, "byteSequence")
			sequence := payload["byteSequence"].(map[string]any)["sequence"].(map[string]any)
			fields := sequence[tc.shape].(map[string]any)
			require.Equal(t, tc.defined, fields["definedContainer"])
			if tc.shape != "nilSlice" {
				require.Equal(t, []any{[]any{input.class(byte(104)), uint16(104)}, []any{input.class(byte(105)), uint16(105)}}, fields["items"])
			}
		})
	}
}
