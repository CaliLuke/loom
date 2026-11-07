package codegen

import (
	"fmt"
	"go/types"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/representation"
)

// transportScalarValue formats a dereferenced physical value using its wire
// primitive and analyzed custom text contract. Named elements retain declaration
// identity until this conversion.
func transportScalarValue(wire, physical expr.DataType, value string, customText bool) string {
	if customText {
		return "fmt.Sprint(" + value + ")"
	}
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
func renderHeaderConversion(wire, physical expr.DataType, result, value string, customText bool) string {
	array := expr.AsArray(wire)
	if array == nil {
		return result + " := " + transportScalarValue(wire, physical, value, customText)
	}
	element := expr.AsArray(physical).ElemType.Type
	_, named := transportUnderlyingType(element)
	if array.ElemType.Type.Kind() == expr.StringKind && !named {
		return result + " := " + value
	}
	return fmt.Sprintf("var %s []string\nif %s != nil {\n%s = make([]string, len(%s))\nfor i, e := range %s {\n%s[i] = %s\n}\n}",
		result, value, result, value, value, result, transportScalarValue(array.ElemType.Type, element, "e", false))
}

// transportFieldReference selects the analyzed physical field without formatting.
func transportFieldReference(attribute *AttributeData, source string) string {
	if attribute.FieldName != "" {
		source += "." + attribute.FieldName
	}
	if attribute.FieldPointer {
		source = "*" + source
	}
	return source
}

// renderIntegerText assigns the decimal wire text of an integer value held in
// its actual physical storage. Storage is converted to the wire's canonical Go
// carrier through the shared value-level contract; an unrepresentable value
// executes fail with the conversion error before any text exists.
func renderIntegerText(wire expr.DataType, physical *expr.AttributeExpr, ctx *codegen.AttributeContext, value, result, local, fail string) string {
	return renderIntegerTextAssign(wire, physical, ctx, value, result, local, fail, "=")
}

func renderIntegerTextAssign(wire expr.DataType, physical *expr.AttributeExpr, ctx *codegen.AttributeContext, value, result, local, fail, assign string) string {
	carrier := &expr.AttributeExpr{Type: wire}
	conversion, _, err := representation.ClassifyIntegerConversion(physical, carrier)
	if err != nil {
		panic(err) // bug: admission accepted non-integer storage for an integer wire
	}
	sourceRef := ctx.Scope.Ref(physical, ctx.Pkg(physical))
	targetRef := codegen.GoNativeTypeName(wire)
	converted := value
	if sourceRef != targetRef {
		converted = targetRef + "(" + value + ")"
	}
	var code strings.Builder
	if conversion.Checked {
		if conversion.SourceOpaque {
			code.WriteString(representation.IntegerCapabilityAssertion(sourceRef))
		}
		fmt.Fprintf(&code, "%s := %s\nif %s {\nreturn %s%s\n}\n", local, converted, representation.IntegerLossCondition(sourceRef, value, local), fail, representation.IntegerLossError(targetRef, value))
		converted = local
	}
	fmt.Fprintf(&code, "%s %s %s\n", result, assign, integerWireText(wire, converted))
	return code.String()
}

// isIntegerValue reports a transport header or cookie value whose integer text
// is owned by the shared physical conversion.
func isIntegerValue(data *AttributeData) bool {
	if data.fieldAttribute == nil || data.fieldContext == nil || data.IsTextUnmarshaler {
		return false
	}
	wire, _ := transportUnderlyingType(data.Type)
	if array := expr.AsArray(wire); array != nil {
		wire = array.ElemType.Type
	}
	return representation.IsIntegerKind(wire)
}

// isStoredIntegerValue reports an integer value whose physical storage may hold
// values outside its wire type, including opaque external storage whose
// Stringer text must never reach the wire. Lossless generated storage keeps
// its existing faithful decimal text.
func isStoredIntegerValue(data *AttributeData) bool {
	if !isIntegerValue(data) || expr.AsArray(data.Type) != nil {
		return false
	}
	wire, _ := transportUnderlyingType(data.Type)
	conversion, _, err := representation.ClassifyIntegerConversion(data.fieldAttribute, &expr.AttributeExpr{Type: wire})
	if err != nil {
		panic(err) // bug: admission accepted non-integer storage for an integer wire
	}
	return conversion.Checked
}

// renderIntegerValue declares result as the wire text of a dereferenced header
// or cookie value held in its actual physical storage. Lists keep nil absence.
func renderIntegerValue(data *AttributeData, result, value, fail string) string {
	wire, _ := transportUnderlyingType(data.Type)
	physical := data.fieldAttribute
	array := expr.AsArray(wire)
	if array == nil {
		return strings.TrimSuffix(renderIntegerTextAssign(wire, physical, data.fieldContext, value, result, result+"Wire", fail, ":="), "\n")
	}
	element := expr.AsArray(physical.Type).ElemType
	return fmt.Sprintf("var %s []string\nif %s != nil {\n%s = make([]string, len(%s))\nfor i, e := range %s {\n%s}\n}",
		result, value, result, value, value, renderIntegerText(array.ElemType.Type, element, data.fieldContext, "e", result+"[i]", "eWire", fail))
}

// integerWireText formats a value already held in the wire's canonical carrier.
func integerWireText(wire expr.DataType, value string) string {
	switch wire.Kind() {
	case expr.IntKind:
		return "strconv.Itoa(" + value + ")"
	case expr.Int32Kind:
		return "strconv.FormatInt(int64(" + value + "), 10)"
	case expr.Int64Kind:
		return "strconv.FormatInt(" + value + ", 10)"
	case expr.UIntKind, expr.UInt32Kind:
		return "strconv.FormatUint(uint64(" + value + "), 10)"
	case expr.UInt64Kind:
		return "strconv.FormatUint(" + value + ", 10)"
	default:
		panic(fmt.Sprintf("unsupported integer wire %s", wire.Name())) // bug
	}
}

// integerDecodeCheck rejects a parsed wire integer that the decoded value's
// physical storage cannot represent, before the existing assignment. The
// parser already enforced the wire domain; parsed holds an int64 or uint64.
func integerDecodeCheck(data any, parsed string) string {
	scalar, ok := data.(*scalarDecodeData)
	if !ok || scalar.fieldAttribute == nil || scalar.IsTextUnmarshaler {
		return ""
	}
	wire, _ := transportUnderlyingType(scalar.Type)
	conversion, integer, err := representation.ClassifyIntegerConversion(&expr.AttributeExpr{Type: wire}, scalar.fieldAttribute)
	if err != nil {
		panic(err) // bug: admission accepted non-integer storage for an integer wire
	}
	if !integer || !conversion.Checked {
		return ""
	}
	targetRef := strings.TrimPrefix(scalar.TypeRef, "*")
	parsedRef := "int64"
	if wire.Kind() >= expr.UIntKind {
		parsedRef = "uint64"
	}
	var code strings.Builder
	if conversion.TargetOpaque {
		code.WriteString(representation.IntegerCapabilityAssertion(targetRef))
	}
	fmt.Fprintf(&code, "if %s {\nerr = loom.MergeErrors(err, loom.InvalidFieldTypeError(%q, %s, %q))\n}\n",
		representation.IntegerLossCondition(parsedRef, parsed, targetRef+"("+parsed+")"), scalar.Name, scalar.Locals.Raw, "integer")
	return code.String()
}

// integerElementText declares result as the wire text of a client query map
// key, map value or slice element selected by path ("", "key", "elem" or
// "elem.elem") from the value's actual physical attribute. It returns "" for
// non-integer elements and for native unnamed storage, whose existing text is
// already exact.
func integerElementText(data *AttributeData, path, result, value string) string {
	if data.fieldAttribute == nil || data.fieldContext == nil || data.IsTextUnmarshaler {
		return ""
	}
	wire := &expr.AttributeExpr{Type: data.Type}
	physical := data.fieldAttribute
	if path != "" {
		for _, step := range strings.Split(path, ".") {
			wire, physical = integerElementStep(wire, step), integerElementStep(physical, step)
			if wire == nil || physical == nil {
				return ""
			}
		}
	}
	base, _ := transportUnderlyingType(wire.Type)
	if !representation.IsIntegerKind(base) || nativeIntegerStorage(physical) {
		return ""
	}
	return strings.TrimSuffix(renderIntegerTextAssign(base, physical, data.fieldContext, value, result, result+"Wire", "", ":="), "\n")
}

// integerElementStep selects a map key or a map value/array element attribute.
func integerElementStep(attribute *expr.AttributeExpr, step string) *expr.AttributeExpr {
	if mapping := expr.AsMap(attribute.Type); mapping != nil {
		if step == "key" {
			return mapping.KeyType
		}
		return mapping.ElemType
	}
	if array := expr.AsArray(attribute.Type); array != nil && step == "elem" {
		return array.ElemType
	}
	return nil
}

// integerCarrier exposes the analyzed attribute data of any transport value
// that embeds it, so decoder partials can reach the actual storage.
func (a *AttributeData) integerCarrier() *AttributeData {
	return a
}

// integerSliceAssign assigns a parsed wire integer to the current element of a
// decoded slice held in its actual element storage. A value that storage
// cannot represent is merged as an invalid field type before assignment. It
// returns "" for non-integer and native unnamed elements, whose existing
// assignment is exact.
func integerSliceAssign(data any, parsed, expected string) string {
	carrier, ok := data.(interface{ integerCarrier() *AttributeData })
	if !ok {
		return ""
	}
	value := carrier.integerCarrier()
	array := expr.AsArray(value.Type)
	if array == nil || value.fieldContext == nil {
		return ""
	}
	element := array.ElemType
	if !representation.IsIntegerKind(element.Type) || nativeIntegerStorage(element) {
		return ""
	}
	base, _ := transportUnderlyingType(element.Type)
	return renderIntegerDecode(base, element, value.fieldContext, parsed, value.VarName+"[i]", value.Name, value.Locals.Raw, expected)
}

// renderIntegerDecode converts a parsed int64 or uint64 wire value to the
// target's physical storage, merging an invalid field type for a value the
// storage cannot represent.
func renderIntegerDecode(wire expr.DataType, physical *expr.AttributeExpr, ctx *codegen.AttributeContext, parsed, target, name, raw, expected string) string {
	conversion, _, err := representation.ClassifyIntegerConversion(&expr.AttributeExpr{Type: wire}, physical)
	if err != nil {
		panic(err) // bug: admission accepted non-integer storage for an integer wire
	}
	ref := ctx.Scope.Ref(physical, ctx.Pkg(physical))
	parsedRef := "int64"
	if wire.Kind() >= expr.UIntKind {
		parsedRef = "uint64"
	}
	var code strings.Builder
	if conversion.Checked {
		if conversion.TargetOpaque {
			code.WriteString(representation.IntegerCapabilityAssertion(ref))
		}
		fmt.Fprintf(&code, "if %s {\nerr = loom.MergeErrors(err, loom.InvalidFieldTypeError(%q, %s, %q))\n}\n",
			representation.IntegerLossCondition(parsedRef, parsed, ref+"("+parsed+")"), name, raw, expected)
	}
	fmt.Fprintf(&code, "%s = %s(%s)", target, ref, parsed)
	return code.String()
}

// integerPointerAssign assigns a parsed int64 or uint64 to an optional decoded
// value whose actual storage has a different underlying type, where a pointer
// conversion cannot compile. It returns "" when the existing assignment is
// exact.
func integerPointerAssign(data any, parsed string) string {
	scalar, ok := data.(*scalarDecodeData)
	if !ok || !scalar.Pointer || scalar.fieldAttribute == nil || scalar.IsTextUnmarshaler {
		return ""
	}
	wire, _ := transportUnderlyingType(scalar.Type)
	parsedRef := "int64"
	if wire.Kind() >= expr.UIntKind {
		parsedRef = "uint64"
	}
	if storage := representation.IntegerStorage(scalar.fieldAttribute); storage != nil && storage.Name() == parsedRef {
		return ""
	}
	return fmt.Sprintf("pv := %s(%s)\n%s = &pv", strings.TrimPrefix(scalar.TypeRef, "*"), parsed, scalar.VarName)
}

// renderIntegerDecimal returns the faithful decimal text of an integer value
// held in its physical storage, writing any statements it needs to code. It
// derives the digits from the numeric value itself, so fmt.Formatter and
// fmt.Stringer implementations never alter the text, and it performs no wire
// range check. Opaque external storage must have an integer underlying type;
// its sign selects the lossless int64 or uint64 carrier.
func renderIntegerDecimal(code *strings.Builder, physical *expr.AttributeExpr, ref, value, local string) string {
	if storage := representation.IntegerStorage(physical); storage != nil {
		if storage.Info()&types.IsUnsigned != 0 {
			return "strconv.FormatUint(uint64(" + value + "), 10)"
		}
		return "strconv.FormatInt(int64(" + value + "), 10)"
	}
	if local == "" {
		panic("opaque integer text of " + value + " has no allocated local") // bug
	}
	code.WriteString(representation.IntegerCapabilityAssertion(ref))
	fmt.Fprintf(code, "%s := strconv.FormatUint(uint64(%s), 10)\nif %s < 0 {\n%s = strconv.FormatInt(int64(%s), 10)\n}\n", local, value, value, local, value)
	return local
}
