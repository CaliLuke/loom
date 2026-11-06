package codegen

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/CaliLuke/loom/internal/httpencoding"
)

// renderParameterPreflight rejects values whose grammar has no representation
// before BuildRequest allocates the request. Ordinary path builder APIs retain
// their historical non-fallible signatures.
func renderParameterPreflight(endpoint *transportir.Endpoint, payload *PayloadData, requestStruct, service, method string) string {
	var checks strings.Builder
	for _, param := range payload.Request.PathParams {
		if param.Styled || strings.Contains(endpoint.Routes[0].Path, "{*"+param.HTTPName+"}") {
			continue
		}
		if param.Codec.Shape != httpencoding.Sequence && param.Type != expr.String && param.Type != expr.Bytes {
			continue
		}
		checks.WriteString("{\n")
		checks.WriteString(renderParameterValue(param.attribute, parameterPhysicalAttribute(param.Element), param.context, parameterServiceValue(param.Element, "p"), "parameterValue", param.FieldPointer))
		fmt.Fprintf(&checks, "if _, parameterError := loomhttp.EncodePathParam(%q, %s, parameterValue); parameterError != nil {\nreturn parameterError\n}\n}\n", param.HTTPName, parameterCodecLiteral(param.Codec))
	}
	for _, param := range payload.Request.QueryParams {
		if param.Required && parameterEmptyHasNoKey(param.Codec) {
			checks.WriteString(renderParameterEmptyCheck(param.Element, "p"))
		}
	}
	for _, cookie := range payload.Request.Cookies {
		if cookie.Required && parameterEmptyHasNoKey(cookie.Codec) {
			checks.WriteString(renderParameterEmptyCheck(cookie.Element, "p"))
		}
	}
	if payload.Request.FormEncoded {
		renderFormPreflight(&checks, endpoint, payload)
	}
	if checks.Len() == 0 {
		return ""
	}
	var code strings.Builder
	code.WriteString("if parameterError := func() error {\n")
	if requestStruct != "" {
		fmt.Fprintf(&code, "parameterRequest, ok := v.(*%s)\nif !ok || parameterRequest == nil {\nreturn loomhttp.ErrInvalidType(%q, %q, %q, v)\n}\np := parameterRequest.Payload\n", requestStruct, service, method, "*"+requestStruct)
	} else {
		fmt.Fprintf(&code, "p, ok := v.(%s)\nif !ok {\nreturn loomhttp.ErrInvalidType(%q, %q, %q, v)\n}\n", payload.Ref, service, method, payload.Ref)
	}
	if expr.IsObject(endpoint.Request.Payload.Type) {
		fmt.Fprintf(&code, "if p == nil {\nreturn fmt.Errorf(\"%%s\", %q)\n}\n", "parameter payload is absent")
	}
	code.WriteString(checks.String())
	fmt.Fprintf(&code, "return nil\n}(); parameterError != nil {\nreturn nil, loomhttp.ErrEncodingError(%q, %q, parameterError)\n}\n", service, method)
	return code.String()
}

func parameterEmptyHasNoKey(codec httpencoding.Codec) bool {
	if codec.Shape == httpencoding.Scalar || codec.Style == httpencoding.Root || codec.Style == httpencoding.JSON {
		return false
	}
	return codec.Style == httpencoding.DeepObject || codec.Explode && (codec.Style == httpencoding.Form || codec.Style == httpencoding.CookieStyle)
}

func parameterServiceValue(element *Element, source string) string {
	if element.FieldName != "" {
		return source + "." + element.FieldName
	}
	return source
}

func renderParameterEmptyCheck(element *Element, source string) string {
	var code strings.Builder
	code.WriteString("{\n")
	code.WriteString(renderParameterValue(element.attribute, parameterPhysicalAttribute(element), element.context, parameterServiceValue(element, source), "parameterValue", element.FieldPointer))
	code.WriteString(renderParameterKeyCheck(element.Codec, element.HTTPName, ""))
	code.WriteString("}\n")
	return code.String()
}

func renderParameterKeyCheck(codec httpencoding.Codec, name, returnPrefix string) string {
	length := "len(parameterValue.Pairs)"
	if codec.Shape == httpencoding.Sequence {
		length = "len(parameterValue.Items)"
	}
	return fmt.Sprintf("if %s == 0 {\nreturn %sfmt.Errorf(\"%%s\", %q)\n}\n", length, returnPrefix, "parameter "+name+": required value produces no wire key")
}

func renderFormPreflight(checks *strings.Builder, endpoint *transportir.Endpoint, payload *PayloadData) {
	selected := selectBodyField(endpoint.Request.Payload, endpoint.Request.Body)
	source := "p"
	if selected.Name != "" {
		source += "." + selected.GoName
	}
	var formChecks strings.Builder
	fields := append(append([]*FormFieldData{}, payload.Request.StyledFormFields...), payload.Request.OrdinaryFormFields...)
	for _, field := range fields {
		if !field.Required || !parameterEmptyHasNoKey(field.Codec) {
			continue
		}
		key, physical := selected.Attribute.FindAttribute(field.AttributeName)
		if physical == nil {
			panic("missing form parameter source " + field.AttributeName)
		}
		element := &Element{attribute: field.Attribute, context: serviceContext("", codegen.NewNameScope()), HTTPName: field.HTTPName, Codec: field.Codec, AttributeData: &AttributeData{
			Type: field.Attribute.Type, FieldType: physical.Type,
			FieldName:    codegen.GoifyAtt(physical, expr.AttributeName(key), true),
			FieldPointer: selected.Attribute.IsPrimitivePointer(key, true),
		}}
		formChecks.WriteString(renderParameterEmptyCheck(element, source))
	}
	if formChecks.Len() > 0 {
		if selected.Name != "" && payload.Request.OptionalObjectBody {
			fmt.Fprintf(checks, "if %s != nil {\n%s}\n", source, formChecks.String())
		} else {
			checks.WriteString(formChecks.String())
		}
	}
}
