//nolint:errcheck // Generator helpers write only to in-memory buffers/builders.
package codegen

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

func transformArray(source, target *expr.Array, sourceVar, targetVar string, newVar bool, targetPtr bool, sourcePtr bool, ta *transformAttrs) (string, error) {
	elem := target.ElemType
	if ta.proto {
		elem = unAlias(elem)
	}
	targetRef := ta.TargetCtx.Scope.Ref(elem, ta.TargetCtx.Pkg(elem))
	code, sourceVar, targetVar, newVar := transformArraySetup(sourceVar, targetVar, newVar, ta)
	src, tgt, err := transformArrayElementAttrs(source.ElemType, elem, ta)
	if err != nil {
		return "", err
	}
	buf, targetElemVar, loopVar, rangeOn := transformArrayInitBuffer(targetVar, sourceVar, targetRef, targetPtr, sourcePtr, newVar)
	elemCode, err := transformAttribute(src, tgt, "val", targetElemVar, false, ta)
	if err != nil {
		return "", err
	}
	writeTransformArrayLoop(&buf, loopVar, rangeOn, transformArrayValueVar(src), elemCode)
	if targetPtr {
		fmt.Fprintf(&buf, "%s = &arr%s\n", targetVar, loopVar)
	}
	return code + buf.String(), nil
}

func transformArraySetup(sourceVar, targetVar string, newVar bool, ta *transformAttrs) (string, string, string, bool) {
	var code string
	if ta.targetInit != "" {
		assign := "="
		if newVar {
			assign = ":="
		}
		code = renderJenLine(exprCode(targetVar).Op(assign).Op("&").Id(ta.targetInit).Values())
		ta.targetInit = ""
	}
	if !ta.wrapped {
		return code, sourceVar, targetVar, newVar
	}
	if ta.proto {
		targetVar += ".Field"
		newVar = false
	} else {
		sourceVar += ".Field"
	}
	ta.wrapped = false
	return code, sourceVar, targetVar, newVar
}

func transformArrayElementAttrs(src, tgt *expr.AttributeExpr, ta *transformAttrs) (*expr.AttributeExpr, *expr.AttributeExpr, error) {
	if err := codegen.IsCompatible(src.Type, tgt.Type, "[0]", "[0]"); err == nil {
		return src, tgt, nil
	}
	if ta.proto {
		ta.targetInit = ta.TargetCtx.Scope.Name(tgt, ta.TargetCtx.Pkg(tgt), ta.TargetCtx.Pointer, ta.TargetCtx.UseDefault)
		tgt = unwrapAttr(expr.DupAtt(tgt))
	} else {
		src = unwrapAttr(expr.DupAtt(src))
	}
	ta.wrapped = true
	if err := codegen.IsCompatible(src.Type, tgt.Type, "[0]", "[0]"); err != nil {
		return nil, nil, err
	}
	return src, tgt, nil
}

func transformArrayInitBuffer(targetVar, sourceVar, targetRef string, targetPtr, sourcePtr, newVar bool) (bytes.Buffer, string, string, string) {
	var buf bytes.Buffer
	loopVar := string(rune(105 + strings.Count(targetVar, "[")))
	rangeOn := sourceVar
	if sourcePtr {
		rangeOn = "*" + rangeOn
	}
	assign := "="
	if newVar {
		assign = ":="
	}
	targetElemVar := targetVar + "[" + loopVar + "]"
	if targetPtr {
		arrayVar := "arr" + loopVar
		targetElemVar = arrayVar + "[" + loopVar + "]"
		fmt.Fprintf(&buf, "%s := make([]%s, len(%s))\n", arrayVar, targetRef, rangeOn)
		return buf, targetElemVar, loopVar, rangeOn
	}
	fmt.Fprintf(&buf, "%s %s make([]%s, len(%s))\n", targetVar, assign, targetRef, rangeOn)
	return buf, targetElemVar, loopVar, rangeOn
}

func transformArrayValueVar(src *expr.AttributeExpr) string {
	if obj := expr.AsObject(src.Type); obj != nil && len(*obj) == 0 {
		return ""
	}
	return "val"
}

func writeTransformArrayLoop(buf *bytes.Buffer, loopVar, rangeOn, valVar, elemCode string) {
	if valVar != "" {
		fmt.Fprintf(buf, "for %s, %s := range %s {\n", loopVar, valVar, rangeOn)
	} else {
		fmt.Fprintf(buf, "for %s := range %s {\n", loopVar, rangeOn)
	}
	fmt.Fprint(buf, codegen.Indent(elemCode, "\t"))
	fmt.Fprint(buf, "}\n")
}

// transformMap returns the code to transform source attribute of map
// type to target attribute of map type. It returns an error if source
// and target are not compatible for transformation.
func transformMap(source, target *expr.Map, sourceVar, targetVar string, newVar bool, targetPtr bool, ta *transformAttrs) (string, error) {
	// Target map key cannot be nested in protocol buffers. So no need to worry
	// about unwrapping.
	if err := codegen.IsCompatible(source.KeyType.Type, target.KeyType.Type, sourceVar+"[key]", targetVar+"[key]"); err != nil {
		return "", err
	}
	targetKeyRef, targetElemRef := transformMapTypeRefs(target, ta)
	code, sourceVar, targetVar, newVar := transformMapTargetSetup(sourceVar, targetVar, newVar, ta)
	src, tgt, err := transformMapElementAttrs(source, target, ta)
	if err != nil {
		return "", err
	}
	loopVar, suffix := transformMapLoopNames(target)
	mapVar, initCode := transformMapInit(targetVar, sourceVar, targetPtr, targetKeyRef, targetElemRef, newVar, loopVar)
	var buf bytes.Buffer
	fmt.Fprint(&buf, initCode)
	fmt.Fprintf(&buf, "for key, val := range %s {\n", sourceVar)
	keyCode, err := transformAttribute(source.KeyType, target.KeyType, "key", "tk", true, ta)
	if err != nil {
		return "", err
	}
	fmt.Fprint(&buf, codegen.Indent(keyCode, "\t"))
	elemTmp := "tv" + suffix
	elemCode, err := transformAttribute(src, tgt, "val", elemTmp, true, ta)
	if err != nil {
		return "", err
	}
	fmt.Fprint(&buf, codegen.Indent(elemCode, "\t"))
	fmt.Fprintf(&buf, "\t%s[tk] = %s\n", mapVar, elemTmp)
	fmt.Fprint(&buf, "}\n")
	if targetPtr {
		fmt.Fprintf(&buf, "%s = &%s\n", targetVar, mapVar)
	}
	return code + buf.String(), nil
}

func transformMapTypeRefs(target *expr.Map, ta *transformAttrs) (string, string) {
	kt := target.KeyType
	et := target.ElemType
	if ta.proto {
		kt = unAlias(kt)
		et = unAlias(et)
	}
	return ta.TargetCtx.Scope.Ref(kt, ta.TargetCtx.Pkg(kt)), ta.TargetCtx.Scope.Ref(et, ta.TargetCtx.Pkg(et))
}

func transformMapTargetSetup(sourceVar, targetVar string, newVar bool, ta *transformAttrs) (string, string, string, bool) {
	code := ""
	if ta.targetInit != "" {
		assign := "="
		if newVar {
			assign = ":="
		}
		code = renderJenLine(exprCode(targetVar).Op(assign).Op("&").Id(ta.targetInit).Values())
		ta.targetInit = ""
	}
	if !ta.wrapped {
		return code, sourceVar, targetVar, newVar
	}
	if ta.proto {
		targetVar += ".Field"
		newVar = false
	} else {
		sourceVar += ".Field"
	}
	ta.wrapped = false
	return code, sourceVar, targetVar, newVar
}

func transformMapElementAttrs(source, target *expr.Map, ta *transformAttrs) (*expr.AttributeExpr, *expr.AttributeExpr, error) {
	src := source.ElemType
	tgt := target.ElemType
	if err := codegen.IsCompatible(src.Type, tgt.Type, "[*]", "[*]"); err == nil {
		return src, tgt, nil
	}
	if ta.proto {
		ta.targetInit = ta.TargetCtx.Scope.Name(tgt, ta.TargetCtx.Pkg(tgt), ta.TargetCtx.Pointer, ta.TargetCtx.UseDefault)
		tgt = unwrapAttr(expr.DupAtt(tgt))
	} else {
		src = unwrapAttr(expr.DupAtt(src))
	}
	ta.wrapped = true
	if err := codegen.IsCompatible(src.Type, tgt.Type, "[*]", "[*]"); err != nil {
		return nil, nil, err
	}
	return src, tgt, nil
}

func transformMapLoopNames(target *expr.Map) (string, string) {
	if depth := codegen.MapDepth(target); depth > 0 {
		loopVar := string(rune(97 + depth))
		return loopVar, loopVar
	}
	return "a", ""
}

func transformMapInit(targetVar, sourceVar string, targetPtr bool, targetKeyRef, targetElemRef string, newVar bool, loopVar string) (string, string) {
	assign := "="
	if newVar {
		assign = ":="
	}
	if targetPtr {
		mapVar := "m" + loopVar
		return mapVar, renderJenLine(exprCode(mapVar).Op(":=").Make(exprCode("map["+targetKeyRef+"]"+targetElemRef), jen.Len(exprCode(sourceVar))))
	}
	return targetVar, renderJenLine(exprCode(targetVar).Op(assign).Make(exprCode("map["+targetKeyRef+"]"+targetElemRef), jen.Len(exprCode(sourceVar))))
}

// transformUnionToProto returns the code to transform an attribute of type
// union from protobuf to Loom. It returns an error if source and target are not
// compatible for transformation.
func transformUnionToProto(source, target *expr.AttributeExpr, sourceVar, targetVar string, sourcePtr bool, ta *transformAttrs) (string, error) {
	if err := codegen.IsCompatible(source.Type, target.Type, sourceVar, targetVar); err != nil {
		return "", err
	}
	tdata := transformUnionData(source, target, ta)
	cases := make([]map[string]any, 0, len(tdata.SourceValues))
	for i, sv := range tdata.SourceValues {
		tv := tdata.TargetValues[i]
		fieldName := ta.TargetCtx.Scope.Field(tv.Attribute, tv.Name, true)
		cases = append(cases, map[string]any{
			"typeTag":           sv.Name,
			"sourceFieldName":   codegen.Goify(sv.Name, true),
			"sourceAttr":        sv.Attribute,
			"targetAttr":        tv.Attribute,
			"targetWrapperType": ta.message + "_" + fieldName,
			"targetFieldName":   fieldName,
		})
	}

	var buf bytes.Buffer
	switchTarget := sourceVar
	if sourcePtr {
		switchTarget = "(*" + sourceVar + ")"
	}
	fmt.Fprintf(&buf, "switch string(%s.Kind()) {\n", switchTarget)
	for _, c := range cases {
		fmt.Fprintf(&buf, "case %q:\n", c["typeTag"])
		fmt.Fprintf(&buf, "\tactual, _ := %s.As%s()\n", switchTarget, c["sourceFieldName"])
		val := convertType(c["sourceAttr"].(*expr.AttributeExpr), c["targetAttr"].(*expr.AttributeExpr), false, false, "actual", ta)
		fmt.Fprintf(&buf, "\t%s = &%s{%s: %s}\n", targetVar, c["targetWrapperType"], c["targetFieldName"], val)
	}
	fmt.Fprint(&buf, "}\n")
	return buf.String(), nil
}

// transformUnionFromProto returns the code to transform an attribute of type
// union from protobuf to Loom. It returns an error if source and target are not
// compatible for transformation.
func transformUnionFromProto(source, target *expr.AttributeExpr, sourceVar, targetVar string, ta *transformAttrs) (string, error) {
	if err := codegen.IsCompatible(source.Type, target.Type, sourceVar, targetVar); err != nil {
		return "", err
	}
	tdata := transformUnionData(source, target, ta)
	cases := make([]map[string]any, 0, len(tdata.SourceValues))
	for i, sv := range tdata.SourceValues {
		tv := tdata.TargetValues[i]
		sourceFieldName := ta.SourceCtx.Scope.Field(sv.Attribute, sv.Name, true)
		targetFieldName := codegen.Goify(tv.Name, true)
		cases = append(cases, map[string]any{
			"sourceValueTypeRef": tdata.SourceValueTypeRefs[i],
			"sourceFieldName":    sourceFieldName,
			"sourceAttr":         sv.Attribute,
			"targetAttr":         tv.Attribute,
			"targetFieldName":    targetFieldName,
		})
	}
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "switch val := %s.(type) {\n", sourceVar)
	for _, c := range cases {
		fmt.Fprintf(&buf, "case %s:\n", c["sourceValueTypeRef"])
		field := "val." + c["sourceFieldName"].(string)
		tmp := convertType(c["sourceAttr"].(*expr.AttributeExpr), c["targetAttr"].(*expr.AttributeExpr), false, false, field, ta)
		fmt.Fprint(&buf, "\t{\n")
		fmt.Fprintf(&buf, "\t\tu := %s\n", targetVar)
		fmt.Fprintf(&buf, "\t\tu.Set%s(%s)\n", c["targetFieldName"], tmp)
		fmt.Fprintf(&buf, "\t\t%s = u\n", targetVar)
		fmt.Fprint(&buf, "\t}\n")
	}
	fmt.Fprint(&buf, "}\n")
	return buf.String(), nil
}

// convertType produces code to initialize a target type from a source type
// held by sourceVar.
func convertType(src, tgt *expr.AttributeExpr, srcPtr bool, tgtPtr bool, srcVar string, ta *transformAttrs) string {
	if expr.IsAlias(src.Type) || expr.IsAlias(tgt.Type) {
		srcp, tgtp := unAlias(src), unAlias(tgt)
		if srcp.Type == tgtp.Type {
			if ta.proto {
				return convertPrimitiveToProto(src, tgtp, srcPtr, tgtPtr, srcVar, ta)
			}
			return convertPrimitiveFromProto(srcp, tgt, srcPtr, tgtPtr, srcVar, ta)
		}
		return renderJen(jen.Id(transformHelperName(src, tgt, ta)).Call(exprCode(srcVar)))
	}

	if _, ok := src.Type.(expr.UserType); ok {
		return renderJen(jen.Id(transformHelperName(src, tgt, ta)).Call(exprCode(srcVar)))
	}

	srcType, _ := codegen.GetMetaType(src)
	tgtType, _ := codegen.GetMetaType(tgt)
	if srcType == "" && tgtType == "" && (src.Type != expr.Int) && (src.Type != expr.UInt) && (src.Type != expr.Any) {
		// Nothing to do, except for Any type which needs special conversion
		return srcVar
	}

	if ta.proto {
		return convertPrimitiveToProto(src, tgt, srcPtr, tgtPtr, srcVar, ta)
	}
	return convertPrimitiveFromProto(src, tgt, srcPtr, tgtPtr, srcVar, ta)
}

const convertGoAnyToProtobufValueFunc = `func() *structpb.Value {
	// Convert Go any to protobuf Value directly
	if %SRC% == nil {
		return structpb.NewNullValue()
	}
	value, err := structpb.NewValue(%SRC%)
	if err != nil {
		panic(%FMT_SPRINTF%("failed to convert value to structpb.Value: %v", err))
	}
	return value
}()`

const convertProtobufValueToGoAnyFunc = `func() any {
	// Convert protobuf Value to Go any directly
	if %SRC% != nil {
		return %SRC%.AsInterface()
	}
	return nil
}()`

// convertPrimitive returns the code to convert a primitive type from one
// representation to another.
// NOTE: For Int and UInt kinds, protocol buffer Go compiler generates
// int32 and uint32 respectively whereas Loom generates int and uint.
func convertPrimitiveToProto(_, tgt *expr.AttributeExpr, srcPtr, _ bool, srcVar string, _ *transformAttrs) string {
	// Special handling for Any type conversion to google.protobuf.Value
	if tgt.Type.Kind() == expr.AnyKind {
		if srcPtr {
			srcVar = "*" + srcVar
		}

		return strings.NewReplacer("%SRC%", srcVar, "%FMT_SPRINTF%", "fmt."+"Sprintf").Replace(convertGoAnyToProtobufValueFunc)
	}

	tgtType := protoBufNativeGoTypeName(tgt.Type)
	if srcPtr {
		srcVar = "*" + srcVar
	}
	return renderJen(exprCode(tgtType).Call(exprCode(srcVar)))
}

func convertPrimitiveFromProto(_, tgt *expr.AttributeExpr, srcPtr, _ bool, srcVar string, ta *transformAttrs) string {
	// Special handling for Any type conversion from google.protobuf.Value
	if tgt.Type.Kind() == expr.AnyKind {
		if srcPtr {
			srcVar = "*" + srcVar
		}

		return strings.NewReplacer("%SRC%", srcVar).Replace(convertProtobufValueToGoAnyFunc)
	}

	tgtType, _ := codegen.GetMetaType(tgt)
	if tgtType == "" {
		tgtType = ta.TargetCtx.Scope.Ref(tgt, ta.TargetCtx.Pkg(tgt))
	}
	if srcPtr {
		srcVar = "*" + srcVar
	}
	return renderJen(exprCode(tgtType).Call(exprCode(srcVar)))
}

// transformUnionData returns data needed by both transformUnion functions.
func transformUnionData(source, target *expr.AttributeExpr, ta *transformAttrs) *unionData {
	src := expr.AsUnion(source.Type)
	tgt := expr.AsUnion(target.Type)
	srcValues := make([]*expr.NamedAttributeExpr, len(src.Values))
	copy(srcValues, src.Values)
	tgtValues := make([]*expr.NamedAttributeExpr, len(tgt.Values))
	copy(tgtValues, tgt.Values)
	sourceValueTypeRefs, targetWrapperRefs := transformUnionTypeRefs(source, target, ta, src, tgt)
	return &unionData{
		Source:              src,
		Target:              tgt,
		SourceValues:        srcValues,
		TargetValues:        tgtValues,
		SourceValueTypeRefs: sourceValueTypeRefs,
		TargetWrapperRefs:   targetWrapperRefs,
	}
}

func transformUnionTypeRefs(source, target *expr.AttributeExpr, ta *transformAttrs, src, tgt *expr.Union) ([]string, []string) {
	sourceValueTypeRefs := make([]string, len(src.Values))
	targetWrapperRefs := make([]string, len(src.Values))
	if ta.proto {
		buildProtoUnionTypeRefs(source, ta, src, sourceValueTypeRefs)
		return sourceValueTypeRefs, targetWrapperRefs
	}
	buildSourceUnionTypeRefs(ta, src, sourceValueTypeRefs)
	buildTargetUnionWrapperRefs(target, ta, tgt, targetWrapperRefs)
	return sourceValueTypeRefs, targetWrapperRefs
}

func buildProtoUnionTypeRefs(source *expr.AttributeExpr, ta *transformAttrs, src *expr.Union, sourceValueTypeRefs []string) {
	unionPkg := ta.SourceCtx.Pkg(source)
	samePkg, commonPkg := unionCommonPkg(src.Values)
	for i, v := range src.Values {
		if _, ok := v.Attribute.Type.(expr.UserType); ok {
			sourceValueTypeRefs[i] = ta.SourceCtx.Scope.Ref(v.Attribute, ta.SourceCtx.Pkg(v.Attribute))
			continue
		}
		w := codegen.Goify(src.Name(), true) + codegen.Goify(v.Name, true)
		pkg := unionPkg
		if samePkg && commonPkg != "" {
			pkg = commonPkg
		}
		if pkg != "" {
			sourceValueTypeRefs[i] = pkg + "." + w
		} else {
			sourceValueTypeRefs[i] = w
		}
	}
}

func buildSourceUnionTypeRefs(ta *transformAttrs, src *expr.Union, sourceValueTypeRefs []string) {
	for i, v := range src.Values {
		fieldName := ta.SourceCtx.Scope.Field(v.Attribute, v.Name, true)
		sourceValueTypeRefs[i] = ta.message + "_" + fieldName
	}
}

func buildTargetUnionWrapperRefs(target *expr.AttributeExpr, ta *transformAttrs, tgt *expr.Union, targetWrapperRefs []string) {
	unionPkg := ta.TargetCtx.Pkg(target)
	samePkg, commonPkg := unionCommonPkg(tgt.Values)
	for i, tv := range tgt.Values {
		if _, ok := tv.Attribute.Type.(expr.UserType); ok {
			continue
		}
		w := codegen.Goify(tgt.Name(), true) + codegen.Goify(tv.Name, true)
		pkg := unionPkg
		if samePkg && commonPkg != "" {
			pkg = commonPkg
		}
		if pkg != "" {
			targetWrapperRefs[i] = pkg + "." + w
		} else {
			targetWrapperRefs[i] = w
		}
	}
}

func unionCommonPkg(values []*expr.NamedAttributeExpr) (bool, string) {
	samePkg := true
	commonPkg := ""
	for _, v := range values {
		ut, ok := v.Attribute.Type.(expr.UserType)
		if !ok {
			return false, ""
		}
		loc := codegen.UserTypeLocation(ut)
		if loc == nil {
			return false, ""
		}
		if commonPkg == "" {
			commonPkg = loc.PackageName()
			continue
		}
		if commonPkg != loc.PackageName() {
			return false, ""
		}
	}
	return samePkg, commonPkg
}

// transformAttributeHelpers returns the Go transform functions and their definitions
// that may be used in code produced by Transform. It returns an error if source and
// target are incompatible (different types, fields of different type etc).
//
// source, target are the source and target attributes used in transformation
//
// ta is the transform attributes
//
// seen keeps track of generated transform functions to avoid recursion
