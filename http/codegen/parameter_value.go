package codegen

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// renderParameterValue lowers an actual native Go value to ordered text without
// applying a location grammar. Nil carriers retain absence; empty carriers retain
// presence so the codec's Rule A check can distinguish them before publication.
func renderParameterValue(wire, physical *expr.AttributeExpr, ctx *codegen.AttributeContext, source, target string, pointer bool) string {
	var code strings.Builder
	base, _ := transportUnderlyingType(wire.Type)
	kind := "loomhttp.ParamScalar"
	if expr.IsArray(base) {
		kind = "loomhttp.ParamList"
	} else if expr.IsObject(base) || expr.IsMap(base) {
		kind = "loomhttp.ParamPairs"
	}
	nullable := pointer || expr.IsArray(base) || expr.IsMap(base) || expr.IsObject(base) || base == expr.Bytes
	if nullable {
		fmt.Fprintf(&code, "%s := loomhttp.ParamValue{Kind: %s}\nif %s != nil {\n%s.Present = true\n", target, kind, source, target)
	} else {
		fmt.Fprintf(&code, "%s := loomhttp.ParamValue{Kind: %s, Present: true}\n", target, kind)
	}
	value := source
	if pointer {
		value = "*" + source
	}
	switch actual := base.(type) {
	case *expr.Array:
		physicalArray := expr.AsArray(physical.Type)
		item := target + "Item"
		fmt.Fprintf(&code, "%s.Items = make([]string, 0, len(%s))\nfor _, %s := range %s {\n", target, value, item, value)
		fmt.Fprintf(&code, "%s.Items = append(%s.Items, %s)\n}\n", target, target, parameterScalarText(actual.ElemType, physicalArray.ElemType, item))
	case *expr.Map:
		physicalMap := expr.AsMap(physical.Type)
		key, item := target+"Key", target+"Item"
		fmt.Fprintf(&code, "%s.Pairs = make([]loomhttp.ParamPair, 0, len(%s))\nfor %s, %s := range %s {\n", target, value, key, item, value)
		fmt.Fprintf(&code, "%s.Pairs = append(%s.Pairs, loomhttp.ParamPair{Key: %s, Value: %s})\n}\n", target, target,
			parameterScalarText(actual.KeyType, physicalMap.KeyType, key), parameterScalarText(actual.ElemType, physicalMap.ElemType, item))
		fmt.Fprintf(&code, "sort.Slice(%s.Pairs, func(i, j int) bool {\nreturn %s.Pairs[i].Key < %s.Pairs[j].Key\n})\n", target, target, target)
	case *expr.Object:
		renderParameterObject(&code, actual, physical, ctx, value, target)
	default:
		fmt.Fprintf(&code, "%s.Text = %s\n", target, transportScalarValue(base, physical.Type, value, parameterCustomText(physical)))
	}
	if nullable {
		code.WriteString("}\n")
	}
	return code.String()
}

func parameterScalarText(wire, physical *expr.AttributeExpr, value string) string {
	base, _ := transportUnderlyingType(wire.Type)
	return transportScalarValue(base, physical.Type, value, parameterCustomText(physical))
}

func parameterCustomText(attribute *expr.AttributeExpr) bool {
	base, _ := transportUnderlyingType(attribute.Type)
	name, _ := codegen.GetMetaType(attribute)
	return base == expr.String && name != ""
}

func renderParameterObject(code *strings.Builder, actual *expr.Object, physical *expr.AttributeExpr, ctx *codegen.AttributeContext, value, target string) {
	parent := physical
	fmt.Fprintf(code, "%s.Pairs = make([]loomhttp.ParamPair, 0, %d)\n", target, len(*actual))
	for _, field := range *actual {
		key, member := parent.FindAttribute(expr.AttributeName(field.Name))
		if member == nil {
			panic(fmt.Sprintf("parameter property %q has no physical field", field.Name))
		}
		fieldValue := value + "." + codegen.GoifyAtt(member, expr.AttributeName(key), true)
		isPointer := ctx.IsPrimitivePointer(key, parent)
		memberBase, _ := transportUnderlyingType(member.Type)
		guard := isPointer || memberBase == expr.Bytes
		if guard {
			fmt.Fprintf(code, "if %s != nil {\n", fieldValue)
		}
		if isPointer {
			fieldValue = "*" + fieldValue
		}
		fmt.Fprintf(code, "%s.Pairs = append(%s.Pairs, loomhttp.ParamPair{Key: %q, Value: %s})\n", target, target,
			expr.ElementName(field.Name), parameterScalarText(field.Attribute, member, fieldValue))
		if guard {
			code.WriteString("}\n")
		}
	}
}
