package codegen

import (
	json "encoding/json/v2"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/CaliLuke/loom/expr"
)

func formatGoLiteral(v any) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%#v", v)
	return b.String()
}

func formatRawJSONLiteral(value any) string {
	encoded, err := json.Marshal(value, json.Deterministic(true))
	if err != nil {
		panic(fmt.Sprintf("encode validated arbitrary JSON default: %v", err))
	}
	return "loom.JSONValue(" + strconv.Quote(string(encoded)) + ")"
}
func formatAttributeGoLiteral(att *expr.AttributeExpr, value any) string {
	if typeName, _ := GetMetaType(att); typeName == "jsontext.Value" {
		actual := reflect.ValueOf(value)
		if actual.IsValid() && actual.Kind() == reflect.Slice && actual.Type().Elem().Kind() == reflect.Uint8 {
			literal := fmt.Sprintf("%#v", actual.Bytes())
			return typeName + strings.TrimPrefix(literal, "[]byte")
		}
	}
	return formatGoLiteral(value)
}

func typedDefaultLiteral(att *expr.AttributeExpr, value any, ta *TransformAttrs) (string, bool) {
	return typedDefaultValueLiteral(att, value, ta)
}

func defaultValueLiteral(att *expr.AttributeExpr, value any, ta *TransformAttrs) string {
	literal, ok := typedDefaultLiteral(att, value, ta)
	if !ok {
		return formatAttributeGoLiteral(att, value)
	}
	return literal
}

func isRawJSONValue(att *expr.AttributeExpr) bool {
	if metaType, _ := GetMetaType(att); metaType != "" {
		return false
	}
	return unalias(att.Type).Kind() == expr.AnyKind
}

func containsRawJSONValue(att *expr.AttributeExpr) bool {
	if isRawJSONValue(att) {
		return true
	}
	switch actual := unalias(att.Type).(type) {
	case *expr.Array:
		return containsRawJSONValue(actual.ElemType)
	case *expr.Map:
		return containsRawJSONValue(actual.KeyType) || containsRawJSONValue(actual.ElemType)
	default:
		return false
	}
}

func typedArrayDefaultLiteral(a *expr.Array, value any, ta *TransformAttrs) (string, bool) {
	actual := reflect.ValueOf(value)
	if !actual.IsValid() || (actual.Kind() != reflect.Array && actual.Kind() != reflect.Slice) {
		return "", false
	}
	items := make([]string, 0, actual.Len())
	for index := range actual.Len() {
		literal, ok := typedDefaultValueLiteral(a.ElemType, actual.Index(index).Interface(), ta)
		if !ok {
			return "", false
		}
		items = append(items, literal)
	}
	elemRef := ta.TargetCtx.Scope.Ref(a.ElemType, ta.TargetCtx.Pkg(a.ElemType))
	return "[]" + elemRef + "{" + strings.Join(items, ", ") + "}", true
}

func typedMapDefaultLiteral(m *expr.Map, value any, ta *TransformAttrs) (string, bool) {
	items, ok := mapDefaultLiteralItems(m, value, ta)
	if !ok {
		return "", false
	}
	keyRef := ta.TargetCtx.Scope.Ref(m.KeyType, ta.TargetCtx.Pkg(m.KeyType))
	if isRawJSONValue(m.KeyType) {
		keyRef = "any"
	}
	elemRef := ta.TargetCtx.Scope.Ref(m.ElemType, ta.TargetCtx.Pkg(m.ElemType))
	return "map[" + keyRef + "]" + elemRef + "{" + strings.Join(items, ", ") + "}", true
}

func mapDefaultLiteralItems(m *expr.Map, value any, ta *TransformAttrs) ([]string, bool) {
	actual := reflect.ValueOf(value)
	if !actual.IsValid() || actual.Kind() != reflect.Map {
		return nil, false
	}
	keys := actual.MapKeys()
	sort.Slice(keys, func(i, j int) bool {
		return fmt.Sprint(keys[i].Interface()) < fmt.Sprint(keys[j].Interface())
	})
	items := make([]string, 0, len(keys))
	for _, key := range keys {
		keyLiteral, ok := typedDefaultMapKeyLiteral(m.KeyType, key.Interface(), ta)
		if !ok {
			return nil, false
		}
		elemLiteral, ok := typedDefaultValueLiteral(m.ElemType, actual.MapIndex(key).Interface(), ta)
		if !ok {
			return nil, false
		}
		items = append(items, keyLiteral+": "+elemLiteral)
	}
	return items, true
}

func typedDefaultMapKeyLiteral(att *expr.AttributeExpr, value any, ta *TransformAttrs) (string, bool) {
	if isRawJSONValue(att) {
		return formatAttributeGoLiteral(att, value), true
	}
	return typedDefaultValueLiteral(att, value, ta)
}

func typedDefaultValueLiteral(att *expr.AttributeExpr, value any, ta *TransformAttrs) (string, bool) {
	if isRawJSONValue(att) {
		return formatRawJSONLiteral(value), true
	}
	switch actual := unalias(att.Type).(type) {
	case *expr.Array:
		if containsRawJSONValue(att) {
			return typedArrayDefaultLiteral(actual, value, ta)
		}
	case *expr.Map:
		return typedMapDefaultLiteral(actual, value, ta)
	}
	return formatAttributeGoLiteral(att, value), true
}
