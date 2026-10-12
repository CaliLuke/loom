package codegen

import (
	json "encoding/json/v2"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/CaliLuke/loom/expr"
)

func formatGoLiteral(v any) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%#v", v)
	return b.String()
}

func formatRawJSONLiteral(value any) (string, error) {
	encoded, err := json.Marshal(value, json.Deterministic(true))
	if err != nil {
		return "", fmt.Errorf("encode authored JSON value: %w", err)
	}
	return "loom.JSONValue(" + strconv.Quote(string(encoded)) + ")", nil
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

func isRawJSONValue(att *expr.AttributeExpr) bool {
	if metaType, _ := GetMetaType(att); metaType != "" {
		return false
	}
	return unalias(att.Type).Kind() == expr.AnyKind
}
