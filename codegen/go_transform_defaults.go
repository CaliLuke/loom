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

// defaultValueLiteral renders a validated default in its target representation.
// Collection values retain their target identity at every nesting level.
func defaultValueLiteral(att *expr.AttributeExpr, value any, ta *TransformAttrs) string {
	if isRawJSONValue(att) {
		return formatRawJSONLiteral(value)
	}
	switch actual := unalias(att.Type).(type) {
	case *expr.Array:
		return typedArrayDefaultLiteral(att, actual, value, ta)
	case *expr.Map:
		return typedMapDefaultLiteral(att, actual, value, ta)
	default:
		return formatAttributeGoLiteral(att, value)
	}
}

func isRawJSONValue(att *expr.AttributeExpr) bool {
	if metaType, _ := GetMetaType(att); metaType != "" {
		return false
	}
	return unalias(att.Type).Kind() == expr.AnyKind
}

func typedArrayDefaultLiteral(att *expr.AttributeExpr, array *expr.Array, value any, ta *TransformAttrs) string {
	actual := reflect.ValueOf(value)
	if !actual.IsValid() || (actual.Kind() != reflect.Array && actual.Kind() != reflect.Slice) {
		panic(NewError(nil, att, fmt.Errorf("render validated array default: unexpected value %T", value)))
	}
	typeRef := ta.TargetCtx.Scope.Ref(att, ta.TargetCtx.Pkg(att))
	if actual.Kind() == reflect.Slice && actual.IsNil() {
		return typeRef + "(nil)"
	}
	items := make([]string, actual.Len())
	for index := range actual.Len() {
		items[index] = defaultValueLiteral(array.ElemType, actual.Index(index).Interface(), ta)
	}
	return typeRef + "{" + strings.Join(items, ", ") + "}"
}

func typedMapDefaultLiteral(att *expr.AttributeExpr, mapping *expr.Map, value any, ta *TransformAttrs) string {
	actual := reflect.ValueOf(value)
	if !actual.IsValid() || actual.Kind() != reflect.Map {
		panic(NewError(nil, att, fmt.Errorf("render validated map default: unexpected value %T", value)))
	}
	typeRef := ta.TargetCtx.Scope.Ref(att, ta.TargetCtx.Pkg(att))
	if actual.IsNil() {
		return typeRef + "(nil)"
	}
	keys := actual.MapKeys()
	sort.Slice(keys, func(i, j int) bool {
		return fmt.Sprint(keys[i].Interface()) < fmt.Sprint(keys[j].Interface())
	})
	items := make([]string, 0, len(keys))
	for _, key := range keys {
		keyLiteral := formatAttributeGoLiteral(mapping.KeyType, key.Interface())
		if !isRawJSONValue(mapping.KeyType) {
			keyLiteral = defaultValueLiteral(mapping.KeyType, key.Interface(), ta)
		}
		elemLiteral := defaultValueLiteral(mapping.ElemType, actual.MapIndex(key).Interface(), ta)
		items = append(items, keyLiteral+": "+elemLiteral)
	}
	return typeRef + "{" + strings.Join(items, ", ") + "}"
}
