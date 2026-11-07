package codegen

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/httpencoding"
)

func renderStyledFormEncode(request *RequestData) string {
	return "{\nparameterEncoded, parameterError := func() (string, error) {\n" + renderFormFieldsEncode(request.ClientBody, request.OrdinaryFormFields, request.StyledFormFields, "body") + "}()\nif parameterError != nil {\nreturn parameterError\n}\nif parameterError := loomhttp.SetRawFormRequest(req, parameterEncoded); parameterError != nil {\nreturn parameterError\n}\n}\n"
}

func renderFormFieldsEncode(body *TypeData, ordinary, styled []*FormFieldData, target string) string {
	var code strings.Builder
	code.WriteString("parameterForm := make(map[string][]string)\nvar parameterBuilder loomhttp.RawPairsBuilder\n")
	for _, field := range ordinary {
		if field.Codec.Style == httpencoding.JSON {
			code.WriteString(renderJSONFormFieldEncode(body, field, target))
			continue
		}
		if field.Required && parameterEmptyHasNoKey(field.Codec) {
			key, physical := body.attribute.FindAttribute(field.AttributeName)
			code.WriteString("{\n")
			code.WriteString(renderParameterValue(field.Attribute, physical, body.context, target+"."+field.FieldName, "parameterValue", body.context.IsPrimitivePointer(key, body.attribute)))
			code.WriteString(renderParameterKeyCheck(field.Codec, field.HTTPName, "\"\", "))
			code.WriteString("}\n")
		}
		fmt.Fprintf(&code, "if _, parameterError := loomhttp.EncodeFormValue(parameterForm, %q, %s.%s); parameterError != nil {\nreturn \"\", parameterError\n}\n", field.HTTPName, target, field.FieldName)
	}
	for _, field := range styled {
		if expr.AsUnion(field.Attribute.Type) != nil {
			element := formUnionElement(body, field)
			code.WriteString("if parameterError := func() error {\n")
			if field.Codec.Style == httpencoding.Content {
				code.WriteString(renderFormContentUnionEncode(element, target))
			} else {
				code.WriteString(renderTextUnionEncode(element, target, "form"))
			}
			code.WriteString("return nil\n}(); parameterError != nil {\nreturn \"\", parameterError\n}\n")
			continue
		}
		key, physical := body.attribute.FindAttribute(field.AttributeName)
		if physical == nil {
			panic("missing styled form carrier " + field.AttributeName)
		}
		code.WriteString("{\n")
		code.WriteString(renderParameterValue(field.Attribute, physical, body.context, target+"."+field.FieldName, "parameterValue", body.context.IsPrimitivePointer(key, body.attribute)))
		if field.Required && parameterEmptyHasNoKey(field.Codec) {
			code.WriteString(renderParameterKeyCheck(field.Codec, field.HTTPName, "\"\", "))
		}
		element := &Element{Namespace: field.Namespace}
		fmt.Fprintf(&code, "if parameterError := parameterBuilder.Add(%q, %s, parameterValue, %s); parameterError != nil {\nreturn \"\", parameterError\n}\n}\n", field.HTTPName, parameterCodecLiteral(field.Codec), parameterNamespaceLiteral(element))
	}
	code.WriteString("return parameterBuilder.Encode(parameterForm), nil\n")
	return code.String()
}

func renderStyledFormDecode(request *RequestData) string {
	code := renderFormFieldsDecode(request.ServerBody, request.OrdinaryFormFields, request.StyledFormFields, "body")
	if formFieldsNeedValues(request.OrdinaryFormFields, request.StyledFormFields) {
		code = "parameterForm := r.PostForm\n" + code
	}
	return code
}

func renderFormFieldsDecode(body *TypeData, ordinary, styled []*FormFieldData, target string) string {
	var code strings.Builder
	for _, field := range ordinary {
		if field.Codec.Style == httpencoding.JSON {
			fmt.Fprintf(&code, "if _, parameterError := loomhttp.DecodeJSONField(parameterForm, %q, &%s.%s, %t, %s); parameterError != nil {\nreturn parameterError\n}\n", field.HTTPName, target, field.FieldName, expr.AllowsNull(field.Attribute), formApplicationLiteral(field.Codec.Application))
		} else {
			fmt.Fprintf(&code, "if _, parameterError := loomhttp.DecodeFormValue(parameterForm, %q, &%s.%s); parameterError != nil {\nreturn parameterError\n}\n", field.HTTPName, target, field.FieldName)
		}
	}
	for _, field := range styled {
		if expr.AsUnion(field.Attribute.Type) != nil {
			element := formUnionElement(body, field)
			code.WriteString("{\n")
			matcher := "matchTextUnion"
			if field.Codec.Style == httpencoding.Content {
				matcher = "matchFormContent"
				code.WriteString(renderFormContentUnionMatcher(element))
			} else {
				code.WriteString("parameterPairs := parameterFormPairs\n")
				code.WriteString(renderTextUnionMatcher(element, "form"))
			}
			fmt.Fprintf(&code, "matched, present, _, parameterError := %s()\nif parameterError != nil {\nreturn parameterError\n}\nif present {\n", matcher)
			source := "matched"
			if !element.FieldPointer {
				source = "*matched"
			}
			fmt.Fprintf(&code, "%s.%s = %s\n", target, field.FieldName, source)
			if field.Required {
				fmt.Fprintf(&code, "} else {\nreturn loom.MissingFieldError(%q, \"body\")\n", field.HTTPName)
			}
			code.WriteString("}\n}\n")
			continue
		}
		key, physical := body.attribute.FindAttribute(field.AttributeName)
		if physical == nil {
			panic("missing styled form carrier " + field.AttributeName)
		}
		element := &Element{Namespace: field.Namespace}
		fmt.Fprintf(&code, "{\nparameterValue, parameterError := loomhttp.DecodePairsParam(parameterFormPairs, %q, %s, %s, %s)\nif parameterError != nil {\nreturn parameterError\n}\nif parameterValue.Present {\n", field.HTTPName, parameterCodecLiteral(field.Codec), parameterKindLiteral(field.Codec), parameterNamespaceLiteral(element))
		code.WriteString(renderParameterAssignment(field.Attribute, physical, body.context, "parameterValue", target+"."+field.FieldName, field.HTTPName, body.context.IsPrimitivePointer(key, body.attribute)))
		code.WriteString("}\n}\n")
	}
	code.WriteString("return nil\n")
	return code.String()
}

func renderMediaFormEncode(variant *MediaVariantData, context, value string, optional bool) string {
	if variant.form != nil && variant.form.Element != nil {
		return fmt.Sprintf("loomhttp.WithMediaFormCodec(%s, loomhttp.MediaFormCodec{Encode: func() (string, error) {\nbody := %s\n%s}})", context, value, renderFormMapEncode(variant.formBody, variant.form, "body"))
	}
	if variant.form != nil && len(variant.form.Branches) != 0 {
		return fmt.Sprintf("loomhttp.WithMediaFormCodec(%s, loomhttp.MediaFormCodec{Encode: func() (string, error) {\nbody := %s\nif body == nil {\nreturn \"\", fmt.Errorf(\"form media body is nil\")\n}\n%s}})", context, value, renderFormUnionEncode(variant.formBody, variant.form, optional))
	}
	return fmt.Sprintf("loomhttp.WithMediaFormCodec(%s, loomhttp.MediaFormCodec{Encode: func() (string, error) {\nbody := %s\nif body == nil {\nreturn \"\", fmt.Errorf(\"form media body is nil\")\n}\n%s}})", context, value, renderFormFieldsEncode(variant.formBody, variant.OrdinaryFormFields, variant.StyledFormFields, "body"))
}

func renderMediaFormDecode(variant *MediaVariantData, context, target string) string {
	if variant.form != nil && variant.form.Element != nil {
		return fmt.Sprintf("loomhttp.WithMediaFormCodec(%s, loomhttp.MediaFormCodec{Decode: func(parameterFormPairs loomhttp.RawPairs, parameterForm map[string][]string) error {\nvar body %s\n%s%s = body\nreturn nil\n}})", context, variant.formBody.Name, renderFormMapDecode(variant.formBody, variant.form, "body"), target)
	}
	if variant.form != nil && len(variant.form.Branches) != 0 {
		return renderFormUnionMediaDecode(variant, context, target)
	}
	return fmt.Sprintf("loomhttp.WithMediaFormCodec(%s, loomhttp.MediaFormCodec{Decode: func(parameterFormPairs loomhttp.RawPairs, parameterForm map[string][]string) error {\nbody := new(%s)\n%s = body\n%s}})", context, variant.formBody.Name, target, renderFormFieldsDecode(variant.formBody, variant.OrdinaryFormFields, variant.StyledFormFields, "body"))
}

func formApplicationLiteral(application httpencoding.Application) string {
	if application == httpencoding.Items {
		return "loomhttp.FieldItems"
	}
	return "loomhttp.FieldValue"
}

func renderJSONFormFieldEncode(body *TypeData, field *FormFieldData, target string) string {
	var code strings.Builder
	key, physical := body.attribute.FindAttribute(field.AttributeName)
	presence := ""
	if body.context.FieldPresence(expr.NewObjectFields(body.attribute), key, physical) != codegen.NativePresence {
		presence = target + "." + field.FieldName + ".Present()"
	} else if expr.AsUnion(physical.Type) != nil && expr.HasNativeJSONPresence(physical) {
		presence = target + "." + field.FieldName + ".Kind() != \"\""
	}
	if presence != "" {
		fmt.Fprintf(&code, "if %s {\n", presence)
	}
	if field.Required && field.Codec.Application == httpencoding.Items {
		fmt.Fprintf(&code, "if parameterPresent, parameterError := loomhttp.EncodeJSONField(parameterForm, %q, %s.%s, %s); parameterError != nil {\nreturn \"\", parameterError\n} else if !parameterPresent {\nreturn \"\", fmt.Errorf(%q)\n}\n", field.HTTPName, target, field.FieldName, formApplicationLiteral(field.Codec.Application), "required form field "+field.HTTPName+" has no encoded items")
	} else {
		fmt.Fprintf(&code, "if _, parameterError := loomhttp.EncodeJSONField(parameterForm, %q, %s.%s, %s); parameterError != nil {\nreturn \"\", parameterError\n}\n", field.HTTPName, target, field.FieldName, formApplicationLiteral(field.Codec.Application))
	}
	if presence != "" {
		if field.Required {
			fmt.Fprintf(&code, "} else {\nreturn \"\", fmt.Errorf(%q)\n", "required form field "+field.HTTPName+" is absent")
		}
		code.WriteString("}\n")
	}
	return code.String()
}

func formFieldsNeedValues(ordinary, styled []*FormFieldData) bool {
	if len(ordinary) > 0 {
		return true
	}
	for _, field := range styled {
		if field.Codec.Style == httpencoding.Content {
			return true
		}
	}
	return false
}
