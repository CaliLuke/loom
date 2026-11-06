package codegen

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// renderParameterAssignment assigns a present text value to its native carrier.
// The emitted statements run in an error-returning scope. Wire errors contain
// only the binding and declared property names, never peer-supplied text.
func renderParameterAssignment(wire, physical *expr.AttributeExpr, ctx *codegen.AttributeContext, source, target, name string, pointer bool) string {
	var code strings.Builder
	base, _ := transportUnderlyingType(wire.Type)
	ref := ctx.Scope.Ref(physical, ctx.Pkg(physical))
	switch actual := base.(type) {
	case *expr.Array:
		physicalArray := expr.AsArray(physical.Type)
		fmt.Fprintf(&code, "%s = make(%s, len(%s.Items))\nfor parameterIndex, parameterText := range %s.Items {\n", target, ref, source, source)
		code.WriteString(parameterScalarAssignment(actual.ElemType, physicalArray.ElemType, ctx, "parameterText", target+"[parameterIndex]", name, false))
		code.WriteString("}\n")
	case *expr.Map:
		physicalMap := expr.AsMap(physical.Type)
		fmt.Fprintf(&code, "%s = make(%s, len(%s.Pairs))\nfor _, parameterPair := range %s.Pairs {\n", target, ref, source, source)
		fmt.Fprintf(&code, "var parameterKey %s\nvar parameterItem %s\n", ctx.Scope.Ref(physicalMap.KeyType, ctx.Pkg(physicalMap.KeyType)), ctx.Scope.Ref(physicalMap.ElemType, ctx.Pkg(physicalMap.ElemType)))
		code.WriteString(parameterScalarAssignment(actual.KeyType, physicalMap.KeyType, ctx, "parameterPair.Key", "parameterKey", name, false))
		code.WriteString(parameterScalarAssignment(actual.ElemType, physicalMap.ElemType, ctx, "parameterPair.Value", "parameterItem", name, false))
		fmt.Fprintf(&code, "if _, parameterExists := %s[parameterKey]; parameterExists {\nreturn fmt.Errorf(\"%%s\", %q)\n}\n%s[parameterKey] = parameterItem\n}\n", target, "parameter "+name+": duplicate converted key", target)
	case *expr.Object:
		fmt.Fprintf(&code, "%s = new(%s)\nparameterSeen := make(map[string]bool, len(%s.Pairs))\nfor _, parameterPair := range %s.Pairs {\n", target, ctx.Scope.Name(physical, ctx.Pkg(physical), ctx.Pointer, ctx.UseDefault), source, source)
		fmt.Fprintf(&code, "if parameterSeen[parameterPair.Key] {\nreturn fmt.Errorf(\"%%s\", %q)\n}\nparameterSeen[parameterPair.Key] = true\nswitch parameterPair.Key {\n", "parameter "+name+": duplicate property")
		for _, field := range *actual {
			key, member := physical.FindAttribute(expr.AttributeName(field.Name))
			if member == nil {
				panic(fmt.Sprintf("parameter property %q has no physical field", field.Name))
			}
			fmt.Fprintf(&code, "case %q:\n", expr.ElementName(field.Name))
			fieldTarget := target + "." + ctx.Scope.Field(member, expr.AttributeName(key), true)
			code.WriteString(parameterScalarAssignment(field.Attribute, member, ctx, "parameterPair.Value", fieldTarget, name, ctx.IsPrimitivePointer(key, physical)))
		}
		fmt.Fprintf(&code, "default:\nreturn fmt.Errorf(\"%%s\", %q)\n}\n}\n", "parameter "+name+": unknown property")
		for _, field := range *actual {
			if wire.IsRequired(field.Name) {
				fmt.Fprintf(&code, "if !parameterSeen[%q] {\nreturn fmt.Errorf(\"%%s\", %q)\n}\n", expr.ElementName(field.Name), "parameter "+name+": missing property "+expr.ElementName(field.Name))
			} else if value, ok := selectedTransportConstraints(field.Attribute).Default(); ok {
				key, member := physical.FindAttribute(expr.AttributeName(field.Name))
				fmt.Fprintf(&code, "if !parameterSeen[%q] {\n", expr.ElementName(field.Name))
				fieldTarget := target + "." + ctx.Scope.Field(member, expr.AttributeName(key), true)
				code.WriteString(renderParameterDefault(member, ctx, value, fieldTarget, name, ctx.IsPrimitivePointer(key, physical)))
				code.WriteString("}\n")
			}
		}
	default:
		code.WriteString(parameterScalarAssignment(wire, physical, ctx, source+".Text", target, name, pointer))
	}
	return code.String()
}

func parameterScalarAssignment(wire, physical *expr.AttributeExpr, ctx *codegen.AttributeContext, source, target, name string, pointer bool) string {
	var code strings.Builder
	base, _ := transportUnderlyingType(wire.Type)
	ref := ctx.Scope.Ref(physical, ctx.Pkg(physical))
	code.WriteString("{\n")
	if parameterCustomText(physical) {
		fmt.Fprintf(&code, "var parameterParsed %s\nif parameterError := parameterParsed.UnmarshalText([]byte(%s)); parameterError != nil {\nreturn fmt.Errorf(\"%%s\", %q)\n}\n", ref, source, "parameter "+name+": invalid text value")
	} else {
		parser := ""
		switch base.Kind() {
		case expr.StringKind, expr.BytesKind:
			// Both conversions are defined for named string and byte-slice carriers.
		case expr.BooleanKind:
			parser = "strconv.ParseBool(" + source + ")"
		case expr.IntKind:
			parser = "strconv.ParseInt(" + source + ", 10, strconv.IntSize)"
		case expr.Int32Kind:
			parser = "strconv.ParseInt(" + source + ", 10, 32)"
		case expr.Int64Kind:
			parser = "strconv.ParseInt(" + source + ", 10, 64)"
		case expr.UIntKind:
			parser = "strconv.ParseUint(" + source + ", 10, strconv.IntSize)"
		case expr.UInt32Kind:
			parser = "strconv.ParseUint(" + source + ", 10, 32)"
		case expr.UInt64Kind:
			parser = "strconv.ParseUint(" + source + ", 10, 64)"
		case expr.Float32Kind:
			parser = "strconv.ParseFloat(" + source + ", 32)"
		case expr.Float64Kind:
			parser = "strconv.ParseFloat(" + source + ", 64)"
		default:
			panic(fmt.Sprintf("unsupported parameter scalar %s", base.Name()))
		}
		if parser != "" {
			fmt.Fprintf(&code, "parameterParsed, parameterError := %s\nif parameterError != nil {\nreturn fmt.Errorf(\"%%s\", %q)\n}\n", parser, "parameter "+name+": invalid "+base.Name()+" value")
			source = "parameterParsed"
		}
		if pointer {
			fmt.Fprintf(&code, "parameterValue := %s(%s)\n%s = &parameterValue\n", ref, source, target)
		} else {
			fmt.Fprintf(&code, "%s = %s(%s)\n", target, ref, source)
		}
		code.WriteString("}\n")
		return code.String()
	}
	if pointer {
		fmt.Fprintf(&code, "%s = &parameterParsed\n", target)
	} else {
		fmt.Fprintf(&code, "%s = parameterParsed\n", target)
	}
	code.WriteString("}\n")
	return code.String()
}
