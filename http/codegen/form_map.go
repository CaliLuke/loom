package codegen

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
	"github.com/CaliLuke/loom/internal/httpencoding"
)

// formMapElement retains the dynamic name while sharing ordinary field codecs.
func formMapElement(body *TypeData, contract *expr.HTTPFormEncoding) *Element {
	mapping := expr.AsMap(body.attribute.Type)
	context := body.context.Dup()
	representation.ApplyLayout(context, mapping.ElemType)
	ref := context.CollectionElementRef(mapping.ElemType, false)
	return &Element{
		attribute: mapping.ElemType, context: context, location: "form", wireNameExpr: "formKey",
		HTTPName: "map entry", Codec: *contract.Element,
		AttributeData: &AttributeData{Type: mapping.ElemType.Type, FieldType: mapping.ElemType.Type,
			Required: true, Pointer: strings.HasPrefix(ref, "*"),
			TypeName: context.Scope.Name(mapping.ElemType, context.Pkg(mapping.ElemType), &codegen.AttributeContext{}), TypeRef: ref},
	}
}

func renderFormMapEncode(body *TypeData, contract *expr.HTTPFormEncoding, value string) string {
	var out strings.Builder
	mapping := expr.AsMap(body.attribute.Type)
	element := formMapElement(body, contract)
	out.WriteString("parameterForm := make(map[string][]string)\nvar parameterBuilder loomhttp.RawPairsBuilder\n")
	fmt.Fprintf(&out, "formKeys := make([]string, 0, len(%s))\nformNativeKeys := make(map[string]%s, len(%s))\nfor mapKey := range %s {\nformKey := %s\nif _, duplicate := formNativeKeys[formKey]; duplicate {\nreturn \"\", fmt.Errorf(\"form map contains duplicate encoded keys\")\n}\nformNativeKeys[formKey] = mapKey\nformKeys = append(formKeys, formKey)\n}\nsort.Strings(formKeys)\nfor _, formKey := range formKeys {\nformValue := %s[formNativeKeys[formKey]]\n", value, body.context.Scope.Ref(mapping.KeyType, body.context.Pkg(mapping.KeyType)), value, value, parameterScalarText(mapping.KeyType, mapping.KeyType, "mapKey"), value)
	out.WriteString("if fieldError := func() error {\n")
	switch {
	case element.Codec.Style == httpencoding.JSON:
		fmt.Fprintf(&out, "if present, err := loomhttp.EncodeJSONField(parameterForm, formKey, formValue, %s); err != nil {\nreturn err\n} else if !present {\nreturn fmt.Errorf(\"form map entry has no encoded value\")\n}\n", formApplicationLiteral(element.Codec.Application))
	case expr.AsUnion(element.Type) != nil:
		if element.Codec.Style == httpencoding.Content {
			out.WriteString(renderFormContentUnionEncode(element, "formValue"))
		} else {
			out.WriteString(renderTextUnionEncode(element, "formValue", "form"))
		}
	default:
		out.WriteString(renderParameterValue(element.attribute, element.attribute, element.context, "formValue", "parameterValue", false))
		out.WriteString("if !parameterValue.Present {\nreturn fmt.Errorf(\"form map entry is absent\")\n}\n")
		if parameterEmptyHasNoKey(element.Codec) {
			out.WriteString(renderParameterKeyCheck(element.Codec, "map entry", ""))
		}
		fmt.Fprintf(&out, "if err := parameterBuilder.Add(formKey, %s, parameterValue, %s); err != nil {\nreturn err\n}\n", parameterCodecLiteral(element.Codec), parameterNamespaceLiteral(element))
	}
	out.WriteString("return nil\n}(); fieldError != nil {\nreturn \"\", fieldError\n}\n}\nreturn parameterBuilder.Encode(parameterForm), nil\n")
	return out.String()
}

func renderFormMapDecode(body *TypeData, contract *expr.HTTPFormEncoding, target string) string {
	var out strings.Builder
	mapping := expr.AsMap(body.attribute.Type)
	element := formMapElement(body, contract)
	fmt.Fprintf(&out, "%s = make(%s, len(parameterForm))\nformKeys := make([]string, 0, len(parameterForm))\nfor formKey := range parameterForm {\nformKeys = append(formKeys, formKey)\n}\nsort.Strings(formKeys)\nfor _, formKey := range formKeys {\nvar mapKey %s\nvar formValue %s\n", target, body.context.Scope.Ref(body.attribute, body.context.Pkg(body.attribute)), body.context.Scope.Ref(mapping.KeyType, body.context.Pkg(mapping.KeyType)), element.TypeRef)
	out.WriteString(parameterScalarAssignment(mapping.KeyType, mapping.KeyType, body.context, "formKey", "mapKey", "form map key", false))
	switch {
	case element.Codec.Style == httpencoding.JSON:
		fmt.Fprintf(&out, "if _, fieldError := loomhttp.DecodeJSONField(parameterForm, formKey, &formValue, %t, %s); fieldError != nil {\nreturn fieldError\n}\n", expr.AllowsNull(element.attribute), formApplicationLiteral(element.Codec.Application))
	case expr.AsUnion(element.Type) != nil:
		matcher := "matchTextUnion"
		if element.Codec.Style == httpencoding.Content {
			matcher = "matchFormContent"
			out.WriteString(renderFormContentUnionMatcher(element))
		} else {
			out.WriteString("parameterPairs := parameterFormPairs\n")
			out.WriteString(renderTextUnionMatcher(element, "form"))
		}
		fmt.Fprintf(&out, "matched, present, _, fieldError := %s()\nif fieldError != nil {\nreturn fieldError\n}\nif !present {\nreturn fmt.Errorf(\"form map entry is absent\")\n}\nformValue = matched\n", matcher)
	default:
		fmt.Fprintf(&out, "parameterValue, fieldError := loomhttp.DecodePairsParam(parameterFormPairs, formKey, %s, %s, %s)\nif fieldError != nil {\nreturn fieldError\n}\n", parameterCodecLiteral(element.Codec), parameterKindLiteral(element.Codec), parameterNamespaceLiteral(element))
		out.WriteString(renderParameterAssignment(element.attribute, element.attribute, element.context, "parameterValue", "formValue", "map entry", false))
	}
	fmt.Fprintf(&out, "if _, exists := %s[mapKey]; exists {\nreturn fmt.Errorf(\"form map contains duplicate converted keys\")\n}\n%s[mapKey] = formValue\n}\n", target, target)
	return out.String()
}

func formMapBody(request *RequestData) bool {
	return request.form != nil && request.form.Element != nil
}
