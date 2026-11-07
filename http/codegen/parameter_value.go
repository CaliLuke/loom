package codegen

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/representation"
)

// renderParameterValue lowers an actual native Go value to ordered text without
// applying a location grammar. Nil carriers retain absence; empty carriers retain
// presence so the codec's Rule A check can distinguish them before publication.
// fail prefixes the returned error when an integer value cannot be represented
// by its wire type, for example `"", ` inside a (string, error) function.
func renderParameterValue(wire, physical *expr.AttributeExpr, ctx *codegen.AttributeContext, source, target string, pointer bool, fail string) string {
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
		text := parameterScalarCode(&code, actual.ElemType, physicalArray.ElemType, ctx, item, item, fail)
		fmt.Fprintf(&code, "%s.Items = append(%s.Items, %s)\n}\n", target, target, text)
	case *expr.Map:
		physicalMap := expr.AsMap(physical.Type)
		key, item := target+"Key", target+"Item"
		fmt.Fprintf(&code, "%s.Pairs = make([]loomhttp.ParamPair, 0, len(%s))\nfor %s, %s := range %s {\n", target, value, key, item, value)
		keyText := parameterScalarCode(&code, actual.KeyType, physicalMap.KeyType, ctx, key, key, fail)
		itemText := parameterScalarCode(&code, actual.ElemType, physicalMap.ElemType, ctx, item, item, fail)
		fmt.Fprintf(&code, "%s.Pairs = append(%s.Pairs, loomhttp.ParamPair{Key: %s, Value: %s})\n}\n", target, target, keyText, itemText)
		fmt.Fprintf(&code, "sort.Slice(%s.Pairs, func(i, j int) bool {\nreturn %s.Pairs[i].Key < %s.Pairs[j].Key\n})\n", target, target, target)
	case *expr.Object:
		renderParameterObject(&code, actual, physical, ctx, value, target, fail)
	default:
		if representation.IsIntegerKind(base) && !parameterCustomText(physical) {
			code.WriteString(renderIntegerText(base, physical, ctx, value, target+".Text", target+"Wire", fail))
			break
		}
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

// parameterScalarCode returns the text expression of a scalar element. Integer
// elements first declare checked wire text in code under the local prefix.
func parameterScalarCode(code *strings.Builder, wire, physical *expr.AttributeExpr, ctx *codegen.AttributeContext, value, local, fail string) string {
	base, _ := transportUnderlyingType(wire.Type)
	if !representation.IsIntegerKind(base) || parameterCustomText(physical) {
		return parameterScalarText(wire, physical, value)
	}
	text := local + "Text"
	fmt.Fprintf(code, "var %s string\n", text)
	code.WriteString(renderIntegerText(base, physical, ctx, value, text, local+"Wire", fail))
	return text
}

func parameterCustomText(attribute *expr.AttributeExpr) bool {
	base, _ := transportUnderlyingType(attribute.Type)
	name, _ := codegen.GetMetaType(attribute)
	return base == expr.String && name != ""
}

func renderParameterObject(code *strings.Builder, actual *expr.Object, physical *expr.AttributeExpr, ctx *codegen.AttributeContext, value, target, fail string) {
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
		text := parameterScalarCode(code, field.Attribute, member, ctx, fieldValue, target+codegen.GoifyAtt(member, expr.AttributeName(key), true), fail)
		fmt.Fprintf(code, "%s.Pairs = append(%s.Pairs, loomhttp.ParamPair{Key: %q, Value: %s})\n", target, target,
			expr.ElementName(field.Name), text)
		if guard {
			code.WriteString("}\n")
		}
	}
}
