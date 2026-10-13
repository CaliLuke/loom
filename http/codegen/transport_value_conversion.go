package codegen

import (
	"fmt"

	"github.com/CaliLuke/loom/expr"
)

// transportScalarValue formats a dereferenced physical value using its wire
// primitive. Named elements retain declaration identity until this conversion.
func transportScalarValue(wire, physical expr.DataType, value string) string {
	_, named := transportUnderlyingType(physical)
	cast := func(name string, always bool) string {
		if named || always {
			return name + "(" + value + ")"
		}
		return value
	}
	switch wire.Kind() {
	case expr.BooleanKind:
		return "strconv.FormatBool(" + cast("bool", false) + ")"
	case expr.IntKind:
		return "strconv.Itoa(" + cast("int", false) + ")"
	case expr.Int32Kind:
		return "strconv.FormatInt(" + cast("int64", true) + ", 10)"
	case expr.Int64Kind:
		return "strconv.FormatInt(" + cast("int64", false) + ", 10)"
	case expr.UIntKind, expr.UInt32Kind:
		return "strconv.FormatUint(" + cast("uint64", true) + ", 10)"
	case expr.UInt64Kind:
		return "strconv.FormatUint(" + cast("uint64", false) + ", 10)"
	case expr.Float32Kind:
		return "strconv.FormatFloat(" + cast("float64", true) + ", 'f', -1, 32)"
	case expr.Float64Kind:
		return "strconv.FormatFloat(" + cast("float64", false) + ", 'f', -1, 64)"
	case expr.StringKind:
		return cast("string", false)
	case expr.BytesKind:
		return cast("string", true)
	case expr.AnyKind:
		return "loom.JSONValueString(" + value + ")"
	default:
		panic(fmt.Sprintf("unsupported transport scalar %s", wire.Name()))
	}
}

// renderHeaderConversion lowers a physical scalar or list to the native string
// carrier consumed by the header codec. List absence and explicit empty survive.
func renderHeaderConversion(wire, physical expr.DataType, result, value string) string {
	array := expr.AsArray(wire)
	if array == nil {
		return result + " := " + transportScalarValue(wire, physical, value)
	}
	element := expr.AsArray(physical).ElemType.Type
	_, named := transportUnderlyingType(element)
	if array.ElemType.Type.Kind() == expr.StringKind && !named {
		return result + " := " + value
	}
	return fmt.Sprintf("var %s []string\nif %s != nil {\n%s = make([]string, len(%s))\nfor i, e := range %s {\n%s[i] = %s\n}\n}",
		result, value, result, value, value, result, transportScalarValue(array.ElemType.Type, element, "e"))
}
