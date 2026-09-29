package valuecontract

import (
	"reflect"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func (g *productionGraph) output(input *productionInput, result expr.ValueResult, raw any) any {
	if value, present := result.Value(); present {
		missing := make([]any, 0, len(result.Missing()))
		for _, path := range result.Missing() {
			mapped := make([]referenceIdentity, 0, len(path))
			for _, id := range path {
				mapped = append(mapped, g.identity(id))
			}
			missing = append(missing, mapped)
		}
		return map[string]any{"ok": map[string]any{"value": g.value(input, value, raw, false), "missing": missing}}
	}
	failure := "invalid"
	switch result.Outcome() {
	case expr.ValueAmbiguous:
		failure = "ambiguous"
	case expr.ValueUnsupported:
		failure = "unsupported"
	default:
		for _, diagnostic := range result.Diagnostics() {
			if diagnostic.Code == "cyclic" {
				failure = "cyclic"
			}
		}
	}
	return map[string]any{"error": failure}
}

func (g *productionGraph) value(input *productionInput, value expr.ResolvedValue, raw any, host bool) any {
	payload := g.valueBody(input, value, raw, host)
	if host && raw != nil {
		return referenceConstructor("host", map[string]any{"identity": input.class(raw), "payload": payload})
	}
	return payload
}

func (g *productionGraph) valueBody(input *productionInput, value expr.ResolvedValue, raw any, host bool) any {
	switch value.Presence() {
	case expr.ValueAbsent:
		return "absent"
	case expr.ValueNull:
		return "null"
	case expr.ValueNil:
		switch value.Kind() {
		case expr.ValueKindArray:
			return "nilArray"
		case expr.ValueKindMap:
			return "nilMap"
		default:
			return "nilBytes"
		}
	}
	switch value.Kind() {
	case expr.ValueKindScalar:
		scalar, present := value.Scalar()
		require.True(g.t, present)
		return referenceConstructor("scalar", map[string]any{"value": input.scalar(scalar)})
	case expr.ValueKindAny:
		payload, present := value.Any()
		require.True(g.t, present)
		owned, present := value.RawAny()
		require.True(g.t, present)
		return referenceConstructor("any", map[string]any{"payload": g.value(input, payload, owned, true)})
	case expr.ValueKindArray:
		if host && productionNativeBytes(reflect.ValueOf(raw)) {
			return g.nativeByteValue(input, value, raw)
		}
		items := make([]any, 0, len(value.Elements()))
		for index, child := range value.Elements() {
			var childRaw any
			if raw != nil {
				childRaw = reflect.ValueOf(raw).Index(index).Interface()
			}
			items = append(items, g.value(input, child, childRaw, host))
		}
		return referenceConstructor("array", map[string]any{"items": items})
	case expr.ValueKindMap:
		entries := make([]any, 0, len(value.Entries()))
		var keys []reflect.Value
		if raw != nil {
			keys = productionMapKeys(reflect.ValueOf(raw))
		}
		for index, entry := range value.Entries() {
			key, present := entry.Key.Scalar()
			require.True(g.t, present)
			var childRaw any
			if len(keys) > index {
				childRaw = reflect.ValueOf(raw).MapIndex(keys[index]).Interface()
			}
			entries = append(entries, []any{input.scalar(key), g.value(input, entry.Value, childRaw, host)})
		}
		return referenceConstructor("map", map[string]any{"entries": entries})
	case expr.ValueKindObject:
		return g.objectValue(input, value, raw)
	case expr.ValueKindUnion:
		occurrence, branch, payload, selected := value.Union()
		require.True(g.t, selected)
		return referenceConstructor("union", map[string]any{"occurrence": g.identity(occurrence), "branch": g.identity(branch), "payload": g.value(input, payload, raw, false)})
	default:
		g.t.Fatalf("unsupported resolved kind %v", value.Kind())
		return nil
	}
}

func (g *productionGraph) objectValue(input *productionInput, value expr.ResolvedValue, raw any) any {
	fields, additional := []any{}, []any{}
	for _, field := range value.Fields() {
		var childRaw any
		if raw != nil {
			inputMap := reflect.ValueOf(raw)
			if inputMap.Kind() == reflect.Map && inputMap.Type().Key().Kind() == reflect.String {
				item := inputMap.MapIndex(reflect.ValueOf(field.Name).Convert(inputMap.Type().Key()))
				if item.IsValid() {
					childRaw = item.Interface()
				}
			}
		}
		if field.Member == (expr.ValueIdentity{}) {
			additional = append(additional, []any{field.Name, g.value(input, field.Value, childRaw, true)})
		} else {
			fields = append(fields, []any{g.identity(field.Member), g.value(input, field.Value, childRaw, false)})
		}
	}
	return referenceConstructor("object", map[string]any{"fields": fields, "additional": additional})
}

// nativeByteValue canonicalizes a raw Any binary carrier only after checking
// the actual resolved elements and their original host identities. Declared
// Array results never take this path, even when supplied with the same bytes.
func (g *productionGraph) nativeByteValue(input *productionInput, value expr.ResolvedValue, raw any) any {
	original := reflect.ValueOf(raw)
	children := value.Elements()
	require.Len(g.t, children, original.Len())
	bytes := make([]uint16, len(children))
	for index, child := range children {
		scalar, present := child.Scalar()
		require.True(g.t, present)
		require.Equal(g.t, original.Index(index).Interface(), scalar)
		require.Equal(g.t, input.class(original.Index(index).Interface()), input.class(scalar))
		bytes[index] = uint16(scalar.(byte))
	}
	return referenceConstructor("scalar", map[string]any{"value": referenceConstructor("bytes", map[string]any{"value": bytes})})
}
