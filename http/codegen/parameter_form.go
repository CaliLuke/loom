package codegen

import (
	"fmt"
	"strings"
)

func renderStyledFormEncode(request *RequestData) string {
	return "{\nparameterEncoded, parameterError := func() (string, error) {\n" + renderFormFieldsEncode(request.ClientBody, request.OrdinaryFormFields, request.StyledFormFields) + "}()\nif parameterError != nil {\nreturn parameterError\n}\nif parameterError := loomhttp.SetRawFormRequest(req, parameterEncoded); parameterError != nil {\nreturn parameterError\n}\n}\n"
}

func renderFormFieldsEncode(body *TypeData, ordinary, styled []*FormFieldData) string {
	var code strings.Builder
	code.WriteString("parameterForm := make(map[string][]string)\nvar parameterBuilder loomhttp.RawPairsBuilder\n")
	for _, field := range ordinary {
		if field.Required && parameterEmptyHasNoKey(field.Codec) {
			key, physical := body.attribute.FindAttribute(field.AttributeName)
			code.WriteString("{\n")
			code.WriteString(renderParameterValue(field.Attribute, physical, body.context, "body."+field.FieldName, "parameterValue", body.context.IsPrimitivePointer(key, body.attribute)))
			code.WriteString(renderParameterKeyCheck(field.Codec, field.HTTPName, "\"\", "))
			code.WriteString("}\n")
		}
		fmt.Fprintf(&code, "if _, parameterError := loomhttp.EncodeFormValue(parameterForm, %q, body.%s); parameterError != nil {\nreturn \"\", parameterError\n}\n", field.HTTPName, field.FieldName)
	}
	for _, field := range styled {
		key, physical := body.attribute.FindAttribute(field.AttributeName)
		if physical == nil {
			panic("missing styled form carrier " + field.AttributeName)
		}
		code.WriteString("{\n")
		code.WriteString(renderParameterValue(field.Attribute, physical, body.context, "body."+field.FieldName, "parameterValue", body.context.IsPrimitivePointer(key, body.attribute)))
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
	code := renderFormFieldsDecode(request.ServerBody, request.OrdinaryFormFields, request.StyledFormFields)
	if len(request.OrdinaryFormFields) > 0 {
		code = "parameterForm := r.PostForm\n" + code
	}
	return code
}

func renderFormFieldsDecode(body *TypeData, ordinary, styled []*FormFieldData) string {
	var code strings.Builder
	for _, field := range ordinary {
		fmt.Fprintf(&code, "if _, parameterError := loomhttp.DecodeFormValue(parameterForm, %q, &body.%s); parameterError != nil {\nreturn parameterError\n}\n", field.HTTPName, field.FieldName)
	}
	for _, field := range styled {
		key, physical := body.attribute.FindAttribute(field.AttributeName)
		if physical == nil {
			panic("missing styled form carrier " + field.AttributeName)
		}
		element := &Element{Namespace: field.Namespace}
		fmt.Fprintf(&code, "{\nparameterValue, parameterError := loomhttp.DecodePairsParam(parameterFormPairs, %q, %s, %s, %s)\nif parameterError != nil {\nreturn parameterError\n}\nif parameterValue.Present {\n", field.HTTPName, parameterCodecLiteral(field.Codec), parameterKindLiteral(field.Codec), parameterNamespaceLiteral(element))
		code.WriteString(renderParameterAssignment(field.Attribute, physical, body.context, "parameterValue", "body."+field.FieldName, field.HTTPName, body.context.IsPrimitivePointer(key, body.attribute)))
		code.WriteString("}\n}\n")
	}
	code.WriteString("return nil\n")
	return code.String()
}

func renderMediaFormEncode(variant *MediaVariantData, context, value string) string {
	return fmt.Sprintf("loomhttp.WithMediaFormCodec(%s, loomhttp.MediaFormCodec{Encode: func() (string, error) {\nbody := %s\nif body == nil {\nreturn \"\", fmt.Errorf(\"form media body is nil\")\n}\n%s}})", context, value, renderFormFieldsEncode(variant.formBody, variant.OrdinaryFormFields, variant.StyledFormFields))
}

func renderMediaFormDecode(variant *MediaVariantData, context, target string) string {
	return fmt.Sprintf("loomhttp.WithMediaFormCodec(%s, loomhttp.MediaFormCodec{Decode: func(parameterFormPairs loomhttp.RawPairs, parameterForm map[string][]string) error {\nbody := new(%s)\n%s = body\n%s}})", context, variant.formBody.Name, target, renderFormFieldsDecode(variant.formBody, variant.OrdinaryFormFields, variant.StyledFormFields))
}
