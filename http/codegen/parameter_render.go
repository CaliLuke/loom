package codegen

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/httpencoding"
)

func parameterCodecLiteral(codec httpencoding.Codec) string {
	styles := map[httpencoding.Style]string{
		httpencoding.Simple: "ParamSimple", httpencoding.Matrix: "ParamMatrix", httpencoding.Label: "ParamLabel",
		httpencoding.Form: "ParamForm", httpencoding.SpaceDelimited: "ParamSpaceDelimited", httpencoding.PipeDelimited: "ParamPipeDelimited",
		httpencoding.DeepObject: "ParamDeepObject", httpencoding.CookieStyle: "ParamCookie",
	}
	style, ok := styles[codec.Style]
	if !ok {
		panic(fmt.Sprintf("no runtime parameter style %v", codec.Style))
	}
	return fmt.Sprintf("loomhttp.ParamCodec{Style: loomhttp.%s, Explode: %t}", style, codec.Explode)
}

func parameterKindLiteral(codec httpencoding.Codec) string {
	switch codec.Shape {
	case httpencoding.Scalar:
		return "loomhttp.ParamScalar"
	case httpencoding.Sequence:
		return "loomhttp.ParamList"
	case httpencoding.Object, httpencoding.ScalarMap:
		return "loomhttp.ParamPairs"
	default:
		panic(fmt.Sprintf("no runtime parameter shape %v", codec.Shape))
	}
}

func parameterNamespaceLiteral(element *Element) string {
	return fmt.Sprintf("loomhttp.ParamNamespace{Literals: %#v, Prefixes: %#v, OwnedLiterals: %#v}", element.Namespace.Literals, element.Namespace.Prefixes, element.Namespace.OwnedLiterals)
}

func renderStyledEncode(element *Element, source, location, header string) string {
	if expr.AsUnion(element.Type) != nil {
		return renderTextUnionEncode(element, source, location, header)
	}
	if element.fieldAttribute == nil {
		panic("styled parameter " + element.Name + " has no retained physical attribute") // bug
	}
	var code strings.Builder
	code.WriteString("{\n")
	value := source
	if element.FieldName != "" {
		value += "." + element.FieldName
	}
	code.WriteString(renderParameterValue(element.attribute, element.fieldAttribute, parameterServiceContext(element), value, "parameterValue", element.FieldPointer, ""))
	codec := parameterCodecLiteral(element.Codec)
	switch location {
	case "header", "response-header":
		fmt.Fprintf(&code, "parameterEncoded, parameterError := loomhttp.EncodeHeaderParam(%q, %s, parameterValue)\nif parameterError != nil {\nreturn parameterError\n}\nif parameterValue.Present {\n%s.Set(%q, parameterEncoded)\n}\n", element.HTTPName, codec, header, element.HTTPName)
	case "query":
		fmt.Fprintf(&code, "if parameterError := parameterBuilder.Add(%q, %s, parameterValue, %s); parameterError != nil {\nreturn parameterError\n}\n", element.HTTPName, codec, parameterNamespaceLiteral(element))
	case "cookie":
		fmt.Fprintf(&code, "if parameterError := loomhttp.AddRequestCookieParam(req, %q, %s, parameterValue, %s); parameterError != nil {\nreturn parameterError\n}\n", element.HTTPName, codec, parameterNamespaceLiteral(element))
	default:
		panic("unsupported styled encoder location " + location)
	}
	code.WriteString("}\n")
	return code.String()
}

// renderStyledDecode emits the body of a request-error closure shared by HTTP
// and JSON-RPC. Their outer templates select the protocol error response.
func renderStyledDecode(element *Element, location, headers string) string {
	if expr.AsUnion(element.Type) != nil {
		return renderTextUnionDecode(element, location, headers)
	}
	var code strings.Builder
	codec, kind := parameterCodecLiteral(element.Codec), parameterKindLiteral(element.Codec)
	switch location {
	case "header", "response-header":
		header := headers
		fmt.Fprintf(&code, "parameterValue, parameterError := loomhttp.DecodeHeaderParam(%q, %s.Values(%q), len(%s.Values(%q)) > 0, %s, %s)\n", element.HTTPName, header, element.HTTPName, header, element.HTTPName, codec, kind)
	case "query":
		fmt.Fprintf(&code, "parameterValue, parameterError := loomhttp.DecodePairsParam(parameterPairs, %q, %s, %s, %s)\n", element.HTTPName, codec, kind, parameterNamespaceLiteral(element))
	case "path":
		fmt.Fprintf(&code, "parameterValue, parameterError := loomhttp.DecodePathParam(%q, mux.RawVars(r)[%q], %s, %s)\n", element.HTTPName, element.HTTPName, codec, kind)
	case "cookie":
		fmt.Fprintf(&code, "parameterValue, parameterError := loomhttp.DecodeCookieParam(r.Cookies(), %q, %s, %s, %s)\n", element.HTTPName, codec, kind, parameterNamespaceLiteral(element))
	default:
		panic("unsupported styled decoder location " + location)
	}
	code.WriteString("if parameterError != nil {\nreturn loom.DecodePayloadError(parameterError.Error())\n}\nif parameterValue.Present {\nif parameterError := func() error {\n")
	code.WriteString(renderParameterAssignment(element.attribute, element.attribute, element.context, "parameterValue", element.VarName, element.HTTPName, element.Pointer))
	code.WriteString("return nil\n}(); parameterError != nil {\nreturn loomhttp.NormalizeParamDecodeError(parameterError)\n}\n")
	if element.Validate != "" {
		code.WriteString(renderParameterValidation(element, "parameterValue.Text"))
	}
	if element.Required {
		fmt.Fprintf(&code, "} else {\nreturn loom.MissingFieldError(%q, %q)\n", element.Name, location)
	} else if element.DefaultValue != nil {
		code.WriteString("} else {\n")
		code.WriteString(renderElementDefault(element))
		code.WriteString(renderParameterValidation(element, fmt.Sprintf("%q", element.DefaultValue)))
	}
	code.WriteString("}\nreturn nil\n")
	return code.String()
}

func hasStyledQuery(params []*ParamData) bool {
	for _, param := range params {
		if param.Styled {
			return true
		}
	}
	return false
}

func hasOrdinaryQuery(params []*ParamData) bool {
	for _, param := range params {
		if !param.Styled {
			return true
		}
	}
	return false
}

func renderParameterValidation(element *Element, text string) string {
	if element.Validate == "" {
		return ""
	}
	if element.IsTextUnmarshaler {
		return fmt.Sprintf("%s := %s\n%s\n", element.Locals.Raw, text, element.Validate)
	}
	return element.Validate + "\n"
}
